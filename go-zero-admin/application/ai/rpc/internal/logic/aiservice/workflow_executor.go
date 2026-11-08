package aiservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
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
	Id       string          `json:"id,omitempty"`
	Name     string          `json:"name"`
	Tool     string          `json:"tool"`
	Input    json.RawMessage `json:"input"`
	Approval bool            `json:"approval"`
}
type workflowDefinition struct {
	Version int            `json:"version,omitempty"`
	Steps   []workflowStep `json:"steps"`
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
	if definition.Version != 0 && definition.Version != 2 {
		return nil, xerr.NewErrCodeMsg(400, "工作流版本不支持")
	}
	prior := make(map[string]bool)
	for index := range definition.Steps {
		step := &definition.Steps[index]
		if definition.Version == 0 && step.Id == "" {
			step.Id = "legacy_" + strconv.Itoa(index+1)
		}
		if !workflowStepID.MatchString(step.Id) || prior[step.Id] {
			return nil, xerr.NewErrCodeMsg(400, "步骤 ID 无效或重复")
		}
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
		value, err := readWorkflowJSON(string(step.Input))
		if err != nil {
			return nil, xerr.NewErrCodeMsg(400, "步骤参数无效")
		}
		if err := checkWorkflowReferences(value, prior, 0); err != nil {
			return nil, xerr.NewErrCodeMsg(400, err.Error())
		}
		prior[step.Id] = true
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
	records, err := model.ReadWorkflowStepRecords(task.StepRecords)
	if err != nil || len(records) < int(task.NextStep) {
		return failWorkflow(ctx, svcCtx, task, "历史任务缺少有效步骤记录，不能安全续跑")
	}
	for task.NextStep < int64(len(definition.Steps)) {
		if ctx.Err() != nil {
			return failWorkflow(ctx, svcCtx, task, "工作流执行已取消或超时")
		}
		step := definition.Steps[task.NextStep]
		var input json.RawMessage
		if approved && int64(len(records)) > task.NextStep && records[task.NextStep].Status == "pending_approval" {
			// 本次审批展示的输入已持久化，不受工作流更新或前端参数影响。
			input = json.RawMessage(records[task.NextStep].Input)
		} else {
			input, err = resolveWorkflowInput(definition, task.NextStep, task.Input, records)
			if err != nil {
				return failWorkflow(ctx, svcCtx, task, "步骤参数解析失败："+err.Error())
			}
		}
		record := model.WorkflowStepRecord{Index: task.NextStep, Id: step.Id, Name: step.Name, Tool: step.Tool, Input: string(input), Status: "running", StartedAt: time.Now().UnixMilli()}
		if int64(len(records)) == task.NextStep {
			records = append(records, record)
		} else if int64(len(records)) == task.NextStep+1 {
			records[task.NextStep] = record
		} else {
			return failWorkflow(ctx, svcCtx, task, "步骤记录与断点不一致")
		}
		if step.Approval && !approved {
			records[task.NextStep].Status = "pending_approval"
			records[task.NextStep].StartedAt = 0
			if err := persistWorkflowStep(ctx, svcCtx, task, task.NextStep, "pending_approval", task.Result, "", records); err != nil {
				return "", aiModelError(err)
			}
			return "pending_approval", nil
		}
		approved = false
		if err := persistWorkflowStep(ctx, svcCtx, task, task.NextStep, "running", task.Result, "", records); err != nil {
			return "", aiModelError(err)
		}
		tool, ok := agenttools.Get(step.Tool)
		if !ok {
			return failWorkflow(ctx, svcCtx, task, "步骤工具不可用")
		}
		if ctx.Err() != nil {
			return failWorkflow(ctx, svcCtx, task, "执行已取消或超时，当前工具结果可能未知")
		}
		meteredCtx := svc.WithUsageSource(ctx, task.CreatedBy, "analysis", task.RequestId, task.Id, task.NextStep)
		output, err := tool.Execute(agenttools.WithActor(meteredCtx, task.CreatedBy), input)
		if err != nil {
			var safe *agenttools.ExecutionError
			if errors.As(err, &safe) {
				return failWorkflow(ctx, svcCtx, task, safe.Message, workflowStepUsage(safe.ModelUsage()))
			}
			return failWorkflow(ctx, svcCtx, task, "步骤工具执行失败")
		}
		var usage *model.WorkflowStepUsage
		if agenttools.IsWorkflowOnly(tool) {
			var analysis agenttools.KbAnalysisResult
			if json.Unmarshal([]byte(output), &analysis) == nil && analysis.Version == 1 && analysis.Type == "kb_analysis" && (analysis.Status == "completed" || analysis.Status == "truncated") {
				usage = workflowStepUsage(&analysis.Usage)
			}
		}
		if ctx.Err() != nil {
			return failWorkflow(ctx, svcCtx, task, "执行已取消或超时，当前工具结果可能未知", usage)
		}
		result := task.Result
		if result != "" {
			result += "\n"
		}
		result += "[" + strconv.Itoa(int(task.NextStep)+1) + "/" + strconv.Itoa(len(definition.Steps)) + "] " + step.Name + ": " + output
		if len(result) > 60000 {
			return failWorkflow(ctx, svcCtx, task, "工作流结果超过大小限制", usage)
		}
		status := "running"
		if task.NextStep+1 == int64(len(definition.Steps)) {
			status = "succeeded"
		}
		records[task.NextStep].Status = "succeeded"
		records[task.NextStep].Output = output
		records[task.NextStep].Usage = usage
		records[task.NextStep].CompletedAt = time.Now().UnixMilli()
		if err := persistWorkflowStep(ctx, svcCtx, task, task.NextStep+1, status, result, "", records); err != nil {
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

func failWorkflow(ctx context.Context, svcCtx *svc.ServiceContext, task *model.TaskRow, message string, usage ...*model.WorkflowStepUsage) (string, error) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	records, _ := model.ReadWorkflowStepRecords(task.StepRecords)
	if task.NextStep < int64(len(records)) {
		r := &records[task.NextStep]
		r.Status = "failed"
		r.Error = message
		r.CompletedAt = time.Now().UnixMilli()
		if len(usage) > 0 {
			r.Usage = usage[0]
		}
	}
	if err := persistWorkflowStep(cleanupCtx, svcCtx, task, task.NextStep, "failed", task.Result, message, records); err != nil {
		if errors.Is(err, model.ErrTaskState) && task.NextStep < int64(len(records)) && records[task.NextStep].Usage != nil {
			late := *records[task.NextStep].Usage
			late.Complete = false
			status, lateErr := svcCtx.AiTaskModel.StoreTerminalStepUsage(cleanupCtx, task.Id, task.CreatedBy, records[task.NextStep], late)
			if lateErr == nil || errors.Is(lateErr, model.ErrTaskCommitted) {
				if lateErr != nil {
					logx.WithContext(cleanupCtx).Errorf("workflow task %d late usage committed; cache invalidation failed", task.Id)
				}
				return status, nil
			}
		}
		return "", aiModelError(err)
	}
	return "failed", nil
}

func workflowStepUsage(usage *agenttools.KbAnalysisUsage) *model.WorkflowStepUsage {
	if usage == nil {
		return nil
	}
	return &model.WorkflowStepUsage{Scope: usage.Scope, PromptTokens: int64(usage.PromptTokens), CompletionTokens: int64(usage.CompletionTokens), TotalTokens: int64(usage.TotalTokens), Complete: usage.Complete}
}

func persistWorkflowStep(ctx context.Context, svcCtx *svc.ServiceContext, task *model.TaskRow, next int64, status, result, message string, records []model.WorkflowStepRecord) error {
	if err := svcCtx.AiTaskModel.AdvanceRecorded(ctx, task.Id, task.CreatedBy, task.NextStep, next, status, result, message, records); err != nil {
		if !errors.Is(err, model.ErrTaskCommitted) {
			return err
		}
		logx.WithContext(ctx).Errorf("workflow task %d progress committed; cache invalidation failed", task.Id)
	}
	raw, _ := json.Marshal(records)
	task.StepRecords = string(raw)
	return nil
}

func requireAiActor(actor *aipb.Actor) error {
	if actor.GetUserId() <= 0 {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	return nil
}

func aiModelError(err error) error {
	if errors.Is(err, model.ErrTaskCommitted) {
		return xerr.NewErrCodeMsg(503, "任务状态已提交，缓存刷新失败，请刷新详情核对")
	}
	if errors.Is(err, model.ErrTaskRequestConflict) {
		return xerr.NewErrCodeMsg(400, "同一 request_id 的工作流或输入不同")
	}
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
	return &aipb.ConversationInfo{Id: row.Id, Title: row.Title, UserId: row.UserId, MessageCount: row.MessageCount, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Pinned: row.Pinned}
}
func messageProto(row *model.MessageRow) *aipb.ChatMessage {
	status := row.Status
	if status == "" {
		status = "succeeded"
	}
	calls := "[]"
	message := ""
	sources := make([]*aipb.KbSearchHit, 0)
	feedback, feedbackNote := "", ""
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
			if row.Role == "assistant" {
				for _, hit := range record.Sources {
					sources = append(sources, &aipb.KbSearchHit{ChunkId: hit.ChunkID, KbId: hit.KbID, DocId: hit.DocID, DocTitle: hit.DocTitle, Snippet: hit.Snippet, Score: hit.Score})
				}
				feedback, feedbackNote = record.Feedback, record.FeedbackNote
			}
		}
	}
	return &aipb.ChatMessage{Id: row.Id, Role: row.Role, Content: row.Content, ToolCalls: calls, Tokens: row.Tokens, CreatedAt: row.CreatedAt, Status: status, Error: message, Sources: sources, Feedback: feedback, FeedbackNote: feedbackNote}
}
func workflowProto(row *model.WorkflowRow) *aipb.WorkflowInfo {
	return &aipb.WorkflowInfo{Id: row.Id, Name: row.Name, Definition: row.Definition, IntervalSeconds: row.IntervalSeconds, Enabled: row.Enabled, NextRunAt: row.NextRunAt, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, StepCount: row.StepCount, Revision: row.RevisionSeq}
}
func taskProto(row *model.TaskRow) *aipb.AgentTaskInfo {
	records, _ := model.ReadWorkflowStepRecords(row.StepRecords)
	steps := make([]*aipb.WorkflowStepInfo, 0, len(records))
	for _, r := range records {
		step := &aipb.WorkflowStepInfo{Index: r.Index, Id: r.Id, Name: r.Name, Tool: r.Tool, Input: r.Input, Output: r.Output, Status: r.Status, Error: r.Error, StartedAt: r.StartedAt, CompletedAt: r.CompletedAt}
		if r.Usage != nil {
			step.Usage = &aipb.WorkflowStepUsage{Scope: r.Usage.Scope, PromptTokens: r.Usage.PromptTokens, CompletionTokens: r.Usage.CompletionTokens, TotalTokens: r.Usage.TotalTokens, Complete: r.Usage.Complete}
		}
		steps = append(steps, step)
	}
	title, workflowName := row.Title, row.WorkflowName
	if title == "" {
		title = "历史任务 #" + strconv.FormatInt(row.Id, 10)
	}
	if workflowName == "" && row.WorkflowId > 0 {
		workflowName = "历史流程 #" + strconv.FormatInt(row.WorkflowId, 10)
	}
	return &aipb.AgentTaskInfo{Id: row.Id, WorkflowId: row.WorkflowId, ConversationId: row.ConversationId, Status: row.Status, Input: row.Input, Result: row.Result, Error: row.Error, ApprovedBy: row.ApprovedBy, CreatedAt: row.CreatedAt, CompletedAt: row.CompletedAt, NextStep: row.NextStep, Steps: steps, UpdatedAt: row.UpdatedAt, RequestId: row.RequestId, Title: title, WorkflowName: workflowName, InputSummary: row.InputSummary, Definition: row.WorkflowSnapshot}
}
