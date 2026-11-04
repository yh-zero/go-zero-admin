package aiservicelogic

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
	"google.golang.org/protobuf/proto"
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
	if in.ConversationId <= 0 || in.BeforeId < 0 || in.PageSize < 0 || in.PageSize > 20 {
		return nil, xerr.NewErrCodeMsg(400, "会话游标或分页大小不合法")
	}
	size := in.PageSize
	if size == 0 {
		size = 20
	}
	row, err := l.svcCtx.AiConversationModel.FindOwned(l.ctx, in.ConversationId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	rows, err := l.svcCtx.AiMessageModel.ListPageOwned(l.ctx, in.ConversationId, in.Actor.UserId, in.BeforeId, size+1)
	if err != nil {
		return nil, aiModelError(err)
	}
	out := &aipb.GetConversationResponse{Conversation: conversationProto(row), Messages: make([]*aipb.ChatMessage, 0, len(rows))}
	// 同时限制 JSON 与 protobuf，留出 2 MiB 余量给默认 4 MiB RPC 上限及封装开销。
	for i, item := range rows {
		if int64(i) >= size {
			out.HasMore = true
			break
		}
		message := messageProto(item)
		out.Messages = append(out.Messages, message)
		data, err := json.Marshal(out)
		if err != nil || len(data) > 2<<20 || proto.Size(out) > 2<<20 {
			out.Messages = out.Messages[:len(out.Messages)-1]
			if len(out.Messages) == 0 {
				return nil, xerr.NewErrCodeMsg(503, "历史消息超过单页大小限制，请联系管理员处理")
			}
			out.HasMore = true
			break
		}
	}
	if out.HasMore {
		out.NextBeforeId = out.Messages[len(out.Messages)-1].Id
	}
	for i, j := 0, len(out.Messages)-1; i < j; i, j = i+1, j-1 {
		out.Messages[i], out.Messages[j] = out.Messages[j], out.Messages[i]
	}
	return out, nil
}
