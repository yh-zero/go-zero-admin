// Code scaffolded by goctl. Safe to edit.
package agentlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
)

type ListAgentConversationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAgentConversationsLogic(ctx context.Context, s *svc.ServiceContext) *ListAgentConversationsLogic {
	return &ListAgentConversationsLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *ListAgentConversationsLogic) ListAgentConversations(in *pb.AgentPageRequest) (*pb.AgentConversationListResponse, error) {
	if in == nil {
		return nil, requestError(agentjobs.ErrInvalid)
	}
	if err := authorize(l.ctx, l.svcCtx, in.Actor, "/v1/ai/conversations", "GET"); err != nil {
		return nil, err
	}
	if in.PageNo < 1 || in.PageNo > 1000000 || in.PageSize < 1 || in.PageSize > 100 {
		return nil, requestError(agentjobs.ErrInvalid)
	}
	rows, total, err := l.svcCtx.AgentJobs.ListConversations(l.ctx, in.Actor.UserID, int(in.PageNo), int(in.PageSize))
	if err != nil {
		return nil, requestError(err)
	}
	out := &pb.AgentConversationListResponse{Total: total, Items: make([]*pb.AgentConversation, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, &pb.AgentConversation{ID: r.ID, Title: r.Title, CreatedAt: timestamp(r.CreatedAt), UpdatedAt: timestamp(r.UpdatedAt)})
	}
	return out, nil
}
