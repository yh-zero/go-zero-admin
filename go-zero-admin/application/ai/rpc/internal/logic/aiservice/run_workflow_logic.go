package aiservicelogic

import (
	"context"
	"encoding/json"
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
func (l *RunWorkflowLogic) RunWorkflow(in *aipb.RunWorkflowRequest) (*aipb.RunWorkflowResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	input := in.Input
	if input == "" {
		input = "{}"
	}
	if len(input) > 16000 || !json.Valid([]byte(input)) {
		return nil, xerr.NewErrCodeMsg(400, "input 必须为最多 16000 字节的 JSON")
	}
	task, err := l.svcCtx.AiWorkflowModel.CreateTaskOwned(l.ctx, in.WorkflowId, in.Actor.UserId, input)
	if err != nil {
		return nil, aiModelError(err)
	}
	status, err := executeWorkflow(l.ctx, l.svcCtx, task, false)
	if err != nil {
		return nil, err
	}
	return &aipb.RunWorkflowResponse{TaskId: task.Id, Status: status}, nil
}
