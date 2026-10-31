package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

type UpdateWorkflowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateWorkflowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateWorkflowLogic {
	return &UpdateWorkflowLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *UpdateWorkflowLogic) UpdateWorkflow(req *types.AiUpdateWorkflowReq) (*types.EmptyResp, error) {
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	_, err = l.svcCtx.AiRpc.UpdateWorkflow(l.ctx, &aipb.UpdateWorkflowRequest{Actor: actor, WorkflowId: req.Id, Name: req.Name, Definition: req.Definition, IntervalSeconds: req.IntervalSeconds, Enabled: req.Enabled})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
