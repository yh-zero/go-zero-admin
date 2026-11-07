package llm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestChatBoundedSendsMaxTokensWithoutToolsAndPreservesChat(t *testing.T) {
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("请求非法")
		}
		requests = append(requests, request)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"回复"},"finish_reason":"length"}],"usage":{"prompt_tokens":4,"completion_tokens":2,"total_tokens":6}}`))
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL + "/v1", APIKey: "local-test-only", Model: "local-test", Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	response, usage, err := client.ChatBounded(context.Background(), []*schema.Message{schema.UserMessage("目标")}, 2048)
	if err != nil || usage == nil || usage.TotalTokens != 6 || response.ResponseMeta.FinishReason != "length" {
		t.Fatalf("响应错误 %v", err)
	}
	if requests[0]["max_tokens"] != float64(2048) || requests[0]["tools"] != nil {
		t.Fatal("有界调用须限制输出且无工具")
	}
	if _, _, err = client.Chat(context.Background(), []*schema.Message{schema.UserMessage("目标")}, nil); err != nil {
		t.Fatal(err)
	}
	if requests[1]["max_tokens"] != nil {
		t.Fatal("旧Chat行为改变")
	}
	if _, _, err = client.ChatBounded(context.Background(), nil, 0); err == nil || len(requests) != 2 {
		t.Fatal("非法预算须在模型调用前拒绝")
	}
}
