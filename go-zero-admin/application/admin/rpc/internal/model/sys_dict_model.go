package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysDictModel = (*customSysDictModel)(nil)

type (
	// SysDictModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysDictModel.
	SysDictModel interface {
		sysDictModel
		ListDictPage(ctx context.Context, filter DictListFilter, page, pageSize int64) ([]*DictRow, int64, error)
		SoftDelete(ctx context.Context, id int64) error
		UpdateActive(ctx context.Context, dict *SysDict) error
	}

	customSysDictModel struct {
		*defaultSysDictModel
	}
)

type DictListFilter struct {
	Name   string
	Code   string
	Status int64
}

type DictRow struct {
	Id          int64  `db:"id"`
	Name        string `db:"name"`
	Code        string `db:"code"`
	Items       string `db:"items"`
	Description string `db:"description"`
	Status      int64  `db:"status"`
	CreatedAt   int64  `db:"created_at"`
	UpdatedAt   int64  `db:"updated_at"`
}

// NewSysDictModel returns a model for the database table.
func NewSysDictModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysDictModel {
	return &customSysDictModel{
		defaultSysDictModel: newSysDictModel(conn, c, opts...),
	}
}

func (m *customSysDictModel) ListDictPage(ctx context.Context, filter DictListFilter, page, pageSize int64) ([]*DictRow, int64, error) {
	where := "deleted_at IS NULL"
	var args []any
	if filter.Name != "" {
		where += " AND name LIKE ?"
		args = append(args, "%"+filter.Name+"%")
	}
	if filter.Code != "" {
		where += " AND code LIKE ?"
		args = append(args, "%"+filter.Code+"%")
	}
	if filter.Status > 0 {
		where += " AND status=?"
		args = append(args, filter.Status)
	}
	var total int64
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) FROM sys_dict WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var rows []*DictRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id,name,code,items,description,status,UNIX_TIMESTAMP(created_at) AS created_at,UNIX_TIMESTAMP(updated_at) AS updated_at FROM sys_dict WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...)
	return rows, total, err
}

func (m *customSysDictModel) SoftDelete(ctx context.Context, id int64) error {
	var code string
	found := false
	err := m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		err := session.QueryRowCtx(ctx, &code, "SELECT code FROM sys_dict WHERE id=? FOR UPDATE", id)
		if errors.Is(err, sqlx.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		_, err = session.ExecCtx(ctx, "UPDATE sys_dict SET deleted_at=NOW() WHERE id=? AND deleted_at IS NULL", id)
		return err
	})
	if err != nil {
		return err
	}
	keys := []string{fmt.Sprintf("%s%d", cacheGozeroAdminNewSysDictIdPrefix, id)}
	if found {
		keys = append(keys, cacheGozeroAdminNewSysDictCodePrefix+code)
	}
	return m.DelCacheCtx(ctx, keys...)
}

func (m *customSysDictModel) UpdateActive(ctx context.Context, dict *SysDict) error {
	var current SysDict
	err := m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := session.QueryRowCtx(ctx, &current, "SELECT "+sysDictRows+" FROM sys_dict WHERE id=? AND deleted_at IS NULL FOR UPDATE", dict.Id); err != nil {
			return err
		}
		// 不写回查询快照中的 deleted_at，避免并发删除后恢复记录。
		_, err := session.ExecCtx(ctx, "UPDATE sys_dict SET name=?,code=?,items=?,description=?,status=? WHERE id=? AND deleted_at IS NULL",
			dict.Name, dict.Code, dict.Items, dict.Description, dict.Status, dict.Id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysDictIdPrefix, dict.Id),
		cacheGozeroAdminNewSysDictCodePrefix+current.Code, cacheGozeroAdminNewSysDictCodePrefix+dict.Code)
}
