package agentlogic

import (
	"context"
	"encoding/json"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
	"go-zero-admin/pkg/aiagent"
	"testing"
	"time"
)

func TestAsyncRPCFlowPersistsToolsMessagesAndIsolatesOwner(t *testing.T) {
	s, a, _ := fixture(t)
	ctx := context.Background()
	s.AgentRunner = eventRunner(func(ctx context.Context, r aiagent.Request, h aiagent.EventHandler) (aiagent.Result, error) {
		var device aiagent.Tool
		for _, tool := range r.Tools {
			if tool.Name == "list_my_devices" {
				device = tool
			}
		}
		if err := h(ctx, aiagent.Event{Kind: "tool_started", Tool: aiagent.ToolSummary{Name: device.Name}}); err != nil {
			return aiagent.Result{}, err
		}
		out, err := device.Call(ctx, json.RawMessage(`{}`))
		if err != nil {
			return aiagent.Result{}, err
		}
		summary := aiagent.ToolSummary{Name: device.Name, Success: true, Summary: out.Summary, Count: out.Count}
		if err := h(ctx, aiagent.Event{Kind: "tool_completed", Tool: summary}); err != nil {
			return aiagent.Result{}, err
		}
		return aiagent.Result{FinalAnswer: "本人设备查询完成", Provider: "qwen", Model: "fixture", Usage: aiagent.Usage{PromptTokens: 10, CompletionTokens: 5}, ToolSummaries: []aiagent.ToolSummary{summary}}, nil
	})
	var err error
	s.AgentJobs, err = agentjobs.New(s.DB.DB, agentjobs.Config{Enabled: true, PollInterval: 5 * time.Millisecond}, func(ctx context.Context, j agentjobs.Job) (agentjobs.Result, error) { return execute(ctx, s, j) })
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AgentJobs.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.AgentJobs.Close)
	request := &pb.CreateAgentRunRequest{Actor: a, RequestId: "10000000-0000-4000-8000-000000000001", Message: "查询我的设备"}
	created, err := NewCreateAgentRunLogic(ctx, s).CreateAgentRun(request)
	if err != nil {
		t.Fatal(err)
	}
	reused, err := NewCreateAgentRunLogic(ctx, s).CreateAgentRun(request)
	if err != nil || created.ID != reused.ID {
		t.Fatal("idempotency failed", reused, err)
	}
	deadline := time.Now().Add(2 * time.Second)
	var run *pb.AgentRun
	for time.Now().Before(deadline) {
		run, err = NewGetAgentRunLogic(ctx, s).GetAgentRun(&pb.AgentIDRequest{Actor: a, ID: created.ID})
		if err != nil {
			t.Fatal(err)
		}
		if agentjobs.IsTerminal(run.Status) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if run.Status != agentjobs.StatusSucceeded || run.Answer == "" || len(run.ToolCalls) != 1 || run.ToolCalls[0].Status != "succeeded" || run.InputTokens != 10 {
		t.Fatal("execution/persistence failed", run)
	}
	convs, err := NewListAgentConversationsLogic(ctx, s).ListAgentConversations(&pb.AgentPageRequest{Actor: a, PageNo: 1, PageSize: 20})
	if err != nil || convs.Total != 1 {
		t.Fatal(convs, err)
	}
	messages, err := NewListAgentMessagesLogic(ctx, s).ListAgentMessages(&pb.AgentMessageListRequest{Actor: a, ID: created.ConversationId, PageNo: 1, PageSize: 20})
	if err != nil || messages.Total != 2 || messages.Items[0].Role != "assistant" || messages.Items[1].Role != "user" {
		t.Fatal(messages, err)
	}
	other := &pb.SessionRequest{UserID: 30, AuthorityId: 1, SessionVersion: 1}
	if _, err := NewGetAgentRunLogic(ctx, s).GetAgentRun(&pb.AgentIDRequest{Actor: other, ID: created.ID}); err == nil {
		t.Fatal("admin read another owner's conversation")
	}
	stopped, err := NewCancelAgentRunLogic(ctx, s).CancelAgentRun(&pb.AgentIDRequest{Actor: a, ID: created.ID})
	if err != nil || stopped.Status != "succeeded" {
		t.Fatal("terminal cancellation changed successful task", stopped, err)
	}
}
