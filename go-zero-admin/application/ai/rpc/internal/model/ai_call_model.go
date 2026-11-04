package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type CallPair struct {
	ConversationId  int64
	User, Assistant *MessageRow
}

// ErrCallState 包括已完成或被其他请求领取的调用，禁止自动重放外部调用。
var ErrCallState = errors.New("call is not running")

// ErrCallCommitted 表示数据库已提交，缓存失效失败；调用者不能继续外部执行。
var ErrCallCommitted = errors.New("call committed but cache invalidation failed")

func (m *customAiMessageModel) FindCall(ctx context.Context, ownerId int64, callId string) (*CallPair, error) {
	var rows []*MessageRow
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, messageSelect+" WHERE m.call_id=? AND c.user_id=? AND c.deleted_at IS NULL ORDER BY m.id", callId, ownerId); err != nil {
		return nil, err
	}
	if len(rows) != 2 {
		return nil, ErrNotFound
	}
	pair := &CallPair{}
	for _, r := range rows {
		if r.Role == "user" {
			pair.User = r
		}
		if r.Role == "assistant" {
			pair.Assistant = r
		}
	}
	if pair.User == nil || pair.Assistant == nil || pair.User.ConversationId != pair.Assistant.ConversationId {
		return nil, ErrCallState
	}
	pair.ConversationId = pair.User.ConversationId
	return pair, nil
}

func (m *customAiMessageModel) ReserveCall(ctx context.Context, ownerId, conversationId int64, callId, content, metadata string) (*CallPair, bool, error) {
	// 同 ID 首次预留会竞争唯一索引，InnoDB 可能选择其中一个事务作死锁牺牲者。
	// 此阶段尚未调用外部模型/工具，只重试完整已回滚的预留事务。
	for attempt := 0; attempt < 3; attempt++ {
		pair, fresh, err := m.reserveCallOnce(ctx, ownerId, conversationId, callId, content, metadata)
		var conflict *mysql.MySQLError
		if !errors.As(err, &conflict) || (conflict.Number != 1213 && conflict.Number != 1205) || attempt == 2 {
			return pair, fresh, err
		}
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
	}
	return nil, false, ErrCallState
}

func (m *customAiMessageModel) reserveCallOnce(ctx context.Context, ownerId, conversationId int64, callId, content, metadata string) (*CallPair, bool, error) {
	if pair, err := m.FindCall(ctx, ownerId, callId); err == nil {
		return pair, false, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}
	pair := &CallPair{ConversationId: conversationId}
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if pair.ConversationId == 0 {
			title := []rune(strings.TrimSpace(content))
			if len(title) > 64 {
				title = title[:64]
			}
			result, err := session.ExecCtx(ctx, "INSERT INTO ai_conversation (user_id,title) VALUES (?,?)", ownerId, string(title))
			if err != nil {
				return err
			}
			pair.ConversationId, err = result.LastInsertId()
			if err != nil {
				return err
			}
		}
		var owner struct {
			Id int64 `db:"id"`
		}
		if err := session.QueryRowCtx(ctx, &owner, "SELECT id FROM ai_conversation WHERE id=? AND user_id=? AND deleted_at IS NULL FOR UPDATE", pair.ConversationId, ownerId); err != nil {
			return err
		}
		now := time.Now().Unix()
		for _, role := range []string{"user", "assistant"} {
			text := ""
			if role == "user" {
				text = content
			}
			result, err := session.ExecCtx(ctx, "INSERT INTO ai_message (conversation_id,role,content,tool_calls,tokens,status,call_id) VALUES (?,?,?,?,0,'running',?)", pair.ConversationId, role, text, metadata, callId)
			if err != nil {
				return err
			}
			id, err := result.LastInsertId()
			if err != nil {
				return err
			}
			row := &MessageRow{Id: id, ConversationId: pair.ConversationId, CallId: callId, Status: "running", Role: role, Content: text, ToolCalls: metadata, CreatedAt: now}
			if role == "user" {
				pair.User = row
			} else {
				pair.Assistant = row
			}
		}
		_, err := session.ExecCtx(ctx, "UPDATE ai_conversation SET updated_at=NOW() WHERE id=?", pair.ConversationId)
		return err
	})
	if err != nil {
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			pair, err = m.FindCall(ctx, ownerId, callId)
			return pair, false, err
		}
		return nil, false, err
	}
	if err := m.invalidatePair(ctx, pair); err != nil {
		return pair, false, ErrCallCommitted
	}
	return pair, true, nil
}

