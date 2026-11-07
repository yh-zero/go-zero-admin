package ai

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"
)

type CreateConversationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateConversationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateConversationLogic {
	return &CreateConversationLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *CreateConversationLogic) CreateConversation(req *types.AiCreateConversationReq) (*types.AiConversationResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.CreateConversation(l.ctx, &aipb.CreateConversationRequest{Actor: actor, Title: req.Title})
	if err != nil {
		return nil, err
	}
	return &types.AiConversationResp{Conversation: mapAiConversation(out.Conversation)}, nil
}

func actorForAi(ctx context.Context, svcCtx *svc.ServiceContext) (*aipb.Actor, error) {
	userId, _ := ctx.Value(middleware.CtxKeyUserID).(int64)
	if userId <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if svcCtx.AiRpc == nil {
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "AI 服务未配置")
	}
	username, _ := ctx.Value(middleware.CtxKeyUsername).(string)
	deptId, _ := ctx.Value(middleware.CtxKeyDeptId).(int64)
	ip, _ := ctx.Value(middleware.CtxKeyClientIP).(string)
	agent, _ := ctx.Value(middleware.CtxKeyUserAgent).(string)
	return &aipb.Actor{UserId: userId, Username: username, DeptId: deptId, Ip: ip, UserAgent: agent}, nil
}
func mapAiConversation(row *aipb.ConversationInfo) *types.AiConversationInfo {
	if row == nil {
		return nil
	}
	return &types.AiConversationInfo{Id: row.Id, Title: row.Title, UserId: row.UserId, MessageCount: row.MessageCount, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, Pinned: row.Pinned}
}
func mapAiMessage(row *aipb.ChatMessage) *types.AiChatMessage {
	if row == nil {
		return nil
	}
	sources := make([]*types.AiKbSearchHit, 0, len(row.Sources))
	for _, hit := range row.Sources {
		sources = append(sources, &types.AiKbSearchHit{ChunkId: hit.ChunkId, KbId: hit.KbId, DocId: hit.DocId, DocTitle: hit.DocTitle, Snippet: hit.Snippet, Score: hit.Score})
	}
	return &types.AiChatMessage{Id: row.Id, Role: row.Role, Content: row.Content, ToolCalls: row.ToolCalls, Tokens: row.Tokens, CreatedAt: row.CreatedAt, Status: row.Status, Error: row.Error, Sources: sources, Feedback: row.Feedback, FeedbackNote: row.FeedbackNote}
}
func mapAiWorkflow(row *aipb.WorkflowInfo) *types.AiWorkflowInfo {
	if row == nil {
		return nil
	}
	return &types.AiWorkflowInfo{Id: row.Id, Name: row.Name, Definition: row.Definition, IntervalSeconds: row.IntervalSeconds, Enabled: row.Enabled, NextRunAt: row.NextRunAt, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, StepCount: row.StepCount}
}
func mapAiTask(row *aipb.AgentTaskInfo) *types.AiTaskInfo {
	if row == nil {
		return nil
	}
	steps := make([]*types.AiWorkflowStepInfo, 0, len(row.Steps))
	for _, r := range row.Steps {
		if r != nil {
			steps = append(steps, &types.AiWorkflowStepInfo{Index: r.Index, Id: r.Id, Name: r.Name, Tool: r.Tool, Input: r.Input, Output: r.Output, Status: r.Status, Error: r.Error, StartedAt: r.StartedAt, CompletedAt: r.CompletedAt})
		}
	}
	return &types.AiTaskInfo{Id: row.Id, WorkflowId: row.WorkflowId, ConversationId: row.ConversationId, Status: row.Status, Input: row.Input, Result: row.Result, Error: row.Error, ApprovedBy: row.ApprovedBy, CreatedAt: row.CreatedAt, CompletedAt: row.CompletedAt, NextStep: row.NextStep, Steps: steps, UpdatedAt: row.UpdatedAt, RequestId: row.RequestId, Title: row.Title, WorkflowName: row.WorkflowName, InputSummary: row.InputSummary, Definition: row.Definition}
}
