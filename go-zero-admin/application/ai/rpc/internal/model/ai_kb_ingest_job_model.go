package model

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiKbIngestJobModel = (*customAiKbIngestJobModel)(nil)

type (
	// AiKbIngestJobModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiKbIngestJobModel.
	AiKbIngestJobModel interface {
		aiKbIngestJobModel
	}

	customAiKbIngestJobModel struct {
		*defaultAiKbIngestJobModel
	}
)

// NewAiKbIngestJobModel returns a model for the database table.
func NewAiKbIngestJobModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiKbIngestJobModel {
	return &customAiKbIngestJobModel{
		defaultAiKbIngestJobModel: newAiKbIngestJobModel(conn, c, opts...),
	}
}
