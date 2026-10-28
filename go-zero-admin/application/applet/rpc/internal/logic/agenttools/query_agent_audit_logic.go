package agenttoolslogic

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agenttools"
)

type QueryAgentAuditLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewQueryAgentAuditLogic(ctx context.Context, svcCtx *svc.ServiceContext) *QueryAgentAuditLogic {
	return &QueryAgentAuditLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *QueryAgentAuditLogic) QueryAgentAudit(in *pb.AgentToolRequest) (*pb.AgentToolResult, error) {
	return executeTool(l.ctx, l.svcCtx, in, agenttools.Specs[0], func() (agenttools.Output, error) {
		return queryAudit(l.ctx, l.svcCtx, json.RawMessage(in.ArgumentsJson))
	})
}
