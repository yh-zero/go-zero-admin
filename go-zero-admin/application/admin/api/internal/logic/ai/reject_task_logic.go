package ai

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

type RejectTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRejectTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejectTaskLogic {
	return &RejectTaskLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *RejectTaskLogic) RejectTask(req *types.AiRejectTaskReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	_, err = l.svcCtx.AiRpc.RejectTask(l.ctx, &aipb.RejectTaskRequest{Actor: actor, TaskId: req.Id, Reason: req.Reason})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
