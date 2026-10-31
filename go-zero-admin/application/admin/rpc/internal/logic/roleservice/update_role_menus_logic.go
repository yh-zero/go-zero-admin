package roleservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

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
	if _, err := l.svcCtx.SysRoleModel.FindOne(l.ctx, uint64(in.RoleId)); err != nil {
		return nil, xerr.NewErrCode(xerr.ROLE_NOT_FOUND)
	}
	// 去重，避免唯一索引冲突导致整批失败
	seen := make(map[int64]struct{}, len(in.MenuIds))
	menuIds := make([]int64, 0, len(in.MenuIds))
	for _, id := range in.MenuIds {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		menuIds = append(menuIds, id)
	}
	// 事务：删旧授权 + 批量插新授权，任一步失败整体回滚
	err := l.svcCtx.DB.TransactCtx(l.ctx, func(ctx context.Context, s sqlx.Session) error {
		if _, err := s.ExecCtx(ctx, "DELETE FROM sys_role_menu WHERE role_id=?", in.RoleId); err != nil {
			return err
		}
		for _, menuId := range menuIds {
			if _, err := s.ExecCtx(ctx, "INSERT INTO sys_role_menu (role_id, menu_id) VALUES (?,?)", in.RoleId, menuId); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		l.Errorf("update role menus error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
