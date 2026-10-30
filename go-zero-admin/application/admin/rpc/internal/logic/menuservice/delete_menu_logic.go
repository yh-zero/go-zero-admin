package menuservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
)

type DeleteMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteMenuLogic {
	return &DeleteMenuLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteMenuLogic) DeleteMenu(in *adminpb.DeleteMenuRequest) (*adminpb.NoDataResponse, error) {
	// 级联软删子菜单（通过 path 前缀）
	var path string
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &path, "SELECT path FROM sys_menu WHERE id=?", in.MenuId)
	if path != "" {
		_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE sys_menu SET deleted_at=NOW() WHERE path LIKE ? AND deleted_at IS NULL", path+"%")
	} else {
		_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE sys_menu SET deleted_at=NOW() WHERE id=?", in.MenuId)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
