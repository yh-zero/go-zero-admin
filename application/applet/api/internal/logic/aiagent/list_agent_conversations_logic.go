// Code scaffolded by goctl. Safe to edit.
package aiagent

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
)

type ListAgentConversationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAgentConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAgentConversationsLogic {
	return &ListAgentConversationsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}
func (l *ListAgentConversationsLogic) ListAgentConversations(req *types.AgentPageRequest) (*types.AgentConversationListResponse, error) {
	if l.svcCtx.AIAgentRPC == nil {
		return nil, aiUnavailable()
	}
	r, e := l.svcCtx.AIAgentRPC.ListAgentConversations(l.ctx, &pb.AgentPageRequest{Actor: actor(l.ctx), PageNo: req.PageNo, PageSize: req.PageSize})
	if e != nil {
		return nil, aiRPCError(e)
	}
	if r == nil {
		return nil, aiUnavailable()
	}
	out := &types.AgentConversationListResponse{Items: make([]types.AgentConversation, 0, len(r.Items)), Total: r.Total}
	for _, v := range r.Items {
		out.Items = append(out.Items, types.AgentConversation{ID: v.ID, Title: v.Title, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt})
	}
	return out, nil
}
