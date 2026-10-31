package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
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
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	row, err := l.svcCtx.AiConversationModel.FindOwned(l.ctx, in.ConversationId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	rows, err := l.svcCtx.AiMessageModel.ListOwned(l.ctx, in.ConversationId, in.Actor.UserId, 0)
	if err != nil {
		return nil, aiModelError(err)
	}
	messages := make([]*aipb.ChatMessage, 0, len(rows))
	for _, item := range rows {
		messages = append(messages, messageProto(item))
	}
	return &aipb.GetConversationResponse{Conversation: conversationProto(row), Messages: messages}, nil
}
