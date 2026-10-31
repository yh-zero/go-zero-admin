package model

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysMenuModel = (*customSysMenuModel)(nil)

type (
	// SysMenuModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysMenuModel.
	SysMenuModel interface {
		sysMenuModel
		// FindMenuRow 查单个有效菜单（含软删过滤），时间戳已转 int64
		FindMenuRow(ctx context.Context, id uint64) (*MenuRow, error)
		// ListMenuPage 分页查菜单，parentId/menuType 为 0 时不过滤
		ListMenuPage(ctx context.Context, parentId, menuType int64, page, pageSize int64) ([]*MenuRow, int64, error)
		// ListMenuTree 全量查有效菜单（树用），menuType/status 为 0 时不过滤
		ListMenuTree(ctx context.Context, menuType, status int64) ([]*MenuRow, error)
		// ListMenusByRoleCodes 按角色码查其有效菜单（IN 在内部展开，空切片返回空）
		ListMenusByRoleCodes(ctx context.Context, roleCodes []string) ([]*MenuRow, error)
		// ListPermCodesByRoleCodes 按角色码查按钮权限码（type=3）
		ListPermCodesByRoleCodes(ctx context.Context, roleCodes []string) ([]string, error)
		// ListRoleCodesByUser 查用户角色码（casbin g 规则 v0=user:N）
		ListRoleCodesByUser(ctx context.Context, userId int64) ([]string, error)
		// SoftDeleteSubtree 事务软删菜单子树（按 parent_id 递归），连带删 sys_role_menu 关联并清 id 缓存；返回删除行数
		SoftDeleteSubtree(ctx context.Context, rootId uint64) (int64, error)
		// CollectSubtreeIds 收集子树全部 id（含 root，带防环 visited 集合）；root 不存在或已删时返回 sqlx.ErrNotFound
		CollectSubtreeIds(ctx context.Context, rootId uint64) ([]int64, error)
	}

	customSysMenuModel struct {
		*defaultSysMenuModel
	}
)

// MenuRow 菜单查询行（时间字段 SQL 层 UNIX_TIMESTAMP 转 int64）
type MenuRow struct {
	Id        int64  `db:"id"`
	ParentId  int64  `db:"parent_id"`
	Title     string `db:"title"`
	Name      string `db:"name"`
	Type      int32  `db:"type"`
	Path      string `db:"path"`
	Component string `db:"component"`
	PermCode  string `db:"perm_code"`
	Icon      string `db:"icon"`
	Sort      int32  `db:"sort"`
	Visible   bool   `db:"visible"`
	Status    int32  `db:"status"`
	CreatedAt int64  `db:"created_at"`
}

// 表达式列必须 AS 别名，否则 sqlx 按表达式原名匹配字段导致静默零值
const menuRowCols = "id, parent_id, title, name, type, path, component, perm_code, icon, sort, visible, status, UNIX_TIMESTAMP(created_at) AS created_at"

// inPlaceholders 展开 IN 占位符（go-sql-driver 不支持切片直传）；调用方保证 n > 0
func inPlaceholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

// NewSysMenuModel returns a model for the database table.
func NewSysMenuModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysMenuModel {
	return &customSysMenuModel{
		defaultSysMenuModel: newSysMenuModel(conn, c, opts...),
	}
}

func (m *customSysMenuModel) FindMenuRow(ctx context.Context, id uint64) (*MenuRow, error) {
	var row MenuRow
	err := m.QueryRowNoCacheCtx(ctx, &row, "SELECT "+menuRowCols+" FROM sys_menu WHERE id=? AND deleted_at IS NULL", id)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (m *customSysMenuModel) ListMenuPage(ctx context.Context, parentId, menuType int64, page, pageSize int64) ([]*MenuRow, int64, error) {
	where := "deleted_at IS NULL"
	var args []any
	if parentId > 0 {
		where += " AND parent_id=?"
		args = append(args, parentId)
	}
	if menuType > 0 {
		where += " AND type=?"
		args = append(args, menuType)
	}
	var total int64
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) FROM sys_menu WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var rows []*MenuRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT "+menuRowCols+" FROM sys_menu WHERE "+where+" ORDER BY sort, id LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...)
	return rows, total, err
}

func (m *customSysMenuModel) ListMenuTree(ctx context.Context, menuType, status int64) ([]*MenuRow, error) {
	where := "deleted_at IS NULL"
	var args []any
	if menuType > 0 {
		where += " AND type=?"
		args = append(args, menuType)
	}
	if status > 0 {
		where += " AND status=?"
		args = append(args, status)
	}
	var rows []*MenuRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT "+menuRowCols+" FROM sys_menu WHERE "+where+" ORDER BY sort, id", args...)
	return rows, err
}

