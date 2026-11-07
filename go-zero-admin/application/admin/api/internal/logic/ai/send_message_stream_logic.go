package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"
)

type SendMessageStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendMessageStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendMessageStreamLogic {
	return &SendMessageStreamLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// SendMessageStream 校验并建立到 ai-rpc 的服务端流；帧编码与心跳由 handler 负责（协议处理留在 handler）。
func (l *SendMessageStreamLogic) SendMessageStream(req *types.AiSendMessageReq) (aipb.AiService_SendMessageStreamClient, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	stream, err := l.svcCtx.AiRpc.SendMessageStream(l.ctx, &aipb.SendMessageRequest{Actor: actor, ConversationId: req.ConversationId, Content: req.Content, RequestId: req.RequestId, KbId: req.KbId, Grounded: req.Grounded})
	if err != nil {
		return nil, err
	}
	return stream, nil
}

// MapSendResult 保留 API 字段的零值与 null，不直接暴露 protobuf 的省略字段规则。
func MapSendResult(result *aipb.SendMessageResponse) *types.AiSendMessageResp {
	if result == nil {
		return nil
	}
	return &types.AiSendMessageResp{
		ConversationId: result.ConversationId,
		Conversation:   mapAiConversation(result.Conversation),
		UserMessage:    mapAiMessage(result.UserMessage),
		AiMessage:      mapAiMessage(result.AiMessage),
	}
}
