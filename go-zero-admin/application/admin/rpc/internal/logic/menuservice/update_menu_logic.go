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

type UpdateMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMenuLogic {
	return &UpdateMenuLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateMenuLogic) UpdateMenu(in *adminpb.UpdateMenuRequest) (*adminpb.NoDataResponse, error) {
	menu := &model.SysMenu{
		Id: uint64(in.MenuId), ParentId: uint64(in.ParentId), Title: in.Title, Name: in.Name,
		Type: int64(in.Type), Path: in.Path, Component: in.Component, PermCode: in.PermCode,
		Icon: in.Icon, Sort: int64(in.Sort), Status: int64(in.Status),
	}
	if in.Visible {
		menu.Visible = 1
	}
	if err := l.svcCtx.SysMenuModel.UpdateActive(l.ctx, menu); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, xerr.NewErrCode(xerr.MENU_NOT_FOUND)
		}
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
