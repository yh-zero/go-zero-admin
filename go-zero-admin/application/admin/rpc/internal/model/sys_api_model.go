package model

import (
	"context"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysApiModel = (*customSysApiModel)(nil)

type (
	// SysApiModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysApiModel.
	SysApiModel interface {
		sysApiModel
		FindList(context.Context, string, string, string, int64, int64) ([]ApiRow, int64, error)
	}

	customSysApiModel struct {
		*defaultSysApiModel
	}
)

type ApiRow struct {
	Id        int64  `db:"id"`
	GroupName string `db:"group_name"`
	Method    string `db:"method"`
	Path      string `db:"path"`
	Title     string `db:"title"`
	CreatedAt int64  `db:"created_at"`
}

func (m *customSysApiModel) FindList(ctx context.Context, group, method, path string, page, pageSize int64) ([]ApiRow, int64, error) {
	where := "1=1"
	var args []any
	if group != "" {
		where += " AND group_name=?"
		args = append(args, group)
	}
	if method != "" {
		where += " AND method=?"
		args = append(args, method)
	}
	if path != "" {
		where += " AND path LIKE ?"
		args = append(args, "%"+path+"%")
	}
	var count totalRow
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM sys_api WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	limit, offset := pageBounds(page, pageSize)
	rows := make([]ApiRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id, group_name, method, path, title, UNIX_TIMESTAMP(created_at) AS created_at FROM sys_api WHERE "+where+" ORDER BY id LIMIT ? OFFSET ?",
		append(args, limit, offset)...)
	return rows, count.Total, err
}

// NewSysApiModel returns a model for the database table.
func NewSysApiModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysApiModel {
	return &customSysApiModel{
		defaultSysApiModel: newSysApiModel(conn, c, opts...),
	}
}
