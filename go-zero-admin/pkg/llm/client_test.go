package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestChatBoundedSetsZeroTemperatureWithoutChangingChatOrStream(t *testing.T) {
	var requests []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("请求非法")
		}
		requests = append(requests, request)
		if request["stream"] == true {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"流式回复\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":2,\"total_tokens\":6}}\n\ndata: [DONE]\n\n"))
			return
		}
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
	if temperature, exists := requests[0]["temperature"]; !exists || temperature != float64(0) {
		t.Errorf("有界调用须显式发送 temperature=0：exists=%v value=%v", exists, temperature)
	}
	if _, _, err = client.Chat(context.Background(), []*schema.Message{schema.UserMessage("目标")}, nil); err != nil {
		t.Fatal(err)
	}
	if requests[1]["max_tokens"] != nil {
		t.Fatal("旧Chat行为改变")
	}
	if _, exists := requests[1]["temperature"]; exists {
		t.Fatal("有界调用的采样参数泄漏到同一客户端后续Chat")
	}
	reader, err := client.ChatStream(context.Background(), []*schema.Message{schema.UserMessage("目标")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var streamed strings.Builder
	for {
		frame, recvErr := reader.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			t.Fatal(recvErr)
		}
		streamed.WriteString(frame.Content)
	}
	if streamed.String() != "流式回复" || len(requests) != 3 {
		t.Fatal("同一客户端后续流式调用未完成或请求数异常")
	}
	if _, exists := requests[2]["temperature"]; exists || requests[2]["max_tokens"] != nil {
		t.Fatal("有界调用的采样参数或输出预算泄漏到同一客户端后续ChatStream")
	}
	if _, _, err = client.ChatBounded(context.Background(), nil, 0); err == nil || len(requests) != 3 {
		t.Fatal("非法预算须在模型调用前拒绝")
	}
}

func TestChatBoundedReturnsUnsupportedTemperatureWithoutRetry(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"temperature is not supported","type":"invalid_request_error","param":"temperature","code":"unsupported_parameter"}}`))
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL + "/v1", APIKey: "local-test-only", Model: "local-test", Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	response, usage, err := client.ChatBounded(context.Background(), []*schema.Message{schema.UserMessage("目标")}, 2048)
	if err == nil || !strings.Contains(err.Error(), "temperature is not supported") || response != nil || usage != nil || requests != 1 {
		t.Fatalf("供应商拒绝须原样返回且只请求一次：err=%v requests=%d", err, requests)
	}
}
