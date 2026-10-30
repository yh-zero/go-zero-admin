package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiWorkflowModel = (*customAiWorkflowModel)(nil)

type (
	// AiWorkflowModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiWorkflowModel.
	AiWorkflowModel interface {
		aiWorkflowModel
	}

	customAiWorkflowModel struct {
		*defaultAiWorkflowModel
	}
)

// NewAiWorkflowModel returns a model for the database table.
func NewAiWorkflowModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiWorkflowModel {
	return &customAiWorkflowModel{
		defaultAiWorkflowModel: newAiWorkflowModel(conn, c, opts...),
	}
}
