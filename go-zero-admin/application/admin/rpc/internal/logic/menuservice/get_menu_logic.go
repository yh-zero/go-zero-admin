package menuservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuLogic {
	return &GetMenuLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetMenuLogic) GetMenu(in *adminpb.GetMenuRequest) (*adminpb.GetMenuResponse, error) {
	menu, err := l.svcCtx.SysMenuModel.FindOne(l.ctx, uint64(in.MenuId))
	if err != nil {
		if !errors.Is(err, model.ErrNotFound) {
			return nil, xerr.NewErrCode(xerr.DB_ERROR)
		}
		return nil, xerr.NewErrCode(xerr.MENU_NOT_FOUND)
	}
	if menu.DeletedAt.Valid {
		return nil, xerr.NewErrCode(xerr.MENU_NOT_FOUND)
	}
	return &adminpb.GetMenuResponse{Menu: &adminpb.MenuInfo{
		Id: int64(menu.Id), ParentId: int64(menu.ParentId), Title: menu.Title, Name: menu.Name,
		Type: int32(menu.Type), Path: menu.Path, Component: menu.Component, PermCode: menu.PermCode,
		Icon: menu.Icon, Sort: int32(menu.Sort), Visible: menu.Visible != 0, Status: int32(menu.Status),
		CreatedAt: menu.CreatedAt.Unix(),
	}}, nil
}
