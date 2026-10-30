package aiservicelogic

import (
	"context"
	"encoding/json"
	"strings"

	"go-zero-admin/pkg/agenttools"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"go-zero-admin/pkg/xerr"
)

type RunWorkflowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRunWorkflowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RunWorkflowLogic {
	return &RunWorkflowLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// workflowDefinition DSL 结构
type workflowDefinition struct {
	Steps []struct {
		Name     string          `json:"name"`
		Tool     string          `json:"tool"`
		Input    json.RawMessage `json:"input"`
		Approval bool            `json:"approval"`
	} `json:"steps"`
}

func (l *RunWorkflowLogic) RunWorkflow(in *aipb.RunWorkflowRequest) (*aipb.RunWorkflowResponse, error) {
	// 1. 读工作流定义
	var def string
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &def,
		"SELECT CAST(definition AS CHAR) FROM ai_workflow WHERE id=? AND deleted_at IS NULL AND enabled=1", in.WorkflowId)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(404, "工作流不存在或已停用")
	}

	var wd workflowDefinition
	if err := json.Unmarshal([]byte(def), &wd); err != nil {
		return nil, xerr.NewErrCodeMsg(400, "工作流定义解析失败："+err.Error())
	}

	// 2. 创建任务记录
	result, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"INSERT INTO ai_task (workflow_id, status, input) VALUES (?,?,?)", in.WorkflowId, "running", in.Input)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	taskID, _ := result.LastInsertId()

	// 3. 顺序执行步骤
	var results []string
	needApproval := false
	for i, step := range wd.Steps {
		tool, ok := agenttools.Get(step.Tool)
		if !ok {
			results = append(results, step.Name+": 未注册工具 "+step.Tool)
			continue
		}
		out, err := tool.Execute(l.ctx, step.Input)
		if err != nil {
			results = append(results, step.Name+": 执行错误 "+err.Error())
		} else {
			results = append(results, step.Name+": "+out)
		}
		// 审批节点：中断等待人工
		if step.Approval && i < len(wd.Steps)-1 {
			needApproval = true
			_, _ = l.svcCtx.DB.ExecCtx(l.ctx,
				"UPDATE ai_task SET status='pending_approval', result=? WHERE id=?", strings.Join(results, "\n"), taskID)
			break
		}
	}

	// 4. 完成（无审批中断）
	if !needApproval {
		_, _ = l.svcCtx.DB.ExecCtx(l.ctx,
			"UPDATE ai_task SET status='succeeded', result=?, completed_at=NOW() WHERE id=?",
			strings.Join(results, "\n"), taskID)
	}
	return &aipb.RunWorkflowResponse{TaskId: taskID, Status: map[bool]string{true: "pending_approval", false: "succeeded"}[needApproval]}, nil
}
