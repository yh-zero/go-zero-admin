package aiservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/llm"
)

type workflowAnalysisSearcher struct {
	calls atomic.Int32
	owner int64
}

func (s *workflowAnalysisSearcher) Search(_ context.Context, owner, kb int64, query string) ([]agenttools.KbSearchHit, error) {
	s.calls.Add(1)
	if owner != s.owner || kb != 3 || query != "预算" {
		return nil, errors.New("unexpected identity")
	}
	return []agenttools.KbSearchHit{{ChunkID: 1, KbID: 3, DocID: 2, DocTitle: "资料", Snippet: "预算五万元", Score: .9}}, nil
}

type workflowAnalysisTool struct {
	agenttools.Tool
	name string
}

func (t *workflowAnalysisTool) Name() string     { return t.name }
func (*workflowAnalysisTool) WorkflowOnly() bool { return true }

func TestAIStreamRejectsForgedWorkflowOnlyToolWithoutExecution(t *testing.T) {
	var requests, ownership atomic.Int32
	searcher := &workflowAnalysisSearcher{owner: 7}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if requests.Add(1) == 1 {
			writeModelFrame(t, w, map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "forged-workflow", "type": "function", "function": map[string]any{"name": "test_kb_analyze_forged", "arguments": `{"kb_id":3,"query":"预算"}`}}}}, "finish_reason": "tool_calls"}}, "usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 2, "total_tokens": 12}})
		} else {
			writeModelFrame(t, w, map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "content": "工作流工具无法在聊天执行"}, "finish_reason": "stop"}}, "usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 2, "total_tokens": 12}})
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	client, err := llm.New(llm.Config{BaseURL: server.URL + "/v1", APIKey: "local-test-only", Model: "local-test", Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	tool := &workflowAnalysisTool{Tool: agenttools.NewKbAnalyze(searcher, func(context.Context, int64, int64) error { ownership.Add(1); return nil }, client, 32000), name: "test_kb_analyze_forged"}
	agenttools.RegisterIfAbsent(tool)
	_, _, err = askModelStream(agenttools.WithActor(context.Background(), 7), client, []*schema.Message{schema.UserMessage("预算")}, func(*aipb.StreamChunk) error { return nil }, func(int64, []schema.ToolCall, bool) error { return nil }, 32000)
	if err != nil || requests.Load() != 2 || ownership.Load() != 0 || searcher.calls.Load() != 0 {
		t.Fatalf("伪造调用必须零执行零嵌套付费: err=%v requests=%d owned=%d search=%d", err, requests.Load(), ownership.Load(), searcher.calls.Load())
	}
}

func TestAIWorkspaceIntegrationKbAnalyzeFinalApprovalRunsOnce(t *testing.T) {
	h := newWorkspaceHarness(t)
	toolName := fmt.Sprintf("test_kb_analyze_approval_%d", time.Now().UnixNano())
	var ownership, chats atomic.Int32
	searcher := &workflowAnalysisSearcher{owner: h.actor.UserId}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chats.Add(1)
		var request struct {
			MaxTokens int   `json:"max_tokens"`
			Tools     []any `json:"tools"`
		}
		if json.NewDecoder(r.Body).Decode(&request) != nil || request.MaxTokens != 2048 || len(request.Tools) != 0 {
			t.Error("分析请求必须有界且不提供工具")
		}
		w.Header().Set("Content-Type", "application/json")
		content := `{"answers":[{"target":"预算","state":"supported","text":"预算五万元。","citations":[1]}]}`
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []any{map[string]any{
				"message":       map[string]any{"role": "assistant", "content": content},
				"finish_reason": "stop",
			}},
			"usage": map[string]int{"prompt_tokens": 100, "completion_tokens": 10, "total_tokens": 110},
		})
	}))
	defer server.Close()
	client, err := llm.New(llm.Config{BaseURL: server.URL + "/v1", APIKey: "local-test-only", Model: "local-test", Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	tool := &workflowAnalysisTool{Tool: agenttools.NewKbAnalyze(searcher, func(_ context.Context, kb, owner int64) error {
		ownership.Add(1)
		if kb != 3 || owner != h.actor.UserId {
			return errors.New("unexpected identity")
		}
		return nil
	}, client, 32000), name: toolName}
	agenttools.RegisterIfAbsent(tool)
	definition := fmt.Sprintf(`{"steps":[{"name":"前置","tool":"text_stats","input":{"text":"前置"}},{"name":"分析","tool":"%s","input":{"kb_id":3,"query":"预算"},"approval":true}]}`, toolName)
	id := h.run(h.create(definition))
	if task := h.task(id); task.Status != "pending_approval" || task.NextStep != 1 || ownership.Load() != 0 || searcher.calls.Load() != 0 || chats.Load() != 0 {
		t.Fatal("最后一步审批前不得查归属、检索或聊天")
	}
	assertWorkspaceCode(t, h.approve(id, 0), 400)
	start, results := make(chan struct{}), make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { <-start; results <- h.approve(id, 1) }()
	}
	close(start)
	success := 0
	for i := 0; i < 2; i++ {
		if err := <-results; err == nil {
			success++
		} else {
			assertWorkspaceCode(t, err, 400)
		}
	}
	if success != 1 || ownership.Load() != 1 || searcher.calls.Load() != 1 || chats.Load() != 1 {
		t.Fatalf("并发批准只能分析一次: success=%d owned=%d search=%d chat=%d", success, ownership.Load(), searcher.calls.Load(), chats.Load())
	}
	done := h.task(id)
	if done.Status != "succeeded" || done.NextStep != 2 || !strings.Contains(done.Result, `"type":"kb_analysis"`) || !strings.Contains(done.Result, "预算五万元。 [1]") || strings.Count(done.Result, "[2/2]") != 1 {
		t.Fatal("分析结果未持久化")
	}
	assertWorkspaceCode(t, h.approve(id, 1), 400)
	if after := h.task(id); after.Result != done.Result || ownership.Load() != 1 || searcher.calls.Load() != 1 || chats.Load() != 1 {
		t.Fatal("迟到批准不得再次付费")
	}
}

func TestAIWorkspaceIntegrationKbAnalyzeSafeFailurePersists(t *testing.T) {
	h := newWorkspaceHarness(t)
	toolName := fmt.Sprintf("test_kb_analyze_failure_%d", time.Now().UnixNano())
	var ownership atomic.Int32
	searcher := &workflowAnalysisSearcher{owner: h.actor.UserId}
	client, err := llm.New(llm.Config{BaseURL: "http://127.0.0.1:1/v1", APIKey: "local-test-only", Model: "local-test", Timeout: 1})
	if err != nil {
		t.Fatal(err)
	}
	tool := &workflowAnalysisTool{Tool: agenttools.NewKbAnalyze(searcher, func(context.Context, int64, int64) error {
		ownership.Add(1)
		return errors.New("private-vendor-detail")
	}, client, 32000), name: toolName}
	agenttools.RegisterIfAbsent(tool)
	id := h.run(h.create(fmt.Sprintf(`{"steps":[{"name":"分析","tool":"%s","input":{"kb_id":3,"query":"预算"}},{"name":"后置","tool":"get_current_time","input":{}}]}`, toolName)))
	failed := h.task(id)
	if failed.Status != "failed" || failed.NextStep != 0 || failed.Result != "" || failed.Error != "知识库不存在、无权访问或暂时不可用" || ownership.Load() != 1 || searcher.calls.Load() != 0 {
		t.Fatal("归属失败须持久化可读安全原因且不执行后续步骤")
	}
}
