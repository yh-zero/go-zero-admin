package agenttoolslogic

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agenttools"
)

type GetAgentFileStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAgentFileStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAgentFileStatusLogic {
	return &GetAgentFileStatusLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetAgentFileStatusLogic) GetAgentFileStatus(in *pb.AgentToolRequest) (*pb.AgentToolResult, error) {
	return executeTool(l.ctx, l.svcCtx, in, agenttools.Specs[1], func() (agenttools.Output, error) {
		return fileStatus(l.ctx, l.svcCtx, in.Actor, json.RawMessage(in.ArgumentsJson))
	})
}
