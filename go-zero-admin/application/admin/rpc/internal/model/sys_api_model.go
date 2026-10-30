package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysApiModel = (*customSysApiModel)(nil)

type (
	// SysApiModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysApiModel.
	SysApiModel interface {
		sysApiModel
	}

	customSysApiModel struct {
		*defaultSysApiModel
	}
)

// NewSysApiModel returns a model for the database table.
func NewSysApiModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysApiModel {
	return &customSysApiModel{
		defaultSysApiModel: newSysApiModel(conn, c, opts...),
	}
}