func (m *customSysMenuModel) ListMenusByRoleCodes(ctx context.Context, roleCodes []string) ([]*MenuRow, error) {
	if len(roleCodes) == 0 {
		return nil, nil
	}
	args := make([]any, 0, len(roleCodes))
	for _, code := range roleCodes {
		args = append(args, code)
	}
	var rows []*MenuRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT DISTINCT m.id, m.parent_id, m.title, m.name, m.type, m.path, m.component, m.perm_code, m.icon, m.sort, m.visible, m.status, UNIX_TIMESTAMP(m.created_at) AS created_at "+
			"FROM sys_menu m JOIN sys_role_menu rm ON m.id=rm.menu_id JOIN sys_role r ON rm.role_id=r.id "+
			"WHERE r.code IN ("+inPlaceholders(len(roleCodes))+") AND m.deleted_at IS NULL AND r.deleted_at IS NULL ORDER BY m.sort, m.id",
		args...)
	return rows, err
}

func (m *customSysMenuModel) ListPermCodesByRoleCodes(ctx context.Context, roleCodes []string) ([]string, error) {
	if len(roleCodes) == 0 {
		return nil, nil
	}
	args := make([]any, 0, len(roleCodes))
	for _, code := range roleCodes {
		args = append(args, code)
	}
	codes := make([]string, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &codes,
		"SELECT DISTINCT m.perm_code FROM sys_menu m JOIN sys_role_menu rm ON m.id=rm.menu_id JOIN sys_role r ON rm.role_id=r.id "+
			"WHERE r.code IN ("+inPlaceholders(len(roleCodes))+") AND m.type=3 AND m.perm_code != '' AND m.deleted_at IS NULL",
		args...)
	return codes, err
}

func (m *customSysMenuModel) CollectSubtreeIds(ctx context.Context, rootId uint64) ([]int64, error) {
	// 先确认目标存在且未删除
	root, err := m.FindOne(ctx, rootId)
	if err != nil {
		return nil, err
	}
	if root.DeletedAt.Valid {
		return nil, sqlx.ErrNotFound
	}
	// 按 parent_id 逐层展开（不用 path 前缀，避免 /system 误伤 /system2）；
	// visited 防环：parent_id 被改成环时不会死循环
	visited := map[int64]struct{}{int64(rootId): {}}
	ids := []int64{int64(rootId)}
	frontier := []int64{int64(rootId)}
	for len(frontier) > 0 {
		args := make([]any, 0, len(frontier))
		for _, id := range frontier {
			args = append(args, id)
		}
		var children []int64
		if err := m.QueryRowsNoCacheCtx(ctx, &children,
			"SELECT id FROM sys_menu WHERE parent_id IN ("+inPlaceholders(len(frontier))+") AND deleted_at IS NULL", args...); err != nil {
			return nil, err
		}
		next := make([]int64, 0, len(children))
		for _, id := range children {
			if _, seen := visited[id]; seen {
				continue
			}
			visited[id] = struct{}{}
			next = append(next, id)
		}
		ids = append(ids, next...)
		frontier = next
	}
	return ids, nil
}

func (m *customSysMenuModel) SoftDeleteSubtree(ctx context.Context, rootId uint64) (int64, error) {
	ids, err := m.CollectSubtreeIds(ctx, rootId)
	if err != nil {
		return 0, err
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	ph := inPlaceholders(len(ids))
	err = m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if _, err := s.ExecCtx(ctx, "UPDATE sys_menu SET deleted_at=NOW() WHERE id IN ("+ph+") AND deleted_at IS NULL", args...); err != nil {
			return err
		}
		_, err := s.ExecCtx(ctx, "DELETE FROM sys_role_menu WHERE menu_id IN ("+ph+")", args...)
		return err
	})
	if err != nil {
		return 0, err
	}
	// 清 id 缓存（当前菜单无缓存读路径，防御性清理防未来接入缓存后取到脏数据）
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysMenuIdPrefix, id))
	}
	if err := m.DelCacheCtx(ctx, keys...); err != nil {
		return int64(len(ids)), err
	}
	return int64(len(ids)), nil
}

func (m *customSysMenuModel) ListRoleCodesByUser(ctx context.Context, userId int64) ([]string, error) {
	codes := make([]string, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &codes,
		"SELECT v1 FROM casbin_rule WHERE ptype='g' AND v0=?", "user:"+strconv.FormatInt(userId, 10))
	return codes, err
}
