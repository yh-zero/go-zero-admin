package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

type GetWorkflowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetWorkflowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWorkflowLogic {
	return &GetWorkflowLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *GetWorkflowLogic) GetWorkflow(req *types.AiIdReq) (*types.AiWorkflowResp, error) {
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.GetWorkflow(l.ctx, &aipb.GetWorkflowRequest{Actor: actor, WorkflowId: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.AiWorkflowResp{Workflow: mapAiWorkflow(out.Workflow)}, nil
}
