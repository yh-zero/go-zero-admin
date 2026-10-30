package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiMessageModel = (*customAiMessageModel)(nil)

type (
	// AiMessageModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiMessageModel.
	AiMessageModel interface {
		aiMessageModel
	}

	customAiMessageModel struct {
		*defaultAiMessageModel
	}
)

// NewAiMessageModel returns a model for the database table.
func NewAiMessageModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiMessageModel {
	return &customAiMessageModel{
		defaultAiMessageModel: newAiMessageModel(conn, c, opts...),
	}
}
