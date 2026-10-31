package roleservicelogic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

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
	if role.DeletedAt.Valid {
		return nil, xerr.NewErrCode(xerr.ROLE_NOT_FOUND)
	}
	role.Name = in.Name
	role.Code = in.Code
	role.Description = in.Description
	role.DataScope = in.DataScope
	role.Sort = int64(in.Sort)
	role.Status = int64(in.Status)
	role.UpdatedAt = time.Now()
	if err = l.svcCtx.SysRoleModel.Update(l.ctx, role); err != nil {
		l.Errorf("update role error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	// 重建 role_menu：去重 + 事务，任一步失败整体回滚（旧实现吞错且无事务，可能删光授权）
	seen := make(map[int64]struct{}, len(in.MenuIds))
	menuIds := make([]int64, 0, len(in.MenuIds))
	for _, id := range in.MenuIds {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		menuIds = append(menuIds, id)
	}
	err = l.svcCtx.DB.TransactCtx(l.ctx, func(ctx context.Context, s sqlx.Session) error {
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
		l.Errorf("rebuild role menus error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	// 角色 code 是 Casbin 策略主体，变更后必须广播刷新
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
