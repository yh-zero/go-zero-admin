package aiservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/eino/schema"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/llm"
)

// 使用真实模型适配器连接本地 SSE 桩，检查点断言约束工具执行前的持久化边界。
func TestAIStreamCheckpointsCombineUsageAndToolTrace(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if requests.Add(1) == 1 {
			writeModelFrame(t, w, map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "call-local", "type": "function", "function": map[string]any{"name": "text_stats", "arguments": `{"text":"hello world"}`}}}}, "finish_reason": "tool_calls"}}, "usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 3, "total_tokens": 13}})
		} else {
			writeModelFrame(t, w, map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "content": "本地回复"}, "finish_reason": "stop"}}, "usage": map[string]any{"prompt_tokens": 20, "completion_tokens": 4, "total_tokens": 24}})
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	client, err := llm.New(llm.Config{BaseURL: server.URL + "/v1", APIKey: "local-test-only", Model: "local-test", Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	var checkpoints []int
	var totals []int64
	answer, complete, err := askModelStream(context.Background(), client, []*schema.Message{schema.UserMessage("本地测试")}, func(*aipb.StreamChunk) error { return nil }, func(total int64, calls []schema.ToolCall, _ bool) error {
		checkpoints = append(checkpoints, len(calls))
		totals = append(totals, total)
		return nil
	}, 32000)
	if err != nil {
		t.Fatal(err)
	}
	if answer != "本地回复" || !complete || requests.Load() != 2 {
		t.Fatalf("模型桩轮次或最终结果错误，轮次=%d complete=%v", requests.Load(), complete)
	}
	if len(checkpoints) != 2 || checkpoints[0] != 1 || checkpoints[1] != 1 || totals[0] != 13 || totals[1] != 37 {
		t.Fatalf("每个模型轮次应仅保存一次完整用量与工具记录，检查点=%v 用量=%v", checkpoints, totals)
	}
}

func TestAIStreamKeepsReceivedUsageOnBrokenFrameOrCancellation(t *testing.T) {
	for _, mode := range []string{"content_usage_then_broken", "usage_only_then_broken", "cancel_after_content_usage"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				if mode == "usage_only_then_broken" {
					fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"部分回复\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n")
				} else {
					fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"部分回复\"}}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\n")
				}
				if mode == "cancel_after_content_usage" {
					w.(http.Flusher).Flush()
					<-r.Context().Done()
					return
				}
				fmt.Fprint(w, "data: {broken-json\n\n")
			}))
			defer server.Close()
			client, err := llm.New(llm.Config{BaseURL: server.URL, APIKey: "local-test-only", Model: "local-test", Timeout: 2})
			if err != nil {
				t.Fatal(err)
			}
			var saved int64
			var savedComplete bool
			answer, complete, err := askModelStream(ctx, client, []*schema.Message{schema.UserMessage("失败用量")}, func(chunk *aipb.StreamChunk) error {
				if mode == "cancel_after_content_usage" && chunk.Type == "delta" {
					cancel()
					return errors.New("local client closed")
				}
				return nil
			}, func(tokens int64, _ []schema.ToolCall, complete bool) error {
				saved, savedComplete = tokens, complete
				return nil
			}, 32000)
			if err == nil || answer != "" || complete || saved != 15 || savedComplete {
				t.Fatalf("坏帧/取消必须保存已收到15 tokens且不标完整: answer=%q complete=%v saved=%d/%v err=%v", answer, complete, saved, savedComplete, err)
			}
		})
	}
}

func TestAIStreamRejectsLaterInvalidUsageWithoutLosingEarlierKnownTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"前段\"}}],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\ndata: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"后段\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":-1,\"completion_tokens\":3,\"total_tokens\":2}}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	client, err := llm.New(llm.Config{BaseURL: server.URL, APIKey: "local-test-only", Model: "local-test", Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	var known int64
	answer, complete, err := askModelStream(context.Background(), client, []*schema.Message{schema.UserMessage("坏用量")}, func(*aipb.StreamChunk) error { return nil }, func(tokens int64, _ []schema.ToolCall, _ bool) error { known = tokens; return nil }, 32000)
	if err == nil || answer != "" || complete || known != 15 {
		t.Fatalf("后续非法用量仍须安全拒绝，同时保留先前15 tokens: answer=%q known=%d complete=%v err=%v", answer, known, complete, err)
	}
}

func TestAIStreamDoesNotClaimInconsistentUsageComplete(t *testing.T) {
	for _, tc := range []struct {
		name               string
		prompt, completion int
		total              int
		wantComplete       bool
		wantError          bool
	}{
		{name: "完整", prompt: 10, completion: 3, total: 13, wantComplete: true},
		{name: "总量缺失", prompt: 10, completion: 3},
		{name: "总量不一致", prompt: 10, completion: 3, total: 14},
		{name: "分项缺失", total: 13},
		{name: "输入用量负数", prompt: -1, completion: 3, total: 2, wantError: true},
		{name: "输出用量负数", prompt: 10, completion: -1, total: 9, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				writeModelFrame(t, w, map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "content": "本地回答"}, "finish_reason": "stop"}}, "usage": map[string]any{"prompt_tokens": tc.prompt, "completion_tokens": tc.completion, "total_tokens": tc.total}})
				fmt.Fprint(w, "data: [DONE]\n\n")
			}))
			defer server.Close()
			client, err := llm.New(llm.Config{BaseURL: server.URL + "/v1", APIKey: "local-test-only", Model: "local-test", Timeout: 2})
			if err != nil {
				t.Fatal(err)
			}
			var saved int64
			var savedComplete bool
			answer, complete, err := askModelStream(context.Background(), client, []*schema.Message{schema.UserMessage("用量边界")}, func(*aipb.StreamChunk) error { return nil }, func(tokens int64, _ []schema.ToolCall, complete bool) error {
				saved, savedComplete = tokens, complete
				return nil
			}, 32000)
			if tc.wantError {
				if err == nil || complete || saved != 0 {
					t.Fatalf("非法用量不得写入或标完整: err=%v complete=%v tokens=%d", err, complete, saved)
				}
				return
			}
			if err != nil || answer != "本地回答" || complete != tc.wantComplete || savedComplete != tc.wantComplete || saved != int64(tc.total) {
				t.Fatalf("仅一致完整的供应商用量可标完整: answer=%q complete=%v saved=%d/%v err=%v", answer, complete, saved, savedComplete, err)
			}
		})
	}
}

func writeModelFrame(t *testing.T, w http.ResponseWriter, frame any) {
	t.Helper()
	data, err := json.Marshal(frame)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
}
