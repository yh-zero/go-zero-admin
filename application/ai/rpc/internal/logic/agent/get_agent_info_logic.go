// Code scaffolded by goctl. Safe to edit.
package agentlogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
	"go-zero-admin/pkg/agenttools"
)

type GetAgentInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAgentInfoLogic(ctx context.Context, s *svc.ServiceContext) *GetAgentInfoLogic {
	return &GetAgentInfoLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *GetAgentInfoLogic) GetAgentInfo(in *pb.AgentInfoRequest) (*pb.AgentInfo, error) {
	if in == nil {
		return nil, requestError(agentjobs.ErrInvalid)
	}
	if err := authorize(l.ctx, l.svcCtx, in.Actor, "/v1/ai/info", "GET"); err != nil {
		return nil, err
	}
	c := l.svcCtx.AgentConfig
	out := &pb.AgentInfo{Enabled: c.Enabled, Configured: c.Enabled && l.svcCtx.AgentRunner != nil, Provider: c.Provider, Model: c.Model, MaxInputChars: int64(c.MaxInputChars), MaxSteps: int64(c.MaxSteps), MaxRunSeconds: int64(c.MaxRunSeconds), Tools: make([]*pb.AgentToolInfo, 0)}
	for _, t := range agenttools.Specs {
		err := toolPermission(l.ctx, l.svcCtx, in.Actor, t)
		if err != nil && !errors.Is(err, errPermissionDenied) {
			return nil, err
		}
		out.Tools = append(out.Tools, &pb.AgentToolInfo{Name: t.Name, Label: t.Label, Description: t.Description, Available: err == nil})
	}
	return out, nil
}
