package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiConversationModel = (*customAiConversationModel)(nil)

type (
	// AiConversationModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiConversationModel.
	AiConversationModel interface {
		aiConversationModel
	}

	customAiConversationModel struct {
		*defaultAiConversationModel
	}
)

// NewAiConversationModel returns a model for the database table.
func NewAiConversationModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiConversationModel {
	return &customAiConversationModel{
		defaultAiConversationModel: newAiConversationModel(conn, c, opts...),
	}
}
