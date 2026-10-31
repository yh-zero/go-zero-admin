package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

type GetConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetConversationLogic {
	return &GetConversationLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *GetConversationLogic) GetConversation(req *types.AiIdReq) (*types.AiGetConversationResp, error) {
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.GetConversation(l.ctx, &aipb.GetConversationRequest{Actor: actor, ConversationId: req.Id})
	if err != nil {
		return nil, err
	}
	messages := make([]*types.AiChatMessage, 0, len(out.Messages))
	for _, item := range out.Messages {
		messages = append(messages, mapAiMessage(item))
	}
	return &types.AiGetConversationResp{Conversation: mapAiConversation(out.Conversation), Messages: messages}, nil
}
