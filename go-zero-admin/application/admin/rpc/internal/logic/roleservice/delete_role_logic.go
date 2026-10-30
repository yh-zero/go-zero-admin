package roleservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type DeleteRoleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteRoleLogic {
	return &DeleteRoleLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteRoleLogic) DeleteRole(in *adminpb.DeleteRoleRequest) (*adminpb.NoDataResponse, error) {
	role, err := l.svcCtx.SysRoleModel.FindOne(l.ctx, uint64(in.RoleId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.ROLE_NOT_FOUND)
	}
	// 软删角色
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE sys_role SET deleted_at=NOW() WHERE id=?", in.RoleId)
	// 删 role_menu
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "DELETE FROM sys_role_menu WHERE role_id=?", in.RoleId)
	// 删 casbin 规则（p: v0=role code; g: v1=role code）
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "DELETE FROM casbin_rule WHERE (ptype='p' AND v0=?) OR (ptype='g' AND v1=?)", role.Code, role.Code)
	// 通知 API 层刷新策略
	_, _ = l.svcCtx.Redis.Publish("casbin:policy:changed", role.Code)
	return &adminpb.NoDataResponse{}, nil
}
