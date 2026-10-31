package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

type ListConversationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListConversationsLogic {
	return &ListConversationsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *ListConversationsLogic) ListConversations(req *types.AiPageReq) (*types.AiListConversationsResp, error) {
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ListConversations(l.ctx, &aipb.ListConversationsRequest{Actor: actor, Page: &aipb.PageRequest{Page: req.Page, PageSize: req.PageSize}})
	if err != nil {
		return nil, err
	}
	list := make([]*types.AiConversationInfo, 0, len(out.List))
	for _, item := range out.List {
		list = append(list, mapAiConversation(item))
	}
	return &types.AiListConversationsResp{List: list, Total: out.GetPage().GetTotal()}, nil
}
