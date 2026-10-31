package model

import (
	"context"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiMessageModel = (*customAiMessageModel)(nil)

type (
	// AiMessageModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiMessageModel.
	AiMessageModel interface {
		aiMessageModel
		ListOwned(ctx context.Context, conversationId, ownerId int64, limit int64) ([]*MessageRow, error)
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

type MessageRow struct {
	Id        int64  `db:"id"`
	Role      string `db:"role"`
	Content   string `db:"content"`
	ToolCalls string `db:"tool_calls"`
	Tokens    int64  `db:"tokens"`
	CreatedAt int64  `db:"created_at"`
}

func (m *customAiMessageModel) ListOwned(ctx context.Context, conversationId, ownerId int64, limit int64) ([]*MessageRow, error) {
	rows := make([]*MessageRow, 0)
	query := "SELECT m.id,m.role,m.content,COALESCE(CAST(m.tool_calls AS CHAR),'[]') AS tool_calls,m.tokens,UNIX_TIMESTAMP(m.created_at) AS created_at FROM ai_message m JOIN ai_conversation c ON c.id=m.conversation_id WHERE c.id=? AND c.user_id=? AND c.deleted_at IS NULL ORDER BY m.id"
	args := []any{conversationId, ownerId}
	if limit > 0 {
		query += " DESC LIMIT ?"
		args = append(args, limit)
	}
	err := m.QueryRowsNoCacheCtx(ctx, &rows, query, args...)
	if limit > 0 {
		for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
			rows[i], rows[j] = rows[j], rows[i]
		}
	}
	return rows, err
}
