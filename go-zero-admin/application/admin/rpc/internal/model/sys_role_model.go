package model

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysRoleModel = (*customSysRoleModel)(nil)

type (
	// SysRoleModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysRoleModel.
	SysRoleModel interface {
		sysRoleModel
		// ListRolePage 分页查有效角色
		ListRolePage(ctx context.Context, f RoleListFilter, page, pageSize int64) ([]*RoleRow, int64, error)
		// ListMenuIdsByRole 查角色已授权的有效菜单 ID
		ListMenuIdsByRole(ctx context.Context, roleId int64) ([]int64, error)
		// ListMenuIdsByRoleIds 批量查多个角色的有效菜单 ID（替代逐角色 N+1）
		ListMenuIdsByRoleIds(ctx context.Context, roleIds []int64) (map[int64][]int64, error)
		// CountUsersByRoleCodes 按角色码统计用户数（casbin g 规则）
		CountUsersByRoleCodes(ctx context.Context, roleCodes []string) (map[string]int64, error)
	}

	customSysRoleModel struct {
		*defaultSysRoleModel
	}
)

// RoleListFilter 角色列表筛选（空值/0 不过滤）
type RoleListFilter struct {
	Name   string
	Code   string
	Status int64
}

// RoleRow 角色列表行（时间列 UNIX_TIMESTAMP 转 int64，带 AS 别名）
type RoleRow struct {
	Id          int64  `db:"id"`
	Name        string `db:"name"`
	Code        string `db:"code"`
	Description string `db:"description"`
	DataScope   string `db:"data_scope"`
	Sort        int32  `db:"sort"`
	Status      int32  `db:"status"`
	CreatedAt   int64  `db:"created_at"`
	UpdatedAt   int64  `db:"updated_at"`
}

// NewSysRoleModel returns a model for the database table.
func NewSysRoleModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysRoleModel {
	return &customSysRoleModel{
		defaultSysRoleModel: newSysRoleModel(conn, c, opts...),
	}
}

func (m *customSysRoleModel) ListRolePage(ctx context.Context, f RoleListFilter, page, pageSize int64) ([]*RoleRow, int64, error) {
	where := "deleted_at IS NULL"
	var args []any
	if f.Name != "" {
		where += " AND name LIKE ?"
		args = append(args, "%"+f.Name+"%")
	}
	if f.Code != "" {
		where += " AND code LIKE ?"
		args = append(args, "%"+f.Code+"%")
	}
	if f.Status > 0 {
		where += " AND status=?"
		args = append(args, f.Status)
	}
	var total int64
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) FROM sys_role WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var rows []*RoleRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id, name, code, description, data_scope, sort, status, "+
			"UNIX_TIMESTAMP(created_at) AS created_at, UNIX_TIMESTAMP(updated_at) AS updated_at "+
			"FROM sys_role WHERE "+where+" ORDER BY sort, id LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...)
	return rows, total, err
}

func (m *customSysRoleModel) ListMenuIdsByRole(ctx context.Context, roleId int64) ([]int64, error) {
	ids := make([]int64, 0)
	// 只统计仍有效的菜单，软删菜单的授权残留不回显
	err := m.QueryRowsNoCacheCtx(ctx, &ids,
		"SELECT rm.menu_id FROM sys_role_menu rm JOIN sys_menu m ON rm.menu_id=m.id WHERE rm.role_id=? AND m.deleted_at IS NULL ORDER BY rm.menu_id",
		roleId)
	return ids, err
}

func (m *customSysRoleModel) ListMenuIdsByRoleIds(ctx context.Context, roleIds []int64) (map[int64][]int64, error) {
	result := make(map[int64][]int64, len(roleIds))
	if len(roleIds) == 0 {
		return result, nil
	}
	args := make([]any, 0, len(roleIds))
	for _, id := range roleIds {
		args = append(args, id)
	}
	type row struct {
		RoleId int64 `db:"role_id"`
		MenuId int64 `db:"menu_id"`
	}
	var rows []row
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT rm.role_id, rm.menu_id FROM sys_role_menu rm JOIN sys_menu m ON rm.menu_id=m.id WHERE rm.role_id IN ("+inPlaceholders(len(roleIds))+") AND m.deleted_at IS NULL ORDER BY rm.role_id, rm.menu_id",
		args...)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		result[r.RoleId] = append(result[r.RoleId], r.MenuId)
	}
	return result, nil
}

func (m *customSysRoleModel) CountUsersByRoleCodes(ctx context.Context, roleCodes []string) (map[string]int64, error) {
	counts := make(map[string]int64, len(roleCodes))
	if len(roleCodes) == 0 {
		return counts, nil
	}
	args := make([]any, 0, len(roleCodes))
	for _, code := range roleCodes {
		args = append(args, code)
	}
	type row struct {
		RoleCode string `db:"role_code"`
		Cnt      int64  `db:"cnt"`
	}
	var rows []row
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT v1 AS role_code, COUNT(DISTINCT v0) AS cnt FROM casbin_rule WHERE ptype='g' AND v0 LIKE 'user:%' AND v1 IN ("+inPlaceholders(len(roleCodes))+") GROUP BY v1",
		args...)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		counts[r.RoleCode] = r.Cnt
	}
	return counts, nil
}
