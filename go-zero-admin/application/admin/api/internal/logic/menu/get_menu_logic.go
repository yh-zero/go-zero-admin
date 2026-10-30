package menu

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMenuLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuLogic {
	return &GetMenuLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetMenuLogic) GetMenu(req *types.GetMenuReq) (resp *types.GetMenuResp, err error) {
	rpcResp, err := l.svcCtx.MenuRpc.GetMenu(l.ctx, &adminpb.GetMenuRequest{MenuId: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.GetMenuResp{Menu: mapMenuInfo(rpcResp.Menu)}, nil
}
