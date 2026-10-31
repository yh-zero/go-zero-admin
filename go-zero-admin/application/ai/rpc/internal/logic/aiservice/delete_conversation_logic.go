package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
)

type DeleteConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteConversationLogic {
	return &DeleteConversationLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *DeleteConversationLogic) DeleteConversation(in *aipb.DeleteConversationRequest) (*aipb.NoDataResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	if err := l.svcCtx.AiConversationModel.DeleteOwned(l.ctx, in.ConversationId, in.Actor.UserId); err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.NoDataResponse{}, nil
}
