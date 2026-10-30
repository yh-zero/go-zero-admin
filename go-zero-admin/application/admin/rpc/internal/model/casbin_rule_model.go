package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ CasbinRuleModel = (*customCasbinRuleModel)(nil)

type (
	// CasbinRuleModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCasbinRuleModel.
	CasbinRuleModel interface {
		casbinRuleModel
	}

	customCasbinRuleModel struct {
		*defaultCasbinRuleModel
	}
)

// NewCasbinRuleModel returns a model for the database table.
func NewCasbinRuleModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) CasbinRuleModel {
	return &customCasbinRuleModel{
		defaultCasbinRuleModel: newCasbinRuleModel(conn, c, opts...),
	}
}
