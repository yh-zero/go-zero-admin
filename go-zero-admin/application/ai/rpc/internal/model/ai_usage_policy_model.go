package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiUsagePolicyModel = (*customAiUsagePolicyModel)(nil)

type (
	// AiUsagePolicyModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiUsagePolicyModel.
	AiUsagePolicyModel interface {
		aiUsagePolicyModel
	}

	customAiUsagePolicyModel struct {
		*defaultAiUsagePolicyModel
	}
)

// NewAiUsagePolicyModel returns a model for the database table.
func NewAiUsagePolicyModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiUsagePolicyModel {
	return &customAiUsagePolicyModel{
		defaultAiUsagePolicyModel: newAiUsagePolicyModel(conn, c, opts...),
	}
}
