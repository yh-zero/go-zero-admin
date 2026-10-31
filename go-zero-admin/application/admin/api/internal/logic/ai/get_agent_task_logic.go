package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

type GetAgentTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAgentTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAgentTaskLogic {
	return &GetAgentTaskLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *GetAgentTaskLogic) GetAgentTask(req *types.AiIdReq) (*types.AiTaskResp, error) {
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.GetAgentTask(l.ctx, &aipb.GetAgentTaskRequest{Actor: actor, TaskId: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.AiTaskResp{Task: mapAiTask(out.Task)}, nil
}
