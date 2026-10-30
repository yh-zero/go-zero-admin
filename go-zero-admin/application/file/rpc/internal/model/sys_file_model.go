package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysFileModel = (*customSysFileModel)(nil)

type (
	// SysFileModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysFileModel.
	SysFileModel interface {
		sysFileModel
	}

	customSysFileModel struct {
		*defaultSysFileModel
	}
)

// NewSysFileModel returns a model for the database table.
func NewSysFileModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysFileModel {
	return &customSysFileModel{
		defaultSysFileModel: newSysFileModel(conn, c, opts...),
	}
}
