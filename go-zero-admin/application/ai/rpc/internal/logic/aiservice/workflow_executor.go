package aiservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/xerr"
)

type workflowStep struct {
	Name     string          `json:"name"`
	Tool     string          `json:"tool"`
	Input    json.RawMessage `json:"input"`
	Approval bool            `json:"approval"`
}
type workflowDefinition struct {
	Steps []workflowStep `json:"steps"`
}

func parseWorkflow(raw string) (*workflowDefinition, error) {
	if len(raw) == 0 || len(raw) > 64<<10 {
		return nil, xerr.NewErrCodeMsg(400, "工作流定义须为 1..65536 字节")
	}
	var definition workflowDefinition
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if !json.Valid([]byte(raw)) || decoder.Decode(&definition) != nil || len(definition.Steps) == 0 || len(definition.Steps) > 32 {
		return nil, xerr.NewErrCodeMsg(400, "工作流须包含 1..32 个有效步骤")
	}
	for _, step := range definition.Steps {
		if strings.TrimSpace(step.Name) == "" || utf8.RuneCountInString(step.Name) > 128 {
			return nil, xerr.NewErrCodeMsg(400, "步骤名称不能为空且最多 128 字符")
		}
		if _, ok := agenttools.Get(step.Tool); !ok {
			return nil, xerr.NewErrCodeMsg(400, "工作流包含未注册工具")
		}
		var input map[string]json.RawMessage
		if json.Unmarshal(step.Input, &input) != nil || input == nil {
			return nil, xerr.NewErrCodeMsg(400, "步骤 input 必须为 JSON 对象")
		}
	}
	return &definition, nil
}

// executeWorkflow 只执行已领取的 running 任务；审批步执行前持久化断点。
func executeWorkflow(ctx context.Context, svcCtx *svc.ServiceContext, task *model.TaskRow, approved bool) (string, error) {
	definition, err := parseWorkflow(task.WorkflowSnapshot)
	if err != nil {
		return failWorkflow(ctx, svcCtx, task, "工作流快照无效")
	}
	if task.NextStep < 0 || task.NextStep >= int64(len(definition.Steps)) {
		return failWorkflow(ctx, svcCtx, task, "工作流断点无效")
	}
	for task.NextStep < int64(len(definition.Steps)) {
		if ctx.Err() != nil {
			return failWorkflow(ctx, svcCtx, task, "工作流执行已取消或超时")
		}
		step := definition.Steps[task.NextStep]
		if step.Approval && !approved {
			if err := svcCtx.AiTaskModel.Advance(ctx, task.Id, task.CreatedBy, task.NextStep, task.NextStep, "pending_approval", task.Result, ""); err != nil {
				return "", aiModelError(err)
			}
			return "pending_approval", nil
		}
		approved = false
		tool, ok := agenttools.Get(step.Tool)
		if !ok {
			return failWorkflow(ctx, svcCtx, task, "步骤工具不可用")
		}
		output, err := tool.Execute(agenttools.WithActor(ctx, task.CreatedBy), step.Input)
		if err != nil {
			return failWorkflow(ctx, svcCtx, task, "步骤工具执行失败")
		}
		result := task.Result
		if result != "" {
			result += "\n"
		}
		result += "[" + strconv.Itoa(int(task.NextStep)+1) + "/" + strconv.Itoa(len(definition.Steps)) + "] " + step.Name + ": " + output
		if len(result) > 60000 {
			return failWorkflow(ctx, svcCtx, task, "工作流结果超过大小限制")
		}
		status := "running"
		if task.NextStep+1 == int64(len(definition.Steps)) {
			status = "succeeded"
		}
		if err := svcCtx.AiTaskModel.Advance(ctx, task.Id, task.CreatedBy, task.NextStep, task.NextStep+1, status, result, ""); err != nil {
			return "", aiModelError(err)
		}
		task.NextStep++
		task.Result = result
		if status == "succeeded" {
			return status, nil
		}
	}
	return "", xerr.NewErrCodeMsg(500, "工作流执行状态异常")
}

func failWorkflow(ctx context.Context, svcCtx *svc.ServiceContext, task *model.TaskRow, message string) (string, error) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := svcCtx.AiTaskModel.Advance(cleanupCtx, task.Id, task.CreatedBy, task.NextStep, task.NextStep, "failed", task.Result, message); err != nil {
		return "", aiModelError(err)
	}
	return "failed", nil
}

func requireAiActor(actor *aipb.Actor) error {
	if actor.GetUserId() <= 0 {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	return nil
}

func aiModelError(err error) error {
	if errors.Is(err, model.ErrWorkflowDisabled) {
		return xerr.NewErrCodeMsg(400, "工作流已停用")
	}
	if errors.Is(err, model.ErrNotFound) {
		return xerr.NewErrCodeMsg(xerr.NOT_FOUND, "资源不存在")
	}
	if errors.Is(err, model.ErrTaskState) {
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "任务状态已变化，不能重复操作")
	}
	if errors.Is(err, model.ErrKbDocState) {
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "文档状态已变化，仅失败或待摄取的文档可重试")
	}
	return xerr.NewErrCode(xerr.DB_ERROR)
}

func aiPagination(page *aipb.PageRequest) (int64, int64, error) {
	n, size := page.GetPage(), page.GetPageSize()
	if n == 0 {
		n = 1
	}
	if size == 0 {
		size = 20
	}
	if n < 1 || n > 1_000_000 || size < 1 || size > 100 {
		return 0, 0, xerr.NewErrCodeMsg(400, "分页参数不合法")
	}
	return n, size, nil
}

func conversationProto(row *model.ConversationRow) *aipb.ConversationInfo {
	return &aipb.ConversationInfo{Id: row.Id, Title: row.Title, UserId: row.UserId, MessageCount: row.MessageCount, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func messageProto(row *model.MessageRow) *aipb.ChatMessage {
	status := row.Status
	if status == "" {
		status = "succeeded"
	}
	calls := "[]"
	message := ""
	if strings.HasPrefix(strings.TrimSpace(row.ToolCalls), "[") && json.Valid([]byte(row.ToolCalls)) {
		calls = row.ToolCalls
	} else {
		var record callRecord
		if json.Unmarshal([]byte(row.ToolCalls), &record) == nil && record.Version == 1 {
			if row.Role == "assistant" && record.Calls != nil {
				data, _ := json.Marshal(record.Calls)
				calls = string(data)
			}
			message = record.Error
		}
	}
	return &aipb.ChatMessage{Id: row.Id, Role: row.Role, Content: row.Content, ToolCalls: calls, Tokens: row.Tokens, CreatedAt: row.CreatedAt, Status: status, Error: message}
}
func workflowProto(row *model.WorkflowRow) *aipb.WorkflowInfo {
	return &aipb.WorkflowInfo{Id: row.Id, Name: row.Name, Definition: row.Definition, IntervalSeconds: row.IntervalSeconds, Enabled: row.Enabled, NextRunAt: row.NextRunAt, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
func taskProto(row *model.TaskRow) *aipb.AgentTaskInfo {
	return &aipb.AgentTaskInfo{Id: row.Id, WorkflowId: row.WorkflowId, ConversationId: row.ConversationId, Status: row.Status, Input: row.Input, Result: row.Result, Error: row.Error, ApprovedBy: row.ApprovedBy, CreatedAt: row.CreatedAt, CompletedAt: row.CompletedAt, NextStep: row.NextStep}
}
