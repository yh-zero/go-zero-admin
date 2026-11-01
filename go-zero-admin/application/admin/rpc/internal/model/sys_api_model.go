package model

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysApiModel = (*customSysApiModel)(nil)

type (
	// SysApiModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysApiModel.
	SysApiModel interface {
		sysApiModel
		DeleteWithCache(ctx context.Context, id int64) error
		// ListApiPage 分页查接口
		ListApiPage(ctx context.Context, f ApiListFilter, page, pageSize int64) ([]*ApiRow, int64, error)
		SyncRoutes(ctx context.Context, routes []ApiRouteInput) (int64, int64, error)
	}

	customSysApiModel struct {
		*defaultSysApiModel
	}
)

type ApiRouteInput struct{ Method, Path, Group, Title string }

// SyncRoutes 只同步实际路由，不删除来源未知的手工资源或覆盖人工标题。
func (m *customSysApiModel) SyncRoutes(ctx context.Context, routes []ApiRouteInput) (int64, int64, error) {
	sort.Slice(routes, func(i, j int) bool { return routes[i].Method+routes[i].Path < routes[j].Method+routes[j].Path })
	var added, updated int64
	keys := make([]string, 0, len(routes)*2)
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		for _, route := range routes {
			var row struct {
				Id    int64  `db:"id"`
				Group string `db:"group_name"`
			}
			err := s.QueryRowCtx(ctx, &row, "SELECT id,group_name FROM sys_api WHERE method=? AND path=? FOR UPDATE", route.Method, route.Path)
			if errors.Is(err, sqlx.ErrNotFound) {
				result, insertErr := s.ExecCtx(ctx, "INSERT INTO sys_api (group_name,method,path,title) VALUES (?,?,?,?)", route.Group, route.Method, route.Path, route.Title)
				if insertErr != nil {
					return insertErr
				}
				row.Id, err = result.LastInsertId()
				if err != nil {
					return err
				}
				added++
			} else if err != nil {
				return err
			} else if row.Group != route.Group {
				if _, err := s.ExecCtx(ctx, "UPDATE sys_api SET group_name=? WHERE id=?", route.Group, row.Id); err != nil {
					return err
				}
				updated++
			}
			keys = append(keys, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysApiIdPrefix, row.Id), fmt.Sprintf("%s%s:%s", cacheGozeroAdminNewSysApiMethodPathPrefix, route.Method, route.Path))
		}
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	if len(keys) > 0 {
		err = m.DelCacheCtx(ctx, keys...)
	}
	return added, updated, err
}

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

func (m *customSysApiModel) DeleteWithCache(ctx context.Context, id int64) error {
	var row struct {
		Method string `db:"method"`
		Path   string `db:"path"`
	}
	found := false
	err := m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		err := session.QueryRowCtx(ctx, &row, "SELECT method,path FROM sys_api WHERE id=? FOR UPDATE", id)
		if errors.Is(err, sqlx.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		_, err = session.ExecCtx(ctx, "DELETE FROM sys_api WHERE id=?", id)
		return err
	})
	if err != nil {
		return err
	}
	keys := []string{fmt.Sprintf("%s%d", cacheGozeroAdminNewSysApiIdPrefix, id)}
	if found {
		keys = append(keys, fmt.Sprintf("%s%s:%s", cacheGozeroAdminNewSysApiMethodPathPrefix, row.Method, row.Path))
	}
	return m.DelCacheCtx(ctx, keys...)
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
