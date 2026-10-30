package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"go-zero-admin/pkg/xerr"
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
	_, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"UPDATE ai_conversation SET deleted_at=NOW() WHERE id=? AND user_id=?", in.ConversationId, in.Actor.UserId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &aipb.NoDataResponse{}, nil
}
