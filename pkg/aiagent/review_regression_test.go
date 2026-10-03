package aiagent

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func TestDeepSeekFollowupExplicitlyDisablesThinking(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Thinking struct {
				Type string `json:"type"`
			} `json:"thinking"`
			Messages []struct {
				Role      string `json:"role"`
				Reasoning string `json:"reasoning_content"`
			} `json:"messages"`
			Tools []json.RawMessage `json:"tools"`
		}
		if json.NewDecoder(r.Body).Decode(&payload) != nil {
			t.Error("invalid request")
		}
		if len(payload.Tools) == 0 {
			t.Error("tool definitions missing from compatibility fixture")
		}
		w.Header().Set("Content-Type", "application/json")
		if requests.Add(1) > 1 && payload.Thinking.Type != "disabled" {
			for _, m := range payload.Messages {
				if m.Role == "assistant" && m.Reasoning == "" {
					w.WriteHeader(http.StatusBadRequest)
					_, _ = w.Write([]byte(`{"error":{"message":"historical assistant reasoning required in thinking mode","type":"invalid_request_error"}}`))
					return
				}
			}
		}
		_, _ = w.Write([]byte(`{"id":"answer","choices":[{"index":0,"message":{"role":"assistant","content":"答案","reasoning_content":"not-persisted"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":3,"total_tokens":23}}`))
	}))
	defer server.Close()
	r := httpFixture(t, ProviderDeepSeek, server.URL+"/v1", nil)
	callable := fixtureTool(func(context.Context, json.RawMessage) (ToolOutput, error) { return ToolOutput{Content: `{}`}, nil })
	first, err := r.Run(context.Background(), fixtureRequest(callable), nil)
	if err != nil {
		t.Fatal(err)
	}
	request := Request{History: []Message{{Role: "user", Content: "第一个问题"}, {Role: "assistant", Content: first.FinalAnswer}, {Role: "user", Content: "继续解释"}}, Tools: []Tool{callable}}
	second, err := r.Run(context.Background(), request, nil)
	if err != nil || second.FinalAnswer != "答案" {
		t.Fatalf("privacy-safe persisted history cannot be followed up: %+v %v", second, err)
	}
}

func TestRejectedAnswerStillReturnsReportedUsage(t *testing.T) {
	r := fixtureRunner(t, functionModel(func(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
		return &schema.Message{Role: schema.Assistant, Content: strings.Repeat("x", 17), ResponseMeta: &schema.ResponseMeta{Usage: &schema.TokenUsage{PromptTokens: 10, CompletionTokens: 4, TotalTokens: 14}}}, nil
	}), func(c *Config) { c.MaxAnswerBytes = 16 })
	result, err := r.Run(context.Background(), fixtureRequest(), nil)
	if !errors.Is(err, ErrBudgetExceeded) || result.FinalAnswer != "" || result.Usage.PromptTokens != 10 || result.Usage.CompletionTokens != 4 || result.Usage.TotalTokens != 14 {
		t.Fatalf("paid response usage was lost when answer was rejected: %+v %v", result, err)
	}
}

func TestGeneratedReasoningCannotExpandNextModelRequestPastBudget(t *testing.T) {
	var requests atomic.Int32
	r := fixtureRunner(t, functionModel(func(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
		if requests.Add(1) == 1 {
			message := toolMessage(`{"fileId":1}`)
			message.ReasoningContent = strings.Repeat("r", 256*1024)
			return message, nil
		}
		return schema.AssistantMessage("答案", nil), nil
	}), nil)
	result, err := r.Run(context.Background(), fixtureRequest(fixtureTool(func(context.Context, json.RawMessage) (ToolOutput, error) { return ToolOutput{Content: `{}`}, nil })), nil)
	if !errors.Is(err, ErrBudgetExceeded) || requests.Load() != 1 || result.FinalAnswer != "" {
		t.Fatalf("generated context bypassed request budget: calls=%d err=%v", requests.Load(), err)
	}
}

