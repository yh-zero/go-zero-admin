package aiservicelogic

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
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
func (l *RunWorkflowLogic) RunWorkflow(in *aipb.RunWorkflowRequest) (*aipb.RunWorkflowResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	workflow, err := l.svcCtx.AiWorkflowModel.FindOwned(l.ctx, in.WorkflowId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	if workflow.Enabled != 1 {
		return nil, xerr.NewErrCodeMsg(400, "工作流已停用")
	}
	if _, err := parseWorkflow(workflow.Definition); err != nil {
		return nil, err
	}
	input := in.Input
	if input == "" {
		input = "{}"
	}
	if len(input) > 16000 || !json.Valid([]byte(input)) {
		return nil, xerr.NewErrCodeMsg(400, "input 必须为最多 16000 字节的 JSON")
	}
	result, err := l.svcCtx.AiTaskModel.Insert(l.ctx, &model.AiTask{WorkflowId: sql.NullInt64{Int64: workflow.Id, Valid: true}, Status: "running", Input: sql.NullString{String: input, Valid: true}, CreatedBy: uint64(in.Actor.UserId), WorkflowSnapshot: sql.NullString{String: workflow.Definition, Valid: true}})
	if err != nil {
		return nil, aiModelError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, aiModelError(err)
	}
	task := &model.TaskRow{Id: id, WorkflowId: workflow.Id, Status: "running", Input: input, CreatedBy: in.Actor.UserId, WorkflowSnapshot: workflow.Definition}
	status, err := executeWorkflow(l.ctx, l.svcCtx, task, false)
	if err != nil {
		return nil, err
	}
	return &aipb.RunWorkflowResponse{TaskId: id, Status: status}, nil
}
