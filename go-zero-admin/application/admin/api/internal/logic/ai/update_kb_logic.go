package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"
)

type UpdateKbLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateKbLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateKbLogic {
	return &UpdateKbLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateKbLogic) UpdateKb(req *types.AiUpdateKbReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.AiRpc.UpdateKb(l.ctx, &aipb.UpdateKbRequest{Actor: actor, KbId: req.Id, Name: req.Name, Description: req.Description}); err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
