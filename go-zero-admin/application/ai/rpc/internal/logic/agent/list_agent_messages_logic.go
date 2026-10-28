// Code scaffolded by goctl. Safe to edit.
package agentlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
)

type ListAgentMessagesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAgentMessagesLogic(ctx context.Context, s *svc.ServiceContext) *ListAgentMessagesLogic {
	return &ListAgentMessagesLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *ListAgentMessagesLogic) ListAgentMessages(in *pb.AgentMessageListRequest) (*pb.AgentMessageListResponse, error) {
	if in == nil {
		return nil, requestError(agentjobs.ErrInvalid)
	}
	if err := authorize(l.ctx, l.svcCtx, in.Actor, "/v1/ai/conversations/:id/messages", "GET"); err != nil {
		return nil, err
	}
	if in.PageNo < 1 || in.PageNo > 1000000 || in.PageSize < 1 || in.PageSize > 100 {
		return nil, requestError(agentjobs.ErrInvalid)
	}
	rows, total, err := l.svcCtx.AgentJobs.ListMessages(l.ctx, in.Actor.UserID, in.ID, int(in.PageNo), int(in.PageSize))
	if err != nil {
		return nil, requestError(err)
	}
	out := &pb.AgentMessageListResponse{Total: total, Items: make([]*pb.AgentMessage, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, &pb.AgentMessage{ID: r.ID, ConversationId: r.ConversationID, RunId: r.RunID, Role: r.Role, Content: r.Content, CreatedAt: timestamp(r.CreatedAt)})
	}
	return out, nil
}
