package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysMenuLockModel = (*customSysMenuLockModel)(nil)

type (
	// SysMenuLockModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysMenuLockModel.
	SysMenuLockModel interface {
		sysMenuLockModel
	}

	customSysMenuLockModel struct {
		*defaultSysMenuLockModel
	}
)

// NewSysMenuLockModel returns a model for the database table.
func NewSysMenuLockModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysMenuLockModel {
	return &customSysMenuLockModel{
		defaultSysMenuLockModel: newSysMenuLockModel(conn, c, opts...),
	}
}
