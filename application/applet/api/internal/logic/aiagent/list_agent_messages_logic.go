// Code scaffolded by goctl. Safe to edit.
package aiagent

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
)

type ListAgentMessagesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAgentMessagesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAgentMessagesLogic {
	return &ListAgentMessagesLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}
func (l *ListAgentMessagesLogic) ListAgentMessages(req *types.AgentMessageListRequest) (*types.AgentMessageListResponse, error) {
	if l.svcCtx.AIAgentRPC == nil {
		return nil, aiUnavailable()
	}
	r, e := l.svcCtx.AIAgentRPC.ListAgentMessages(l.ctx, &pb.AgentMessageListRequest{Actor: actor(l.ctx), ID: req.ID, PageNo: req.PageNo, PageSize: req.PageSize})
	if e != nil {
		return nil, aiRPCError(e)
	}
	if r == nil {
		return nil, aiUnavailable()
	}
	out := &types.AgentMessageListResponse{Items: make([]types.AgentMessage, 0, len(r.Items)), Total: r.Total}
	for _, v := range r.Items {
		out.Items = append(out.Items, types.AgentMessage{ID: v.ID, ConversationId: v.ConversationId, RunId: v.RunId, Role: v.Role, Content: v.Content, CreatedAt: v.CreatedAt})
	}
	return out, nil
}
