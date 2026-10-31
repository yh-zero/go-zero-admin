package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

type RunWorkflowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRunWorkflowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RunWorkflowLogic {
	return &RunWorkflowLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *RunWorkflowLogic) RunWorkflow(req *types.AiRunWorkflowReq) (*types.AiRunWorkflowResp, error) {
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.RunWorkflow(l.ctx, &aipb.RunWorkflowRequest{Actor: actor, WorkflowId: req.Id, Input: req.Input})
	if err != nil {
		return nil, err
	}
	return &types.AiRunWorkflowResp{TaskId: out.TaskId, Status: out.Status}, nil
}
