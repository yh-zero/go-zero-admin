package agenttoolslogic

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agenttools"
)

type ListAgentDevicesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAgentDevicesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAgentDevicesLogic {
	return &ListAgentDevicesLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ListAgentDevicesLogic) ListAgentDevices(in *pb.AgentToolRequest) (*pb.AgentToolResult, error) {
	return executeTool(l.ctx, l.svcCtx, in, agenttools.Specs[2], func() (agenttools.Output, error) {
		return myDevices(l.ctx, l.svcCtx, in.Actor, json.RawMessage(in.ArgumentsJson))
	})
}
