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
		ListPageOwned(ctx context.Context, conversationId, ownerId, beforeId, limit int64) ([]*MessageRow, error)
		FindCall(ctx context.Context, ownerId int64, callId string) (*CallPair, error)
		ReserveCall(ctx context.Context, ownerId, conversationId int64, callId, content, metadata string) (*CallPair, bool, error)
		SaveCallUsage(ctx context.Context, ownerId int64, pair *CallPair, tokens int64, metadata string) error
		FinishCall(ctx context.Context, ownerId int64, pair *CallPair, status, answer, metadata string, tokens int64) error
		SetFeedbackOwned(ctx context.Context, ownerId, conversationId, messageId int64, feedback, note string) error
	}

	customAiMessageModel struct {
		*defaultAiMessageModel
		conn sqlx.SqlConn
	}
)

// NewAiMessageModel returns a model for the database table.
func NewAiMessageModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiMessageModel {
	return &customAiMessageModel{
		defaultAiMessageModel: newAiMessageModel(conn, c, opts...),
		conn:                  conn,
	}
}

type MessageRow struct {
	ConversationId int64  `db:"conversation_id"`
	CallId         string `db:"call_id"`
	Status         string `db:"status"`
	Id             int64  `db:"id"`
	Role           string `db:"role"`
	Content        string `db:"content"`
	ToolCalls      string `db:"tool_calls"`
	Tokens         int64  `db:"tokens"`
	CreatedAt      int64  `db:"created_at"`
}

func (m *customAiMessageModel) ListOwned(ctx context.Context, conversationId, ownerId int64, limit int64) ([]*MessageRow, error) {
	rows := make([]*MessageRow, 0)
	query := messageSelect + " WHERE c.id=? AND c.user_id=? AND c.deleted_at IS NULL AND COALESCE(m.status,'succeeded')='succeeded' AND m.role IN ('user','assistant') ORDER BY m.id"
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

const messageSelect = "SELECT m.id,m.conversation_id,COALESCE(m.call_id,'') AS call_id,COALESCE(m.status,'succeeded') AS status,m.role,m.content,COALESCE(CAST(m.tool_calls AS CHAR),'[]') AS tool_calls,m.tokens,UNIX_TIMESTAMP(m.created_at) AS created_at FROM ai_message m JOIN ai_conversation c ON c.id=m.conversation_id"

// ListPageOwned 只读取有界最新页，接口层按预算选择后再以正序返回。
func (m *customAiMessageModel) ListPageOwned(ctx context.Context, conversationId, ownerId, beforeId, limit int64) ([]*MessageRow, error) {
	rows := make([]*MessageRow, 0)
	query := messageSelect + " WHERE c.id=? AND c.user_id=? AND c.deleted_at IS NULL"
	args := []any{conversationId, ownerId}
	if beforeId > 0 {
		query += " AND m.id<?"
		args = append(args, beforeId)
	}
	query += " ORDER BY m.id DESC LIMIT ?"
	args = append(args, limit)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, query, args...)
	return rows, err
}
