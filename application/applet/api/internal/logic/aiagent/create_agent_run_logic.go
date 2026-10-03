// Code scaffolded by goctl. Safe to edit.
package aiagent

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
)

type CreateAgentRunLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateAgentRunLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAgentRunLogic {
	return &CreateAgentRunLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}
func (l *CreateAgentRunLogic) CreateAgentRun(req *types.CreateAgentRunRequest) (*types.AgentRun, error) {
	if l.svcCtx.AIAgentRPC == nil {
		return nil, aiUnavailable()
	}
	r, e := l.svcCtx.AIAgentRPC.CreateAgentRun(l.ctx, &pb.CreateAgentRunRequest{Actor: actor(l.ctx), ConversationId: req.ConversationId, RequestId: req.RequestId, Message: req.Message})
	if e != nil {
		return nil, aiRPCError(e)
	}
	if r == nil {
		return nil, aiUnavailable()
	}
	return runResponse(r), nil
}
