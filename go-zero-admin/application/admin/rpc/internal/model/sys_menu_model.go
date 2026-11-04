package model

import (
	"context"
	"fmt"
	"sort"
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
		Create(ctx context.Context, menu *SysMenu) error
		UpdatePartial(ctx context.Context, id uint64, patch MenuPatch) error
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
		ListRoleCodesByUser(ctx context.Context, userId, deptId int64) ([]string, error)
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

// 结构写入先锁固定行，再按ID锁菜单。菜单路径不反向锁角色，保持role->menu锁序。
func lockMenuStructure(ctx context.Context, s sqlx.Session) error {
	var id int64
	if err := s.QueryRowCtx(ctx, &id, "SELECT id FROM sys_menu_lock WHERE id=1 FOR UPDATE"); err != nil {
		return fmt.Errorf("菜单结构锁不可用: %w", err)
	}
	return nil
}

func lockMenuTree(ctx context.Context, s sqlx.Session) ([]*SysMenu, error) {
	var rows []*SysMenu
	err := s.QueryRowsCtx(ctx, &rows, "SELECT "+sysMenuRows+" FROM sys_menu WHERE deleted_at IS NULL ORDER BY id FOR UPDATE")
	return rows, err
}

func validateMenuParent(menu *SysMenu, rows []*SysMenu) error {
	byId := make(map[uint64]*SysMenu, len(rows))
	for _, row := range rows {
		byId[row.Id] = row
	}
	if menu.Type < 1 || menu.Type > 3 || menu.Status < 1 || menu.Status > 2 {
		return ErrInvalidMutation
	}
	if menu.Type == 3 {
		for _, row := range rows {
			if row.ParentId == menu.Id && menu.Id != 0 {
				return fmt.Errorf("%w: 按钮不能包含子菜单", ErrInvalidMutation)
			}
		}
	}
	seen := map[uint64]bool{}
	for parent := menu.ParentId; parent != 0; {
		if parent == menu.Id || seen[parent] {
			return fmt.Errorf("%w: 不能挂载到自身或子菜单下", ErrInvalidMutation)
		}
		seen[parent] = true
		row, ok := byId[parent]
		if !ok {
			return fmt.Errorf("%w: 父菜单不存在", ErrInvalidMutation)
		}
		if parent == menu.ParentId && row.Type == 3 {
			return fmt.Errorf("%w: 按钮不能包含子菜单", ErrInvalidMutation)
		}
		parent = row.ParentId
	}
	return nil
}

func (m *customSysMenuModel) Create(ctx context.Context, menu *SysMenu) error {
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if err := lockMenuStructure(ctx, s); err != nil {
			return err
		}
		rows, err := lockMenuTree(ctx, s)
		if err != nil {
			return err
		}
		if err := validateMenuParent(menu, rows); err != nil {
			return err
		}
		result, err := s.ExecCtx(ctx, "INSERT INTO sys_menu (parent_id,title,name,type,path,component,perm_code,icon,sort,visible,status) VALUES (?,?,?,?,?,?,?,?,?,?,?)", menu.ParentId, menu.Title, menu.Name, menu.Type, menu.Path, menu.Component, menu.PermCode, menu.Icon, menu.Sort, menu.Visible, menu.Status)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		menu.Id = uint64(id)
		return err
	})
	if err != nil {
		return err
	}
	return committedCacheError(m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysMenuIdPrefix, menu.Id)))
}

