package aiservicelogic

import (
	"context"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateConversationLogic {
	return &UpdateConversationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateConversationLogic) UpdateConversation(in *aipb.UpdateConversationRequest) (*aipb.CreateConversationResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	if in.ConversationId <= 0 || in.Title == nil && in.Pinned == nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if in.Title != nil && (!utf8.ValidString(*in.Title) || strings.TrimSpace(*in.Title) == "" || utf8.RuneCountInString(*in.Title) > 128) {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if err := l.svcCtx.AiConversationModel.UpdateOwned(l.ctx, in.ConversationId, in.Actor.UserId, in.Title, in.Pinned); err != nil {
		return nil, aiModelError(err)
	}
	row, err := l.svcCtx.AiConversationModel.FindOwned(l.ctx, in.ConversationId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.CreateConversationResponse{Conversation: conversationProto(row)}, nil
}
