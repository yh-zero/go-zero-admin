package ai

import (
	"context"
	"go-zero-admin/pkg/xerr"

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
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
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