func (m *customSysMenuModel) UpdatePartial(ctx context.Context, id uint64, patch MenuPatch) error {
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		var current *SysMenu
		if patch.ParentId != nil || patch.Type != nil {
			if err := lockMenuStructure(ctx, s); err != nil {
				return err
			}
			rows, err := lockMenuTree(ctx, s)
			if err != nil {
				return err
			}
			for _, row := range rows {
				if row.Id == id {
					current = row
					break
				}
			}
			if current == nil {
				return sqlx.ErrNotFound
			}
			if err := applyMenuPatch(current, patch); err != nil {
				return err
			}
			if err := validateMenuParent(current, rows); err != nil {
				return err
			}
		} else {
			current = &SysMenu{}
			if err := s.QueryRowCtx(ctx, current, "SELECT "+sysMenuRows+" FROM sys_menu WHERE id=? AND deleted_at IS NULL FOR UPDATE", id); err != nil {
				return err
			}
			if err := applyMenuPatch(current, patch); err != nil {
				return err
			}
		}
		_, err := s.ExecCtx(ctx, "UPDATE sys_menu SET parent_id=?,title=?,name=?,type=?,path=?,component=?,perm_code=?,icon=?,sort=?,visible=?,status=? WHERE id=? AND deleted_at IS NULL", current.ParentId, current.Title, current.Name, current.Type, current.Path, current.Component, current.PermCode, current.Icon, current.Sort, current.Visible, current.Status, id)
		return err
	})
	if err != nil {
		return err
	}
	return committedCacheError(m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysMenuIdPrefix, id)))
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
			"WHERE r.code IN ("+inPlaceholders(len(roleCodes))+") AND m.deleted_at IS NULL AND r.deleted_at IS NULL AND m.status=1 AND r.status=1 ORDER BY m.sort, m.id",
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
	var rows []struct {
		Code string `db:"perm_code"`
	}
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT DISTINCT m.perm_code AS perm_code FROM sys_menu m JOIN sys_role_menu rm ON m.id=rm.menu_id JOIN sys_role r ON rm.role_id=r.id "+
			"WHERE r.code IN ("+inPlaceholders(len(roleCodes))+") AND m.type=3 AND m.perm_code != '' AND m.deleted_at IS NULL AND m.status=1 AND r.status=1 AND r.deleted_at IS NULL",
		args...)
	for _, row := range rows {
		codes = append(codes, row.Code)
	}
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
	var ids []int64
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if err := lockMenuStructure(ctx, s); err != nil {
			return err
		}
		rows, err := lockMenuTree(ctx, s)
		if err != nil {
			return err
		}
		exists := false
		for _, row := range rows {
			if row.Id == rootId {
				exists = true
				break
			}
		}
		if !exists {
			return sqlx.ErrNotFound
		}
		seen := map[uint64]bool{rootId: true}
		for changed := true; changed; {
			changed = false
			for _, row := range rows {
				if seen[row.ParentId] && !seen[row.Id] {
					seen[row.Id] = true
					changed = true
				}
			}
		}
		for id := range seen {
			ids = append(ids, int64(id))
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		args := make([]any, 0, len(ids))
		for _, id := range ids {
			args = append(args, id)
		}
		ph := inPlaceholders(len(ids))
		if _, err = s.ExecCtx(ctx, "DELETE FROM sys_role_menu WHERE menu_id IN ("+ph+")", args...); err != nil {
			return err
		}
		_, err = s.ExecCtx(ctx, "UPDATE sys_menu SET deleted_at=NOW() WHERE id IN ("+ph+") AND deleted_at IS NULL", args...)
		return err
	})
	if err != nil {
		return 0, err
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysMenuIdPrefix, id))
	}
	return int64(len(ids)), committedCacheError(m.DelCacheCtx(ctx, keys...))
}

func (m *customSysMenuModel) ListRoleCodesByUser(ctx context.Context, userId, deptId int64) ([]string, error) {
	codes := make([]string, 0)
	var rows []struct {
		Code string `db:"code"`
	}
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT DISTINCT g.v1 AS code FROM casbin_rule g JOIN sys_role r ON r.code=g.v1 WHERE g.ptype='g' AND g.v0=? AND (g.v2=? OR g.v2='*') AND r.status=1 AND r.deleted_at IS NULL", "user:"+strconv.FormatInt(userId, 10), "dept:"+strconv.FormatInt(deptId, 10))
	for _, row := range rows {
		codes = append(codes, row.Code)
	}
	return codes, err
}
