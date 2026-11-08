package model

import (
	"context"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiKbEvaluationModel = (*customAiKbEvaluationModel)(nil)

type (
	// AiKbEvaluationModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiKbEvaluationModel.
	AiKbEvaluationModel interface {
		aiKbEvaluationModel
		CreateOwned(context.Context, *KbEvaluationRow) (*KbEvaluationRow, bool, error)
		FindRequestOwned(context.Context, int64, string) (*KbEvaluationRow, error)
		FindOwned(context.Context, int64, int64, int64) (*KbEvaluationRow, error)
		ListOwned(context.Context, int64, int64, int64, int64, string, string) ([]*KbEvaluationRow, int64, error)
		ActivateOwned(context.Context, int64, int64, int64, string) (*KbEvaluationRow, error)
		HeartbeatOwned(context.Context, int64, int64, int64, string) error
		TryStartCase(context.Context, int64, int64, int64, int64, string) (*KbEvaluationRow, error)
		SaveCase(context.Context, int64, int64, int64, int64, string, EvaluationObservation) (*KbEvaluationRow, error)
		FinishOwned(context.Context, int64, int64, int64, string, string, bool, string) (*KbEvaluationRow, error)
		StopOwned(context.Context, int64, int64, int64) (*KbEvaluationRow, error)
		ExpireOwned(context.Context, int64, int64, int64) (*KbEvaluationRow, error)
		ReadCorpusOwned(context.Context, int64, int64) ([]EvaluationCorpusDocument, error)
	}

	customAiKbEvaluationModel struct {
		*defaultAiKbEvaluationModel
		conn sqlx.SqlConn
	}
)

// NewAiKbEvaluationModel returns a model for the database table.
func NewAiKbEvaluationModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiKbEvaluationModel {
	return &customAiKbEvaluationModel{
		defaultAiKbEvaluationModel: newAiKbEvaluationModel(conn, c, opts...),
		conn:                       conn,
	}
}
