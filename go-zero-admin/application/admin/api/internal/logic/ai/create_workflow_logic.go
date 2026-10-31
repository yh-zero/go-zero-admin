package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

type CreateWorkflowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateWorkflowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateWorkflowLogic {
	return &CreateWorkflowLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *CreateWorkflowLogic) CreateWorkflow(req *types.AiCreateWorkflowReq) (*types.AiWorkflowResp, error) {
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.CreateWorkflow(l.ctx, &aipb.CreateWorkflowRequest{Actor: actor, Name: req.Name, Definition: req.Definition, IntervalSeconds: req.IntervalSeconds})
	if err != nil {
		return nil, err
	}
	return &types.AiWorkflowResp{Workflow: mapAiWorkflow(out.Workflow)}, nil
}
