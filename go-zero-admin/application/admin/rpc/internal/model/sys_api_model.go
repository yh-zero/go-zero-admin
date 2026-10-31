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
		// ListApiPage 分页查接口
		ListApiPage(ctx context.Context, f ApiListFilter, page, pageSize int64) ([]*ApiRow, int64, error)
	}

	customSysApiModel struct {
		*defaultSysApiModel
	}
)

// ApiListFilter 接口列表筛选（空值不过滤）
type ApiListFilter struct {
	Group  string
	Method string
	Path   string
}

// ApiRow 接口行（时间列 UNIX_TIMESTAMP 转 int64）
type ApiRow struct {
	Id        int64  `db:"id"`
	GroupName string `db:"group_name"`
	Method    string `db:"method"`
	Path      string `db:"path"`
	Title     string `db:"title"`
	CreatedAt int64  `db:"created_at"`
}

// NewSysApiModel returns a model for the database table.
func NewSysApiModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysApiModel {
	return &customSysApiModel{
		defaultSysApiModel: newSysApiModel(conn, c, opts...),
	}
}

func (m *customSysApiModel) ListApiPage(ctx context.Context, f ApiListFilter, page, pageSize int64) ([]*ApiRow, int64, error) {
	where := "1=1"
	var args []any
	if f.Group != "" {
		where += " AND group_name=?"
		args = append(args, f.Group)
	}
	if f.Method != "" {
		where += " AND method=?"
		args = append(args, f.Method)
	}
	if f.Path != "" {
		where += " AND path LIKE ?"
		args = append(args, "%"+f.Path+"%")
	}
	var total int64
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) FROM sys_api WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var rows []*ApiRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id, group_name, method, path, title, UNIX_TIMESTAMP(created_at) AS created_at "+
			"FROM sys_api WHERE "+where+" ORDER BY id LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...)
	return rows, total, err
}
