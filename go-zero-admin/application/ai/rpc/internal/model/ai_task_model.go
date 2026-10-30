package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiTaskModel = (*customAiTaskModel)(nil)

type (
	// AiTaskModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiTaskModel.
	AiTaskModel interface {
		aiTaskModel
	}

	customAiTaskModel struct {
		*defaultAiTaskModel
	}
)

// NewAiTaskModel returns a model for the database table.
func NewAiTaskModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiTaskModel {
	return &customAiTaskModel{
		defaultAiTaskModel: newAiTaskModel(conn, c, opts...),
	}
}
