package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TenantProvisionEventModel = (*customTenantProvisionEventModel)(nil)

type (
	// TenantProvisionEventModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTenantProvisionEventModel.
	TenantProvisionEventModel interface {
		tenantProvisionEventModel
	}

	customTenantProvisionEventModel struct {
		*defaultTenantProvisionEventModel
	}
)

// NewTenantProvisionEventModel returns a model for the database table.
func NewTenantProvisionEventModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TenantProvisionEventModel {
	return &customTenantProvisionEventModel{
		defaultTenantProvisionEventModel: newTenantProvisionEventModel(conn, c, opts...),
	}
}
