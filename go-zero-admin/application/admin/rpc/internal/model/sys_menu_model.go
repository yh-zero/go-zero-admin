package model

import (
	"context"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"strings"
)

var (
	_                SysMenuModel = (*customSysMenuModel)(nil)
	ErrMenuHierarchy              = errors.New("菜单层级异常，无法安全删除")
)

type (
	// SysMenuModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysMenuModel.
	SysMenuModel interface {
		sysMenuModel
		FindList(context.Context, int64, int32, int64, int64) ([]MenuRow, int64, error)
		FindTree(context.Context, int32, int32) ([]MenuRow, error)
		FindByUser(context.Context, int64) ([]MenuRow, error)
		FindPermCodesByUser(context.Context, int64) ([]string, error)
		DeleteSubtree(context.Context, uint64) error
		UpdateActive(context.Context, *SysMenu) error
	}

	customSysMenuModel struct {
		*defaultSysMenuModel
	}
)

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

type totalRow struct {
	Total int64 `db:"total"`
}

type menuHierarchyRow struct {
	Id       uint64 `db:"id"`
	ParentId uint64 `db:"parent_id"`
}

func (m *customSysMenuModel) UpdateActive(ctx context.Context, data *SysMenu) error {
	err := m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var current menuHierarchyRow
		// 锁定有效记录以串行化更新和软删除；字段未变时不能用RowsAffected判断不存在。
		if err := session.QueryRowCtx(ctx, &current,
			"SELECT id, parent_id FROM sys_menu WHERE id=? AND deleted_at IS NULL FOR UPDATE", data.Id); err != nil {
			return err
		}
		_, err := session.ExecCtx(ctx,
			"UPDATE sys_menu SET parent_id=?, title=?, name=?, type=?, path=?, component=?, perm_code=?, icon=?, sort=?, visible=?, status=? WHERE id=? AND deleted_at IS NULL",
			data.ParentId, data.Title, data.Name, data.Type, data.Path, data.Component, data.PermCode,
			data.Icon, data.Sort, data.Visible, data.Status, data.Id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, m.formatPrimary(data.Id))
}

func menuSubtreeIds(rows []menuHierarchyRow, root uint64) ([]uint64, error) {
	parents := make(map[uint64]uint64, len(rows))
	children := make(map[uint64][]uint64)
	for _, row := range rows {
		if _, duplicate := parents[row.Id]; row.Id == 0 || duplicate {
			return nil, ErrMenuHierarchy
		}
		parents[row.Id] = row.ParentId
		children[row.ParentId] = append(children[row.ParentId], row.Id)
	}
	if _, exists := parents[root]; !exists {
		return nil, ErrNotFound
	}
	ids := []uint64{root}
	seen := make(map[uint64]bool)
	for i := 0; i < len(ids); i++ {
		id := ids[i]
		if seen[id] {
			return nil, ErrMenuHierarchy
		}
		seen[id] = true
		ids = append(ids, children[id]...)
	}
	return ids, nil
}

func (m *customSysMenuModel) DeleteSubtree(ctx context.Context, root uint64) error {
	var ids []uint64
	err := m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var rows []menuHierarchyRow
		// 锁定现有层级，避免删除集合计算后并发移动节点。
		if err := session.QueryRowsCtx(ctx, &rows, "SELECT id, parent_id FROM sys_menu WHERE deleted_at IS NULL ORDER BY id FOR UPDATE"); err != nil {
			return err
		}
		var err error
		ids, err = menuSubtreeIds(rows, root)
		if err != nil {
			return err
		}
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
		args := make([]any, 0, len(ids))
		for _, id := range ids {
			args = append(args, id)
		}
		if _, err := session.ExecCtx(ctx, "DELETE FROM sys_role_menu WHERE menu_id IN ("+placeholders+")", args...); err != nil {
			return err
		}
		_, err = session.ExecCtx(ctx, "UPDATE sys_menu SET deleted_at=NOW() WHERE id IN ("+placeholders+") AND deleted_at IS NULL", args...)
		return err
	})
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, m.formatPrimary(id))
	}
	return m.DelCacheCtx(ctx, keys...)
}

const menuSelectColumns = "id, parent_id, title, name, type, path, component, perm_code, icon, sort, visible, status, UNIX_TIMESTAMP(created_at) AS created_at"

const userMenuJoins = " FROM sys_menu m JOIN sys_role_menu rm ON m.id=rm.menu_id JOIN sys_role r ON rm.role_id=r.id " +
	"JOIN casbin_rule g ON g.v1=r.code AND g.ptype='g' " +
	"WHERE g.v0=? AND r.status=1 AND r.deleted_at IS NULL AND m.status=1 AND m.deleted_at IS NULL"

func pageBounds(page, pageSize int64) (int64, int64) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return pageSize, (page - 1) * pageSize
}

func (m *customSysMenuModel) FindList(ctx context.Context, parentId int64, menuType int32, page, pageSize int64) ([]MenuRow, int64, error) {
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
	var count totalRow
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM sys_menu WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	limit, offset := pageBounds(page, pageSize)
	rows := make([]MenuRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT "+menuSelectColumns+" FROM sys_menu WHERE "+where+" ORDER BY sort, id LIMIT ? OFFSET ?", append(args, limit, offset)...)
	return rows, count.Total, err
}

func (m *customSysMenuModel) FindTree(ctx context.Context, menuType, status int32) ([]MenuRow, error) {
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
	rows := make([]MenuRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT "+menuSelectColumns+" FROM sys_menu WHERE "+where+" ORDER BY sort, id", args...)
	return rows, err
}

func (m *customSysMenuModel) FindByUser(ctx context.Context, userId int64) ([]MenuRow, error) {
	rows := make([]MenuRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT DISTINCT m.id, m.parent_id, m.title, m.name, m.type, m.path, m.component, m.perm_code, m.icon, m.sort, m.visible, m.status, UNIX_TIMESTAMP(m.created_at) AS created_at"+
			userMenuJoins+" ORDER BY m.sort, m.id", fmt.Sprintf("user:%d", userId))
	return rows, err
}

func (m *customSysMenuModel) FindPermCodesByUser(ctx context.Context, userId int64) ([]string, error) {
	type permissionRow struct {
		PermCode string `db:"perm_code"`
	}
	var rows []permissionRow
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT DISTINCT m.perm_code AS perm_code"+userMenuJoins+
		" AND m.type=3 AND m.perm_code != '' ORDER BY m.perm_code", fmt.Sprintf("user:%d", userId)); err != nil {
		return nil, err
	}
	codes := make([]string, 0, len(rows))
	for _, row := range rows {
		codes = append(codes, row.PermCode)
	}
	return codes, nil
}

// NewSysMenuModel returns a model for the database table.
func NewSysMenuModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysMenuModel {
	return &customSysMenuModel{
		defaultSysMenuModel: newSysMenuModel(conn, c, opts...),
	}
}
