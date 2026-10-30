package menu

import (
	"context"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyPermCodesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMyPermCodesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyPermCodesLogic {
	return &GetMyPermCodesLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetMyPermCodesLogic) GetMyPermCodes(req *types.GetMyPermCodesReq) (resp *types.GetMyPermCodesResp, err error) {
	userID := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	actor := &adminpb.Actor{UserId: userID}
	rpcResp, err := l.svcCtx.MenuRpc.GetMyPermCodes(l.ctx, &adminpb.GetMyPermCodesRequest{Actor: actor})
	if err != nil {
		return nil, err
	}
	return &types.GetMyPermCodesResp{Codes: rpcResp.Codes}, nil
}
