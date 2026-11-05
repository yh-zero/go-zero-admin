package aiservicelogic

import (
	"context"
	"encoding/json"
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

func writeModelFrame(t *testing.T, w http.ResponseWriter, frame any) {
	t.Helper()
	data, err := json.Marshal(frame)
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
}
