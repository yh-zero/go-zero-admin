package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"go-zero-admin/pkg/xerr"
)

type GetConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationLogic {
	return &GetConversationLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetConversationLogic) GetConversation(in *aipb.GetConversationRequest) (*aipb.GetConversationResponse, error) {
	type convRow struct {
		Id        int64  `db:"id"`
		Title     string `db:"title"`
		CreatedAt int64  `db:"created_at"`
		UpdatedAt int64  `db:"updated_at"`
	}
	var cr convRow
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &cr,
		"SELECT id, title, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM ai_conversation WHERE id=? AND user_id=? AND deleted_at IS NULL",
		in.ConversationId, in.Actor.UserId)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(404, "会话不存在")
	}

	type msgRow struct {
		Id        int64  `db:"id"`
		Role      string `db:"role"`
		Content   string `db:"content"`
		ToolCalls string `db:"tool_calls"`
		Tokens    int64  `db:"tokens"`
		CreatedAt int64  `db:"created_at"`
	}
	var msgRows []msgRow
	_ = l.svcCtx.DB.QueryRowsCtx(l.ctx, &msgRows,
		"SELECT id, role, content, COALESCE(CAST(tool_calls AS CHAR), ''), tokens, UNIX_TIMESTAMP(created_at) FROM ai_message WHERE conversation_id=? ORDER BY id",
		in.ConversationId)
	msgs := make([]*aipb.ChatMessage, 0, len(msgRows))
	for _, r := range msgRows {
		msgs = append(msgs, &aipb.ChatMessage{Id: r.Id, Role: r.Role, Content: r.Content, ToolCalls: r.ToolCalls, Tokens: r.Tokens, CreatedAt: r.CreatedAt})
	}
	return &aipb.GetConversationResponse{
		Conversation: &aipb.ConversationInfo{Id: cr.Id, Title: cr.Title, UserId: in.Actor.UserId, MessageCount: int64(len(msgs)), CreatedAt: cr.CreatedAt, UpdatedAt: cr.UpdatedAt},
		Messages:     msgs,
	}, nil
}