func (m *customAiMessageModel) SaveCallUsage(ctx context.Context, ownerId int64, pair *CallPair, tokens int64, metadata string) error {
	return m.mutateCall(ctx, ownerId, pair, func(ctx context.Context, session sqlx.Session) error {
		_, err := session.ExecCtx(ctx, "UPDATE ai_message SET tokens=?,tool_calls=? WHERE id=? AND status='running'", tokens, metadata, pair.Assistant.Id)
		return err
	})
}

func (m *customAiMessageModel) FinishCall(ctx context.Context, ownerId int64, pair *CallPair, status, answer, metadata string, tokens int64) error {
	if status != "failed" && status != "succeeded" {
		return ErrCallState
	}
	if status == "failed" {
		answer = ""
	}
	return m.mutateCall(ctx, ownerId, pair, func(ctx context.Context, session sqlx.Session) error {
		result, err := session.ExecCtx(ctx, "UPDATE ai_message SET status=?,tool_calls=? WHERE id=? AND status='running'", status, metadata, pair.User.Id)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrCallState
		}
		result, err = session.ExecCtx(ctx, "UPDATE ai_message SET status=?,content=?,tokens=?,tool_calls=? WHERE id=? AND status='running'", status, answer, tokens, metadata, pair.Assistant.Id)
		if err != nil {
			return err
		}
		n, err = result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrCallState
		}
		_, err = session.ExecCtx(ctx, "UPDATE ai_conversation SET updated_at=NOW() WHERE id=?", pair.ConversationId)
		return err
	})
}

func (m *customAiMessageModel) mutateCall(ctx context.Context, ownerId int64, pair *CallPair, fn func(context.Context, sqlx.Session) error) error {
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		// 按会话再消息的固定锁序，与预留/删除串行；外部模型与工具调用均不在事务中。
		var owner struct {
			Id int64 `db:"id"`
		}
		// 已预留调用的结算只验证owner；软删除不应丢弃已经取得的外部用量或复活会话。
		if err := session.QueryRowCtx(ctx, &owner, "SELECT id FROM ai_conversation WHERE id=? AND user_id=? FOR UPDATE", pair.ConversationId, ownerId); err != nil {
			return err
		}
		var rows []struct {
			Id int64 `db:"id"`
		}
		if err := session.QueryRowsCtx(ctx, &rows, "SELECT id FROM ai_message WHERE id IN (?,?) AND conversation_id=? AND call_id=? AND status='running' ORDER BY id FOR UPDATE", pair.User.Id, pair.Assistant.Id, pair.ConversationId, pair.Assistant.CallId); err != nil {
			return err
		}
		if len(rows) != 2 {
			return ErrCallState
		}
		return fn(ctx, session)
	})
	if err != nil {
		return err
	}
	if err := m.invalidatePair(ctx, pair); err != nil {
		return ErrCallCommitted
	}
	return nil
}

func (m *customAiMessageModel) invalidatePair(ctx context.Context, pair *CallPair) error {
	keys := []string{fmt.Sprintf("%s%d", cacheGozeroAdminAiConversationIdPrefix, pair.ConversationId)}
	for _, r := range []*MessageRow{pair.User, pair.Assistant} {
		keys = append(keys, fmt.Sprintf("%s%d", cacheGozeroAdminAiMessageIdPrefix, r.Id), fmt.Sprintf("%s%v:%v", cacheGozeroAdminAiMessageCallIdRolePrefix, sql.NullString{String: r.CallId, Valid: true}, r.Role))
	}
	return m.DelCacheCtx(ctx, keys...)
}
