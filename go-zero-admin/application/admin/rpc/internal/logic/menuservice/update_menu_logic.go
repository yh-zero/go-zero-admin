package menuservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

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
	err := l.svcCtx.SysMenuModel.UpdatePartial(l.ctx, uint64(in.MenuId), model.MenuPatch{
		ParentId: in.ParentId, Title: in.Title, Name: in.Name, Type: in.Type, Path: in.Path, Component: in.Component, PermCode: in.PermCode, Icon: in.Icon, Sort: in.Sort, Visible: in.Visible, Status: in.Status,
	})
	if model.MutationCommitted(err) {
		l.svcCtx.PublishPolicyChanged(l.ctx)
	}
	if err != nil {
		return nil, menuMutationError(err)
	}
	return &adminpb.NoDataResponse{}, nil
}

func menuMutationError(err error) error {
	if errors.Is(err, sqlx.ErrNotFound) {
		return xerr.NewErrCode(xerr.MENU_NOT_FOUND)
	}
	if errors.Is(err, model.ErrInvalidMutation) {
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, err.Error())
	}
	return xerr.NewErrCode(xerr.DB_ERROR)
}
