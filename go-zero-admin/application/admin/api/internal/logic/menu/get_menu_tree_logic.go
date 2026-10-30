package menu

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMenuTreeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMenuTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuTreeLogic {
	return &GetMenuTreeLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetMenuTreeLogic) GetMenuTree(req *types.GetMenuTreeReq) (resp *types.GetMenuTreeResp, err error) {
	rpcResp, err := l.svcCtx.MenuRpc.GetMenuTree(l.ctx, &adminpb.GetMenuTreeRequest{Type: req.Type, Status: req.Status})
	if err != nil {
		return nil, err
	}
	list := make([]*types.MenuInfo, 0, len(rpcResp.List))
	for _, m := range rpcResp.List {
		list = append(list, mapMenuInfo(m))
	}
	return &types.GetMenuTreeResp{List: list}, nil
}

func mapMenuInfo(m *adminpb.MenuInfo) *types.MenuInfo {
	return &types.MenuInfo{Id: m.Id, ParentId: m.ParentId, Title: m.Title, Name: m.Name, Type: m.Type, Path: m.Path, Component: m.Component, PermCode: m.PermCode, Icon: m.Icon, Sort: m.Sort, Visible: m.Visible, Status: m.Status, CreatedAt: m.CreatedAt}
}
