package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiKbIngestBatchModel = (*customAiKbIngestBatchModel)(nil)

type (
	// AiKbIngestBatchModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiKbIngestBatchModel.
	AiKbIngestBatchModel interface {
		aiKbIngestBatchModel
	}

	customAiKbIngestBatchModel struct {
		*defaultAiKbIngestBatchModel
	}
)

// NewAiKbIngestBatchModel returns a model for the database table.
func NewAiKbIngestBatchModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiKbIngestBatchModel {
	return &customAiKbIngestBatchModel{
		defaultAiKbIngestBatchModel: newAiKbIngestBatchModel(conn, c, opts...),
	}
}
