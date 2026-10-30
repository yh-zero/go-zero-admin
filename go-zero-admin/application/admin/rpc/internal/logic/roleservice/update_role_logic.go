package roleservicelogic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type UpdateRoleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRoleLogic {
	return &UpdateRoleLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateRoleLogic) UpdateRole(in *adminpb.UpdateRoleRequest) (*adminpb.NoDataResponse, error) {
	role, err := l.svcCtx.SysRoleModel.FindOne(l.ctx, uint64(in.RoleId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.ROLE_NOT_FOUND)
	}
	role.Name = in.Name
	role.Code = in.Code
	role.Description = in.Description
	role.DataScope = in.DataScope
	role.Sort = int64(in.Sort)
	role.Status = int64(in.Status)
	role.UpdatedAt = time.Now()
	err = l.svcCtx.SysRoleModel.Update(l.ctx, role)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	// 重建 role_menu
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "DELETE FROM sys_role_menu WHERE role_id=?", in.RoleId)
	for _, menuID := range in.MenuIds {
		_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?,?)", in.RoleId, menuID)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
