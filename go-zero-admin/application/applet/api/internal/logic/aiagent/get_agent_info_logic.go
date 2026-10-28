// Code scaffolded by goctl. Safe to edit.
package aiagent

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
)

type GetAgentInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAgentInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAgentInfoLogic {
	return &GetAgentInfoLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}
func (l *GetAgentInfoLogic) GetAgentInfo() (*types.AgentInfo, error) {
	if l.svcCtx.AIAgentRPC == nil {
		return nil, aiUnavailable()
	}
	r, e := l.svcCtx.AIAgentRPC.GetAgentInfo(l.ctx, &pb.AgentInfoRequest{Actor: actor(l.ctx)})
	if e != nil {
		return nil, aiRPCError(e)
	}
	if r == nil {
		return nil, aiUnavailable()
	}
	out := &types.AgentInfo{Enabled: r.Enabled, Configured: r.Configured, Provider: r.Provider, Model: r.Model, MaxInputChars: r.MaxInputChars, MaxSteps: r.MaxSteps, MaxRunSeconds: r.MaxRunSeconds, Tools: make([]types.AgentToolInfo, 0, len(r.Tools))}
	for _, t := range r.Tools {
		out.Tools = append(out.Tools, types.AgentToolInfo{Name: t.Name, Label: t.Label, Description: t.Description, Available: t.Available})
	}
	return out, nil
}
