package roleservicelogic

import (
	"context"
	"fmt"

	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

type roleMutationInput struct {
	RoleID    uint64
	Code      string
	DataScope string
	MenuIDs   []int64
}

func requireRoleMutationAllowed(ctx context.Context, s *svc.ServiceContext, actor *adminpb.Actor, input roleMutationInput) error {
	if err := requireRoleActor(ctx, s, actor); err != nil {
		return err
	}
	super, err := s.SysUserModel.IsSuperAdmin(ctx, actor.UserId)
	if err != nil {
		return xerr.NewErrCode(xerr.DB_ERROR)
	}
	if super {
		return nil
	}
	code := input.Code
	if input.RoleID > 0 {
		role, err := s.SysRoleModel.FindOne(ctx, input.RoleID)
		if err != nil {
			return roleMutationError(err)
		}
		code = role.Code
		bound, err := roleBoundToUser(ctx, s, actor.UserId, role.Code)
		if err != nil {
			return xerr.NewErrCode(xerr.DB_ERROR)
		}
		if bound {
			return xerr.NewErrCodeMsg(xerr.PERMISSION_DENIED, "不能修改自己绑定的角色")
		}
		menuIDs, err := s.SysRoleModel.ListMenuIdsByRole(ctx, int64(input.RoleID))
		if err != nil {
			return xerr.NewErrCode(xerr.DB_ERROR)
		}
		if err := requireMenusAllowed(ctx, s, actor, menuIDs); err != nil {
			return err
		}
	}
	if code == "super_admin" {
		return xerr.NewErrCodeMsg(xerr.PERMISSION_DENIED, "内置超级管理员角色只能由超级管理员维护")
	}
	if input.DataScope != "" {
		allowed, err := model.CanDelegateScope(ctx, s.DB.QueryRowsCtx, model.UserActor{UserId: actor.UserId, DeptId: actor.DeptId}, input.DataScope)
		if err != nil {
			return xerr.NewErrCode(xerr.DB_ERROR)
		}
		if !allowed {
			return xerr.NewErrCodeMsg(xerr.PERMISSION_DENIED, "数据范围不能超过当前账号自身范围")
		}
	}
	return requireMenusAllowed(ctx, s, actor, input.MenuIDs)
}

func roleBoundToUser(ctx context.Context, s *svc.ServiceContext, userId int64, roleCode string) (bool, error) {
	var count int64
	err := s.DB.QueryRowCtx(ctx, &count, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='g' AND v0=? AND v1=?", fmt.Sprintf("user:%d", userId), roleCode)
	return count > 0, err
}

func requireMenusAllowed(ctx context.Context, s *svc.ServiceContext, actor *adminpb.Actor, menuIDs []int64) error {
	if len(menuIDs) == 0 {
		return nil
	}
	routes, err := s.SysPermissionModel.ListByMenuIDs(ctx, menuIDs)
	if err != nil {
		return xerr.NewErrCode(xerr.DB_ERROR)
	}
	sub := fmt.Sprintf("user:%d", actor.UserId)
	domain := fmt.Sprintf("dept:%d", actor.DeptId)
	for _, route := range routes {
		allowed, err := s.Enforcer.Enforce(sub, domain, route.Path, route.Method)
		if err != nil {
			return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
		if !allowed {
			return xerr.NewErrCodeMsg(xerr.PERMISSION_DENIED, "不能授予超出当前账号权限的操作")
		}
	}
	return nil
}
