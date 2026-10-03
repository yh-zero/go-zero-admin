// Code scaffolded by goctl. Safe to edit.
package agentlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
	"unicode/utf8"
)

type CreateAgentRunLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateAgentRunLogic(ctx context.Context, s *svc.ServiceContext) *CreateAgentRunLogic {
	return &CreateAgentRunLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *CreateAgentRunLogic) CreateAgentRun(in *pb.CreateAgentRunRequest) (*pb.AgentRun, error) {
	if in == nil {
		return nil, requestError(agentjobs.ErrInvalid)
	}
	if err := authorize(l.ctx, l.svcCtx, in.Actor, "/v1/ai/runs", "POST"); err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(in.Message) > l.svcCtx.AgentConfig.MaxInputChars {
		return nil, requestError(agentjobs.ErrInvalid)
	}
	r, err := l.svcCtx.AgentJobs.Submit(l.ctx, agentjobs.SubmitInput{OwnerID: in.Actor.UserID, AuthorityID: in.Actor.AuthorityId, SessionID: in.Actor.SessionID, SessionVersion: in.Actor.SessionVersion, RequestID: in.RequestId, Question: in.Message, ConversationID: in.ConversationId})
	if err != nil {
		return nil, requestError(err)
	}
	return runProto(r), nil
}
