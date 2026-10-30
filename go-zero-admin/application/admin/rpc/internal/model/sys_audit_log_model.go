package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysAuditLogModel = (*customSysAuditLogModel)(nil)

type (
	// SysAuditLogModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysAuditLogModel.
	SysAuditLogModel interface {
		sysAuditLogModel
	}

	customSysAuditLogModel struct {
		*defaultSysAuditLogModel
	}
)

// NewSysAuditLogModel returns a model for the database table.
func NewSysAuditLogModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysAuditLogModel {
	return &customSysAuditLogModel{
		defaultSysAuditLogModel: newSysAuditLogModel(conn, c, opts...),
	}
}
