package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysDeptModel = (*customSysDeptModel)(nil)

type (
	// SysDeptModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysDeptModel.
	SysDeptModel interface {
		sysDeptModel
		// ListDeptTree 全量查有效部门（树用），status 为 0 时不过滤
		ListDeptTree(ctx context.Context, status int64) ([]*DeptRow, error)
		// CollectSubtreeIds 收集子树全部 id（含 root，带防环 visited 集合）；root 不存在或已删时返回 sqlx.ErrNotFound
		CollectSubtreeIds(ctx context.Context, rootId uint64) ([]int64, error)
		// SoftDelete 软删并清 id 缓存；目标不存在或已删时返回 sqlx.ErrNotFound
		SoftDelete(ctx context.Context, id uint64) error
	}

	customSysDeptModel struct {
		*defaultSysDeptModel
	}
)

// DeptRow 部门行（时间列 UNIX_TIMESTAMP 转 int64）
type DeptRow struct {
	Id        int64  `db:"id"`
	ParentId  int64  `db:"parent_id"`
	Name      string `db:"name"`
	Leader    string `db:"leader"`
	Phone     string `db:"phone"`
	Email     string `db:"email"`
	Sort      int32  `db:"sort"`
	Status    int32  `db:"status"`
	Path      string `db:"path"`
	CreatedAt int64  `db:"created_at"`
}

// NewSysDeptModel returns a model for the database table.
func NewSysDeptModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysDeptModel {
	return &customSysDeptModel{
		defaultSysDeptModel: newSysDeptModel(conn, c, opts...),
	}
}

func (m *customSysDeptModel) CollectSubtreeIds(ctx context.Context, rootId uint64) ([]int64, error) {
	root, err := m.FindOne(ctx, rootId)
	if err != nil {
		return nil, err
	}
	if root.DeletedAt.Valid {
		return nil, sqlx.ErrNotFound
	}
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
			"SELECT id FROM sys_dept WHERE parent_id IN ("+inPlaceholders(len(frontier))+") AND deleted_at IS NULL", args...); err != nil {
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

func (m *customSysDeptModel) SoftDelete(ctx context.Context, id uint64) error {
	if _, err := m.FindOne(ctx, id); err != nil {
		return err
	}
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		_, err := s.ExecCtx(ctx, "UPDATE sys_dept SET deleted_at=NOW() WHERE id=? AND deleted_at IS NULL", id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysDeptIdPrefix, id))
}

func (m *customSysDeptModel) ListDeptTree(ctx context.Context, status int64) ([]*DeptRow, error) {
	where := "deleted_at IS NULL"
	var args []any
	if status > 0 {
		where += " AND status=?"
		args = append(args, status)
	}
	var rows []*DeptRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id, parent_id, name, leader, phone, email, sort, status, path, UNIX_TIMESTAMP(created_at) AS created_at "+
			"FROM sys_dept WHERE "+where+" ORDER BY sort, id", args...)
	return rows, err
}
