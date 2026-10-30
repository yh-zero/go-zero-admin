package roleservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type UpdateRoleMenusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateRoleMenusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRoleMenusLogic {
	return &UpdateRoleMenusLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateRoleMenusLogic) UpdateRoleMenus(in *adminpb.UpdateRoleMenusRequest) (*adminpb.NoDataResponse, error) {
	_, err := l.svcCtx.SysRoleModel.FindOne(l.ctx, uint64(in.RoleId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.ROLE_NOT_FOUND)
	}
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "DELETE FROM sys_role_menu WHERE role_id=?", in.RoleId)
	for _, menuID := range in.MenuIds {
		_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?,?)", in.RoleId, menuID)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
