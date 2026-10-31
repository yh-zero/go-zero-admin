package model

import (
	"context"

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
