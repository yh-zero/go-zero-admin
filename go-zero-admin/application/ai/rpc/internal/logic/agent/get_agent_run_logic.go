// Code scaffolded by goctl. Safe to edit.
package agentlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
)

type GetAgentRunLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAgentRunLogic(ctx context.Context, s *svc.ServiceContext) *GetAgentRunLogic {
	return &GetAgentRunLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *GetAgentRunLogic) GetAgentRun(in *pb.AgentIDRequest) (*pb.AgentRun, error) {
	if in == nil {
		return nil, requestError(agentjobs.ErrInvalid)
	}
	if err := authorize(l.ctx, l.svcCtx, in.Actor, "/v1/ai/runs/:id", "GET"); err != nil {
		return nil, err
	}
	r, err := l.svcCtx.AgentJobs.Get(l.ctx, in.Actor.UserID, in.ID)
	if err != nil {
		return nil, requestError(err)
	}
	return runProto(r), nil
}
