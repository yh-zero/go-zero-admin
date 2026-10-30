package menu

import (
	"context"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyMenusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMyMenusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyMenusLogic {
	return &GetMyMenusLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetMyMenusLogic) GetMyMenus(req *types.GetMyMenusReq) (resp *types.GetMyMenusResp, err error) {
	userID := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	actor := &adminpb.Actor{UserId: userID}
	rpcResp, err := l.svcCtx.MenuRpc.GetMyMenus(l.ctx, &adminpb.GetMyMenusRequest{Actor: actor})
	if err != nil {
		return nil, err
	}
	list := make([]*types.MenuInfo, 0, len(rpcResp.List))
	for _, m := range rpcResp.List {
		list = append(list, mapMenuInfo(m))
	}
	return &types.GetMyMenusResp{List: list}, nil
}
