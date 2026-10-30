package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TenantRegistryModel = (*customTenantRegistryModel)(nil)

type (
	// TenantRegistryModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTenantRegistryModel.
	TenantRegistryModel interface {
		tenantRegistryModel
	}

	customTenantRegistryModel struct {
		*defaultTenantRegistryModel
	}
)

// NewTenantRegistryModel returns a model for the database table.
func NewTenantRegistryModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TenantRegistryModel {
	return &customTenantRegistryModel{
		defaultTenantRegistryModel: newTenantRegistryModel(conn, c, opts...),
	}
}
