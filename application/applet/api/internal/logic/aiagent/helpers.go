package aiagent

import (
	"context"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/ctxJwt"
	"go-zero-admin/pkg/result/xerr"
	"google.golang.org/grpc/status"
)

func aiUnavailable() error {
	return xerr.NewErrCodeMsg(xerr.SERVER_COMMON_ERROR, "AI服务暂不可用，请稍后重试")
}

func aiRPCError(err error) error {
	if s, ok := status.FromError(err); ok && xerr.IsCodeErr(uint32(s.Code())) {
		return err
	}
	return aiUnavailable()
}

func actor(ctx context.Context) *pb.SessionRequest {
	d := ctxJwt.GetJwtData(ctx)
	return &pb.SessionRequest{UserID: d.ID, AuthorityId: d.AuthorityId, SessionVersion: d.SessionVersion, SessionID: d.SessionID}
}
func runResponse(r *pb.AgentRun) *types.AgentRun {
	out := &types.AgentRun{ID: r.ID, ConversationId: r.ConversationId, RequestId: r.RequestId, Status: r.Status, Answer: r.Answer, Error: r.Error, Provider: r.Provider, Model: r.Model, InputTokens: r.InputTokens, OutputTokens: r.OutputTokens, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, ToolCalls: make([]types.AgentToolCall, 0, len(r.ToolCalls))}
	for _, t := range r.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, types.AgentToolCall{Name: t.Name, Status: t.Status, Error: t.Error, Summary: t.Summary})
	}
	return out
}
