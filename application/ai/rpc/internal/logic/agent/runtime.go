package agentlogic

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/application/ai/rpc/pb"
	appletpb "go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
	"go-zero-admin/pkg/aiagent"
	"go-zero-admin/pkg/result/xerr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errPermissionDenied = xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "无权使用此AI能力")

// NewAgentExecutor binds the business workflow to the shared dependencies.
// Model and task infrastructure are constructed by svc, not request logic.
func NewAgentExecutor(s *svc.ServiceContext) agentjobs.Executor {
	return func(ctx context.Context, job agentjobs.Job) (agentjobs.Result, error) {
		return execute(ctx, s, job)
	}
}

func session(ctx context.Context, s *svc.ServiceContext, a *pb.SessionRequest) error {
	if a == nil || a.UserID <= 0 || a.AuthorityId <= 0 || a.SessionVersion <= 0 {
		return xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	if s.AppletUserRPC == nil {
		return businessUnavailable()
	}
	r, err := s.AppletUserRPC.CheckSession(ctx, businessActor(a))
	if err != nil {
		return businessError(ctx, err)
	}
	if r == nil || !r.Valid {
		return xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	return nil
}
func authorize(ctx context.Context, s *svc.ServiceContext, a *pb.SessionRequest, path, method string) error {
	if err := session(ctx, s, a); err != nil {
		return err
	}
	return enforce(ctx, s, a, path, method)
}

// enforce is used only within a boundary that has already checked the session.
// Every model/tool boundary still performs a fresh session check.
func enforce(ctx context.Context, s *svc.ServiceContext, a *pb.SessionRequest, path, method string) error {
	if s.AppletCasbinRPC == nil {
		return businessUnavailable()
	}
	r, err := s.AppletCasbinRPC.Enforce(ctx, &appletpb.EnforceRequest{AuthorityId: strconv.FormatInt(a.AuthorityId, 10), Path: path, Method: method})
	if err != nil {
		return businessError(ctx, err)
	}
	if r == nil {
		return businessUnavailable()
	}
	if !r.Pass {
		return errPermissionDenied
	}
	return nil
}

func businessActor(a *pb.SessionRequest) *appletpb.SessionRequest {
	return &appletpb.SessionRequest{UserID: a.UserID, AuthorityId: a.AuthorityId, SessionID: a.SessionID, SessionVersion: a.SessionVersion}
}

func businessUnavailable() error {
	return xerr.NewErrMsg("业务鉴权服务暂不可用，请稍后重试")
}

func businessError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if status.Code(err) == codes.Code(xerr.TOKEN_EXPIRE_ERROR) {
		return xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	// Transport and remote failures may contain connection details or input data.
	return businessUnavailable()
}

type toolCall struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Error   string `json:"error"`
	Summary string `json:"summary"`
}

func execute(ctx context.Context, s *svc.ServiceContext, job agentjobs.Job) (agentjobs.Result, error) {
	a := &pb.SessionRequest{UserID: job.Run.OwnerID, AuthorityId: job.Run.AuthorityID, SessionID: job.Run.SessionID, SessionVersion: job.Run.SessionVersion}
	if err := authorize(ctx, s, a, "/v1/ai/runs", "POST"); err != nil {
		return agentjobs.Result{}, err
	}
	tools, err := availableTools(ctx, s, a)
	if err != nil {
		return agentjobs.Result{}, err
	}
	if s.AgentRunner == nil || s.AgentJobs == nil {
		return agentjobs.Result{}, agentjobs.ErrDisabled
	}
	history := make([]aiagent.Message, 0, len(job.History)+1)
	for _, m := range job.History {
		history = append(history, aiagent.Message{Role: m.Role, Content: m.Content})
	}
	history = append(history, aiagent.Message{Role: "user", Content: job.Run.Question})
	// Keep recent complete turns when long but valid historical answers would
	// exceed the runner's context byte budget. Never truncate the current question.
	for historyBytes(history) > 64*1024 && len(history) > 2 {
		history = history[2:]
	}
	traces := make([]toolCall, 0)
	result, err := s.AgentRunner.Run(ctx, aiagent.Request{History: history, Tools: tools}, func(ctx context.Context, e aiagent.Event) error {
		// A queued task can outlive its browser session. Check again before
		// every paid model request, as well as before and after each tool.
		if e.Kind == "model_started" {
			return authorize(ctx, s, a, "/v1/ai/runs", "POST")
		}
		if e.Tool.Name == "" {
			return nil
		}
		state := "running"
		if e.Kind == "tool_completed" {
			state = "succeeded"
			if !e.Tool.Success {
				state = "failed"
			}
		}
		// End events replace the corresponding start entry; repeated calls remain
		// separate steps. Only summaries cross the polling API boundary.
		trace := toolCall{Name: e.Tool.Name, Status: state, Summary: e.Tool.Summary}
		if state == "failed" {
			trace.Error = "工具执行失败或没有权限"
		}
		if state != "running" && len(traces) > 0 && traces[len(traces)-1].Name == trace.Name && traces[len(traces)-1].Status == "running" {
			traces[len(traces)-1] = trace
		} else {
			traces = append(traces, trace)
		}
		raw, _ := json.Marshal(traces)
		return s.AgentJobs.UpdateProgress(ctx, job.Run.ID, string(raw))
	})
	if err != nil {
		raw, _ := json.Marshal(traces)
		return agentjobs.Result{Provider: result.Provider, Model: result.Model, InputTokens: int64(result.Usage.PromptTokens), OutputTokens: int64(result.Usage.CompletionTokens), ExtraJSON: string(raw)}, err
	}
	if err := authorize(ctx, s, a, "/v1/ai/runs", "POST"); err != nil {
		raw, _ := json.Marshal(traces)
		return agentjobs.Result{Provider: result.Provider, Model: result.Model, InputTokens: int64(result.Usage.PromptTokens), OutputTokens: int64(result.Usage.CompletionTokens), ExtraJSON: string(raw)}, err
	}
	traces = make([]toolCall, 0, len(result.ToolSummaries))
	for _, t := range result.ToolSummaries {
		status := "succeeded"
		msg := ""
		if !t.Success {
			status = "failed"
			msg = "工具执行失败或没有权限"
		}
		traces = append(traces, toolCall{Name: t.Name, Status: status, Error: msg, Summary: t.Summary})
	}
	raw, _ := json.Marshal(traces)
	return agentjobs.Result{Text: result.FinalAnswer, Provider: result.Provider, Model: result.Model, InputTokens: int64(result.Usage.PromptTokens), OutputTokens: int64(result.Usage.CompletionTokens), ExtraJSON: string(raw)}, nil
}

func historyBytes(history []aiagent.Message) int {
	total := 0
	for _, m := range history {
		total += len(m.Content)
	}
	return total
}

func runProto(r agentjobs.Run) *pb.AgentRun {
	out := &pb.AgentRun{ID: r.ID, ConversationId: r.ConversationID, RequestId: r.RequestID, Status: r.Status, Answer: r.Text, Error: r.Error, Provider: r.Provider, Model: r.Model, InputTokens: r.InputTokens, OutputTokens: r.OutputTokens, CreatedAt: timestamp(r.CreatedAt), UpdatedAt: timestamp(r.UpdatedAt), ToolCalls: make([]*pb.AgentToolCall, 0)}
	var calls []toolCall
	if json.Unmarshal([]byte(r.ExtraJSON), &calls) == nil {
		for _, c := range calls {
			out.ToolCalls = append(out.ToolCalls, &pb.AgentToolCall{Name: c.Name, Status: c.Status, Error: c.Error, Summary: c.Summary})
		}
	}
	return out
}
func timestamp(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }
func requestError(err error) error {
	if err == nil {
		return nil
	}
	msg := "AI任务处理失败，请稍后重试"
	switch {
	case errors.Is(err, agentjobs.ErrDisabled):
		msg = "AI助手尚未启用"
	case errors.Is(err, agentjobs.ErrInvalid):
		msg = "AI请求参数无效"
	case errors.Is(err, agentjobs.ErrNotFound):
		msg = "AI记录不存在或无权访问"
	case errors.Is(err, agentjobs.ErrConflict):
		msg = "请求编号已用于其他内容，请重新提交"
	case errors.Is(err, agentjobs.ErrBusy):
		msg = "已有任务执行中，请等待或停止后重试"
	}
	return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, msg)
}
