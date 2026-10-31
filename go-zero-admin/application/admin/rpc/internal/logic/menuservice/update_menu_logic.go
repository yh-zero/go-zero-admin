package menuservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
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
	visible := int64(0)
	if in.Visible {
		visible = 1
	}
	_, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"UPDATE sys_menu SET parent_id=?, title=?, name=?, type=?, path=?, component=?, perm_code=?, icon=?, sort=?, visible=?, status=? WHERE id=?",
		in.ParentId, in.Title, in.Name, in.Type, in.Path, in.Component, in.PermCode, in.Icon, in.Sort, visible, in.Status, in.MenuId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
