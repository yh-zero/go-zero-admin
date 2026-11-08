package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiUsageCallModel = (*customAiUsageCallModel)(nil)

type (
	// AiUsageCallModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiUsageCallModel.
	AiUsageCallModel interface {
		aiUsageCallModel
	}

	customAiUsageCallModel struct {
		*defaultAiUsageCallModel
	}
)

// NewAiUsageCallModel returns a model for the database table.
func NewAiUsageCallModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiUsageCallModel {
	return &customAiUsageCallModel{
		defaultAiUsageCallModel: newAiUsageCallModel(conn, c, opts...),
	}
}
