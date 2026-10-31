package aiservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
	"strings"
	"time"
	"unicode/utf8"
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
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		title = "新会话"
	}
	if utf8.RuneCountInString(title) > 128 {
		return nil, xerr.NewErrCodeMsg(400, "会话标题最多 128 字符")
	}
	result, err := l.svcCtx.AiConversationModel.Insert(l.ctx, &model.AiConversation{UserId: uint64(in.Actor.UserId), Title: title})
	if err != nil {
		return nil, aiModelError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, aiModelError(err)
	}
	now := time.Now().Unix()
	return &aipb.CreateConversationResponse{Conversation: &aipb.ConversationInfo{Id: id, Title: title, UserId: in.Actor.UserId, CreatedAt: now, UpdatedAt: now}}, nil
}
