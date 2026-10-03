// Code scaffolded by goctl. Safe to edit.
package aiagent

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
)

type CancelAgentRunLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCancelAgentRunLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelAgentRunLogic {
	return &CancelAgentRunLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}
func (l *CancelAgentRunLogic) CancelAgentRun(req *types.AgentIDRequest) (*types.AgentRun, error) {
	if l.svcCtx.AIAgentRPC == nil {
		return nil, aiUnavailable()
	}
	r, e := l.svcCtx.AIAgentRPC.CancelAgentRun(l.ctx, &pb.AgentIDRequest{Actor: actor(l.ctx), ID: req.ID})
	if e != nil {
		return nil, aiRPCError(e)
	}
	if r == nil {
		return nil, aiUnavailable()
	}
	return runResponse(r), nil
}
