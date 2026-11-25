package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysPermissionModel = (*customSysPermissionModel)(nil)

type (
	SysPermissionModel interface {
		ListByMenuIDs(ctx context.Context, menuIDs []int64) ([]PermissionRoute, error)
		ReplaceForMenu(ctx context.Context, menuID int64, apiIDs []int64) error
		DeleteByMenuID(ctx context.Context, menuID int64) error
	}
	customSysPermissionModel struct{ *defaultSysPermissionModel }
	PermissionRoute          struct {
		MenuID int64  `db:"menu_id"`
		ApiID  int64  `db:"api_id"`
		Method string `db:"method"`
		Path   string `db:"path"`
	}
)

func NewSysPermissionModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysPermissionModel {
	return &customSysPermissionModel{defaultSysPermissionModel: newSysPermissionModel(conn, c, opts...)}
}

func (m *customSysPermissionModel) ListByMenuIDs(ctx context.Context, menuIDs []int64) ([]PermissionRoute, error) {
	if len(menuIDs) == 0 {
		return []PermissionRoute{}, nil
	}
	args := make([]any, len(menuIDs))
	for i, id := range menuIDs {
		args[i] = id
	}
	var rows []PermissionRoute
	err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT p.menu_id,p.api_id,a.method,a.path FROM sys_permission p JOIN sys_api a ON a.id=p.api_id WHERE p.menu_id IN ("+inPlaceholders(len(args))+") ORDER BY p.menu_id,p.api_id", args...)
	return rows, err
}

func (m *customSysPermissionModel) ReplaceForMenu(ctx context.Context, menuID int64, apiIDs []int64) error {
	return m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if _, err := s.ExecCtx(ctx, "DELETE FROM sys_permission WHERE menu_id=?", menuID); err != nil {
			return err
		}
		seen := make(map[int64]bool, len(apiIDs))
		for _, apiID := range apiIDs {
			if apiID <= 0 {
				return fmt.Errorf("%w: API 不存在", ErrInvalidMutation)
			}
			if seen[apiID] {
				continue
			}
			seen[apiID] = true
			result, err := s.ExecCtx(ctx, "INSERT INTO sys_permission (menu_id,api_id) SELECT ?,id FROM sys_api WHERE id=?", menuID, apiID)
			if err != nil {
				return err
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affected != 1 {
				return fmt.Errorf("%w: API 不存在", ErrInvalidMutation)
			}
		}
		return nil
	})
}

func (m *customSysPermissionModel) DeleteByMenuID(ctx context.Context, menuID int64) error {
	_, err := m.ExecNoCacheCtx(ctx, "DELETE FROM sys_permission WHERE menu_id=?", menuID)
	return err
}
