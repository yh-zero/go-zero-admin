package ai

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

type SendMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendMessageLogic {
	return &SendMessageLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *SendMessageLogic) SendMessage(req *types.AiSendMessageReq) (*types.AiSendMessageResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.SendMessage(l.ctx, &aipb.SendMessageRequest{Actor: actor, ConversationId: req.ConversationId, Content: req.Content, RequestId: req.RequestId})
	if err != nil {
		return nil, err
	}
	return &types.AiSendMessageResp{ConversationId: out.ConversationId, UserMessage: mapAiMessage(out.UserMessage), AiMessage: mapAiMessage(out.AiMessage)}, nil
}
