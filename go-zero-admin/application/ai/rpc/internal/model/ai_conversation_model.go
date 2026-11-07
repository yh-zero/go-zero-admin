package model

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"strings"
	"time"
)

var _ AiConversationModel = (*customAiConversationModel)(nil)

type (
	// AiConversationModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiConversationModel.
	AiConversationModel interface {
		aiConversationModel
		CheckOwned(ctx context.Context, id, ownerId int64) error
		FindOwned(ctx context.Context, id, ownerId int64) (*ConversationRow, error)
		ListOwned(ctx context.Context, ownerId, page, pageSize int64) ([]*ConversationRow, int64, error)
		SearchOwned(ctx context.Context, ownerId, page, pageSize int64, query string) ([]*ConversationRow, int64, error)
		UpdateOwned(ctx context.Context, id, ownerId int64, title *string, pinned *bool) error
		DeleteOwned(ctx context.Context, id, ownerId int64) error
		AppendTurn(ctx context.Context, id, ownerId int64, userContent, assistantContent, toolCalls string, tokens int64) (int64, int64, error)
	}

	customAiConversationModel struct {
		*defaultAiConversationModel
		conn sqlx.SqlConn
	}
)

// NewAiConversationModel returns a model for the database table.
func NewAiConversationModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiConversationModel {
	return &customAiConversationModel{
		defaultAiConversationModel: newAiConversationModel(conn, c, opts...),
		conn:                       conn,
	}
}

type ConversationRow struct {
	Id           int64  `db:"id"`
	UserId       int64  `db:"user_id"`
	Title        string `db:"title"`
	Pinned       bool   `db:"pinned"`
	MessageCount int64  `db:"message_count"`
	CreatedAt    int64  `db:"created_at"`
	UpdatedAt    int64  `db:"updated_at"`
	Deleted      bool   `db:"deleted"`
}

const conversationSelect = "SELECT c.id,c.user_id,c.title,c.pinned,(SELECT COUNT(*) FROM ai_message m WHERE m.conversation_id=c.id) AS message_count,UNIX_TIMESTAMP(c.created_at) AS created_at,UNIX_TIMESTAMP(c.updated_at) AS updated_at,c.deleted_at IS NOT NULL AS deleted FROM ai_conversation c "

func (m *customAiConversationModel) CheckOwned(ctx context.Context, id, ownerId int64) error {
	var ownedId int64
	return m.QueryRowNoCacheCtx(ctx, &ownedId, "SELECT id FROM ai_conversation WHERE id=? AND user_id=? AND deleted_at IS NULL", id, ownerId)
}

func (m *customAiConversationModel) FindOwned(ctx context.Context, id, ownerId int64) (*ConversationRow, error) {
	var row ConversationRow
	err := m.QueryRowNoCacheCtx(ctx, &row, conversationSelect+"WHERE c.id=? AND c.user_id=? AND c.deleted_at IS NULL", id, ownerId)
	return &row, err
}

func (m *customAiConversationModel) ListOwned(ctx context.Context, ownerId, page, pageSize int64) ([]*ConversationRow, int64, error) {
	return m.SearchOwned(ctx, ownerId, page, pageSize, "")
}

func (m *customAiConversationModel) SearchOwned(ctx context.Context, ownerId, page, pageSize int64, query string) ([]*ConversationRow, int64, error) {
	where := " WHERE c.user_id=? AND c.deleted_at IS NULL"
	args := []any{ownerId}
	if query != "" {
		pattern := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(query) + "%"
		where += " AND (c.title LIKE ? ESCAPE '!' OR EXISTS (SELECT 1 FROM ai_message search_message WHERE search_message.conversation_id=c.id AND search_message.content LIKE ? ESCAPE '!'))"
		args = append(args, pattern, pattern)
	}
	var count struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM ai_conversation c"+where, args...); err != nil {
		return nil, 0, err
	}
	rows := make([]*ConversationRow, 0)
	args = append(args, pageSize, (page-1)*pageSize)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, conversationSelect+where+" ORDER BY c.pinned DESC,c.updated_at DESC,c.id DESC LIMIT ? OFFSET ?", args...)
	return rows, count.Total, err
}

func (m *customAiConversationModel) UpdateOwned(ctx context.Context, id, ownerId int64, title *string, pinned *bool) error {
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var owned int64
		if err := session.QueryRowCtx(ctx, &owned, "SELECT id FROM ai_conversation WHERE id=? AND user_id=? AND deleted_at IS NULL FOR UPDATE", id, ownerId); err != nil {
			return err
		}
		sets := make([]string, 0, 2)
		args := make([]any, 0, 3)
		if title != nil {
			sets = append(sets, "title=?")
			args = append(args, *title)
		}
		if pinned != nil {
			sets = append(sets, "pinned=?")
			args = append(args, *pinned)
		}
		if len(sets) == 0 {
			return nil
		}
		args = append(args, id)
		_, err := session.ExecCtx(ctx, "UPDATE ai_conversation SET "+strings.Join(sets, ",")+" WHERE id=?", args...)
		return err
	})
	if err != nil {
		return err
	}
	if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiConversationIdPrefix, id)); err != nil {
		return ErrCallCommitted
	}
	return nil
}

func (m *customAiConversationModel) DeleteOwned(ctx context.Context, id, ownerId int64) error {
	result, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, "UPDATE ai_conversation SET deleted_at=NOW() WHERE id=? AND user_id=? AND deleted_at IS NULL", id, ownerId)
	}, fmt.Sprintf("%s%d", cacheGozeroAdminAiConversationIdPrefix, id))
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// AppendTurn 将本轮两条消息一起提交；模型失败不留下半轮对话。
func (m *customAiConversationModel) AppendTurn(ctx context.Context, id, ownerId int64, userContent, assistantContent, toolCalls string, tokens int64) (int64, int64, error) {
	var userId, assistantId int64
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var owner struct {
			Id int64 `db:"id"`
		}
		if err := session.QueryRowCtx(ctx, &owner, "SELECT id FROM ai_conversation WHERE id=? AND user_id=? AND deleted_at IS NULL FOR UPDATE", id, ownerId); err != nil {
			return err
		}
		result, err := session.ExecCtx(ctx, "INSERT INTO ai_message (conversation_id,role,content) VALUES (?,'user',?)", id, userContent)
		if err != nil {
			return err
		}
		userId, err = result.LastInsertId()
		if err != nil {
			return err
		}
		result, err = session.ExecCtx(ctx, "INSERT INTO ai_message (conversation_id,role,content,tool_calls,tokens) VALUES (?,'assistant',?,?,?)", id, assistantContent, toolCalls, tokens)
		if err != nil {
			return err
		}
		assistantId, err = result.LastInsertId()
		if err != nil {
			return err
		}
		_, err = session.ExecCtx(ctx, "UPDATE ai_conversation SET updated_at=? WHERE id=?", time.Now(), id)
		return err
	})
	if err != nil {
		return 0, 0, err
	}
	err = m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiConversationIdPrefix, id))
	return userId, assistantId, err
}
