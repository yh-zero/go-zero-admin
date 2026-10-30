package menu

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMenuListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMenuListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuListLogic {
	return &GetMenuListLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetMenuListLogic) GetMenuList(req *types.GetMenuListReq) (resp *types.GetMenuListResp, err error) {
	rpcResp, err := l.svcCtx.MenuRpc.GetMenuList(l.ctx, &adminpb.GetMenuListRequest{
		Page:     &adminpb.PageRequest{Page: req.Page, PageSize: req.PageSize},
		ParentId: req.ParentId, Type: req.Type,
	})
	if err != nil {
		return nil, err
	}
	list := make([]*types.MenuInfo, 0, len(rpcResp.List))
	for _, m := range rpcResp.List {
		list = append(list, mapMenuInfo(m))
	}
	return &types.GetMenuListResp{List: list, Total: rpcResp.Page.Total}, nil
}
