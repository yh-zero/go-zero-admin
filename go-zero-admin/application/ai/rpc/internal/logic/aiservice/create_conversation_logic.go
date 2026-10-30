package aiservicelogic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"go-zero-admin/pkg/xerr"
)

type CreateConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateConversationLogic {
	return &CreateConversationLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateConversationLogic) CreateConversation(in *aipb.CreateConversationRequest) (*aipb.CreateConversationResponse, error) {
	title := in.Title
	if title == "" {
		title = "新会话"
	}
	now := time.Now()
	result, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"INSERT INTO ai_conversation (user_id, title, created_at, updated_at) VALUES (?,?,?,?)",
		in.Actor.UserId, title, now, now)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	id, _ := result.LastInsertId()
	return &aipb.CreateConversationResponse{Conversation: &aipb.ConversationInfo{
		Id: id, Title: title, UserId: in.Actor.UserId, CreatedAt: now.Unix(), UpdatedAt: now.Unix(),
	}}, nil
}
