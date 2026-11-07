package model

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// SetFeedbackOwned 只更新已完成助手消息中的反馈，保留真实来源、用量和请求指纹。
func (m *customAiMessageModel) SetFeedbackOwned(ctx context.Context, ownerId, conversationId, messageId int64, feedback, note string) error {
	var callId sql.NullString
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var parent int64
		if err := session.QueryRowCtx(ctx, &parent, "SELECT id FROM ai_conversation WHERE id=? AND user_id=? AND deleted_at IS NULL FOR UPDATE", conversationId, ownerId); err != nil {
			return err
		}
		var row struct {
			CallId sql.NullString `db:"call_id"`
		}
		if err := session.QueryRowCtx(ctx, &row, "SELECT call_id FROM ai_message WHERE id=? AND conversation_id=? AND role='assistant' AND COALESCE(status,'succeeded')='succeeded' FOR UPDATE", messageId, conversationId); err != nil {
			return err
		}
		callId = row.CallId
		_, err := session.ExecCtx(ctx, "UPDATE ai_message SET tool_calls=JSON_SET(CASE WHEN JSON_TYPE(tool_calls)='OBJECT' THEN tool_calls ELSE JSON_OBJECT('version',1,'calls',COALESCE(tool_calls,JSON_ARRAY())) END,'$.feedback',?,'$.feedback_note',?) WHERE id=?", feedback, note, messageId)
		return err
	})
	if err != nil {
		return err
	}
	if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiMessageIdPrefix, messageId), fmt.Sprintf("%s%v:%v", cacheGozeroAdminAiMessageCallIdRolePrefix, callId, "assistant")); err != nil {
		return ErrCallCommitted
	}
	return nil
}
