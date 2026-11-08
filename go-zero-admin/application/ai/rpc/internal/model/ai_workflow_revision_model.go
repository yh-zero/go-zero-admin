package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiWorkflowRevisionModel = (*customAiWorkflowRevisionModel)(nil)

type (
	// AiWorkflowRevisionModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiWorkflowRevisionModel.
	AiWorkflowRevisionModel interface {
		aiWorkflowRevisionModel
	}

	customAiWorkflowRevisionModel struct {
		*defaultAiWorkflowRevisionModel
	}
)

// NewAiWorkflowRevisionModel returns a model for the database table.
func NewAiWorkflowRevisionModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiWorkflowRevisionModel {
	return &customAiWorkflowRevisionModel{
		defaultAiWorkflowRevisionModel: newAiWorkflowRevisionModel(conn, c, opts...),
	}
}
