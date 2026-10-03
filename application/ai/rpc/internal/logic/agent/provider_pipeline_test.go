package agentlogic

import (
	"context"
	"encoding/json"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
	"go-zero-admin/pkg/aiagent"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The external model is local, but the adapter, Eino loop, authorization,
// remote business tools, progress storage and task/message persistence are real.
func TestProviderBusinessToolPersistencePipeline(t *testing.T) {
	for _, provider := range []string{aiagent.ProviderDeepSeek, aiagent.ProviderQwen} {
		t.Run(provider, func(t *testing.T) {
			s, a, _ := fixture(t)
			var calls atomic.Int32
			var sawTool atomic.Bool
			var sawHistory atomic.Bool
			providerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer local-fixture-key" {
					t.Error("unexpected provider request")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var request struct {
					Messages []struct {
						Role       string `json:"role"`
						Content    string `json:"content"`
						ToolCallID string `json:"tool_call_id"`
					} `json:"messages"`
					Tools []struct {
						Function struct {
							Name string `json:"name"`
						} `json:"function"`
					} `json:"tools"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if len(request.Tools) != 3 {
					t.Errorf("registered tools=%d", len(request.Tools))
				}
				for _, m := range request.Messages {
					if m.Role == "assistant" && m.Content == "文件状态正常，当前引用数为0。" {
						sawHistory.Store(true)
					}
				}
				var message map[string]any
				finish := "tool_calls"
				if calls.Add(1) == 1 {
					message = map[string]any{"role": "assistant", "content": "", "tool_calls": []any{map[string]any{"id": "file_call", "type": "function", "function": map[string]any{"name": "get_file_status", "arguments": `{"fileId":1}`}}}}
				} else {
					for _, m := range request.Messages {
						if m.Role == "tool" && m.ToolCallID == "file_call" {
							if strings.Contains(m.Content, "never-send-object-key") || !strings.Contains(m.Content, `"references":0`) {
								t.Error("authorized tool result missing or leaked private fields")
							}
							sawTool.Store(true)
						}
					}
					finish = "stop"
					message = map[string]any{"role": "assistant", "content": "文件状态正常，当前引用数为0。"}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"id": "fixture", "object": "chat.completion", "model": "fixture", "choices": []any{map[string]any{"index": 0, "message": message, "finish_reason": finish}}, "usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}})
			}))
			t.Cleanup(providerServer.Close)
			installPipeline(t, s, provider, providerServer.URL)
			created, err := NewCreateAgentRunLogic(context.Background(), s).CreateAgentRun(&pb.CreateAgentRunRequest{Actor: a, RequestId: "20000000-0000-4000-8000-000000000001", Message: "查询文件1的状态和引用数"})
			if err != nil {
				t.Fatal(err)
			}
			run := waitPipelineRun(t, s, a.UserID, created.ID)
			if run.Status != agentjobs.StatusSucceeded || run.Text == "" || calls.Load() != 2 || !sawTool.Load() || run.Provider != provider || run.InputTokens != 20 || run.OutputTokens != 10 {
				t.Fatalf("pipeline failed status=%s calls=%d tool=%t error=%s", run.Status, calls.Load(), sawTool.Load(), run.Error)
			}
			response := runProto(run)
			if len(response.ToolCalls) != 1 || response.ToolCalls[0].Name != "get_file_status" || response.ToolCalls[0].Status != "succeeded" {
				t.Fatal("tool progress not persisted", response.ToolCalls)
			}
			messages, total, err := s.AgentJobs.ListMessages(context.Background(), a.UserID, run.ConversationID, 1, 20)
			if err != nil || total != 2 || messages[0].Role != "assistant" || messages[1].Role != "user" || messages[0].Content != run.Text {
				t.Fatal("history not persisted correctly", total, err)
			}
			followup, err := NewCreateAgentRunLogic(context.Background(), s).CreateAgentRun(&pb.CreateAgentRunRequest{Actor: a, ConversationId: run.ConversationID, RequestId: "20000000-0000-4000-8000-000000000003", Message: "继续检查这个文件"})
			if err != nil {
				t.Fatal(err)
			}
			next := waitPipelineRun(t, s, a.UserID, followup.ID)
			if next.Status != agentjobs.StatusSucceeded || !sawHistory.Load() || next.ConversationID != run.ConversationID || calls.Load() != 3 {
				t.Fatal("persisted history did not reach follow-up provider request", next.Status)
			}
			_, total, err = s.AgentJobs.ListMessages(context.Background(), a.UserID, run.ConversationID, 1, 20)
			if err != nil || total != 4 {
				t.Fatal("follow-up history was not persisted", total, err)
			}
		})
	}
}

func TestProviderHTTPCallStopsWhenTaskIsCancelled(t *testing.T) {
	s, a, _ := fixture(t)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	providerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			t.Error(err)
			return
		}
		close(started)
		select {
		case <-r.Context().Done():
			close(cancelled)
		case <-time.After(4 * time.Second):
			t.Error("model HTTP request did not receive cancellation")
		}
	}))
	t.Cleanup(providerServer.Close)
	installPipeline(t, s, aiagent.ProviderQwen, providerServer.URL)
	created, err := NewCreateAgentRunLogic(context.Background(), s).CreateAgentRun(&pb.CreateAgentRunRequest{Actor: a, RequestId: "20000000-0000-4000-8000-000000000002", Message: "查询我的设备"})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("provider call did not start")
	}
	stopped, err := NewCancelAgentRunLogic(context.Background(), s).CancelAgentRun(&pb.AgentIDRequest{Actor: a, ID: created.ID})
	if err != nil || stopped.Status != agentjobs.StatusCanceled {
		t.Fatal(stopped, err)
	}
	select {
	case <-cancelled:
	case <-time.After(2 * time.Second):
		t.Fatal("cancel did not reach provider HTTP transport")
	}
	run := waitPipelineRun(t, s, a.UserID, created.ID)
	if run.Status != agentjobs.StatusCanceled || run.Text != "" {
		t.Fatal("cancelled run gained an answer")
	}
	_, total, err := s.AgentJobs.ListMessages(context.Background(), a.UserID, run.ConversationID, 1, 20)
	if err != nil || total != 1 {
		t.Fatal("cancelled task persisted an assistant message", total, err)
	}
}

func installPipeline(t *testing.T, s *svc.ServiceContext, provider, endpoint string) {
	t.Helper()
	cfg := aiagent.Config{Enabled: true, Provider: provider, Model: "fixture", APIKey: "local-fixture-key", BaseURL: endpoint + "/v1", AllowHTTPForLoopback: true, MaxInputChars: 2000, MaxRunSeconds: 10, MaxSteps: 6, RequestTimeout: 5 * time.Second, RunTimeout: 10 * time.Second}
	var err error
	s.AgentConfig = cfg
	s.AgentRunner, err = aiagent.New(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.AgentJobs, err = agentjobs.New(s.DB.DB, agentjobs.Config{Enabled: true, PollInterval: 5 * time.Millisecond, RunTimeout: 10 * time.Second}, func(ctx context.Context, j agentjobs.Job) (agentjobs.Result, error) { return execute(ctx, s, j) })
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AgentJobs.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.AgentJobs.Close)
}

func waitPipelineRun(t *testing.T, s *svc.ServiceContext, ownerID int64, runID string) agentjobs.Run {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		run, err := s.AgentJobs.Get(context.Background(), ownerID, runID)
		if err != nil {
			t.Fatal(err)
		}
		if agentjobs.IsTerminal(run.Status) {
			return run
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("pipeline run did not finish")
	return agentjobs.Run{}
}