func TestContextBudgetAllowsLongHistoryAndOrdinaryToolOutput(t *testing.T) {
	var requests atomic.Int32
	r := fixtureRunner(t, functionModel(func(_ context.Context, messages []*schema.Message, _ ...model.Option) (*schema.Message, error) {
		if requests.Add(1) == 1 {
			return toolMessage(`{"fileId":1}`), nil
		}
		if messages[len(messages)-1].Role != schema.Tool || len(messages[len(messages)-1].Content) < 16000 {
			t.Error("ordinary tool result was not retained")
		}
		return schema.AssistantMessage("答案", nil), nil
	}), nil)
	history := []Message{}
	for i := 0; i < 6; i++ {
		history = append(history, Message{Role: "user", Content: "历史问题"}, Message{Role: "assistant", Content: strings.Repeat("x", 8192)})
	}
	history = append(history, Message{Role: "user", Content: "继续查看"})
	request := Request{History: history, Tools: []Tool{fixtureTool(func(context.Context, json.RawMessage) (ToolOutput, error) {
		return ToolOutput{Content: `{"data":"` + strings.Repeat("x", 16000) + `"}`, Count: 1}, nil
	})}}
	result, err := r.Run(context.Background(), request, nil)
	if err != nil || requests.Load() != 2 || result.FinalAnswer != "答案" {
		t.Fatalf("ordinary context was rejected: %+v %v", result, err)
	}
}

func TestModelInputBudgetBoundary(t *testing.T) {
	messages := []*schema.Message{schema.UserMessage("")}
	emptyPayload, _ := json.Marshal(struct {
		Messages []*schema.Message `json:"messages"`
		Tools    []ToolDefinition  `json:"tools"`
	}{Messages: messages, Tools: []ToolDefinition{}})
	contentLength := maxModelInputBytes - 1024 - len(emptyPayload)
	messages[0].Content = strings.Repeat("x", contentLength)
	if !boundedModelInput(messages, nil) {
		t.Fatal("budget rejected exact allowed boundary")
	}
	messages[0].Content += "x"
	if boundedModelInput(messages, nil) {
		t.Fatal("budget accepted boundary plus one byte")
	}
}

type fixtureRoundTripper func(*http.Request) (*http.Response, error)

func (f fixtureRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type trackedRequestBody struct {
	io.Reader
	closed bool
}

func (b *trackedRequestBody) Close() error { b.closed = true; return nil }

func TestExactHTTPBodyBudgetBeforeTransportForKnownAndUnknownSizes(t *testing.T) {
	for _, unknownSize := range []bool{false, true} {
		for _, excess := range []bool{false, true} {
			name := "known"
			if unknownSize {
				name = "unknown"
			}
			if excess {
				name += "-excess"
			} else {
				name += "-boundary"
			}
			t.Run(name, func(t *testing.T) {
				size := maxModelInputBytes
				if excess {
					size++
				}
				body := &trackedRequestBody{Reader: strings.NewReader(strings.Repeat("x", size))}
				request, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://fixture.invalid/v1/chat/completions", body)
				request.ContentLength = int64(size)
				if unknownSize {
					request.ContentLength = -1
				}
				var called bool
				transport := &boundedTransport{maxBytes: 1024, base: fixtureRoundTripper(func(r *http.Request) (*http.Response, error) {
					called = true
					data, err := io.ReadAll(r.Body)
					if err != nil || len(data) != maxModelInputBytes {
						t.Error("allowed body did not arrive intact")
					}
					_ = r.Body.Close()
					return &http.Response{StatusCode: 200, ContentLength: 2, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
				})}
				response, err := transport.RoundTrip(request)
				if response != nil {
					_ = response.Body.Close()
				}
				if excess {
					if !errors.Is(err, ErrBudgetExceeded) || called {
						t.Fatal("oversized body reached external transport")
					}
				} else if err != nil || !called {
					t.Fatal("allowed boundary was rejected")
				}
				if !body.closed {
					t.Fatal("original request body was not closed")
				}
			})
		}
	}
}
