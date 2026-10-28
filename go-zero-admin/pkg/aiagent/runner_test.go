package aiagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type functionModel func(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error)

func (f functionModel) Generate(ctx context.Context, messages []*schema.Message, options ...model.Option) (*schema.Message, error) {
	return f(ctx, messages, options...)
}
func (f functionModel) Stream(context.Context, []*schema.Message, ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, errors.New("unexpected stream")
}

func fixtureRunner(t *testing.T, chatModel model.BaseChatModel, modify func(*Config)) *runner {
	t.Helper()
	c := Config{Enabled: true, Provider: ProviderDeepSeek, Model: "fixture", APIKey: "test-key", BaseURL: "https://example.com/v1"}
	if modify != nil {
		modify(&c)
	}
	c, err := c.normalized()
	if err != nil {
		t.Fatal(err)
	}
	return &runner{config: c, chatModel: chatModel}
}
func fixtureTool(call func(context.Context, json.RawMessage) (ToolOutput, error)) Tool {
	return Tool{ToolDefinition: ToolDefinition{Name: "get_file_status", Description: "只读文件状态", Parameters: json.RawMessage(`{"type":"object","properties":{"fileId":{"type":"integer"}},"required":["fileId"],"additionalProperties":false}`)}, Call: call}
}
func toolMessage(arguments string) *schema.Message {
	return &schema.Message{Role: schema.Assistant, ReasoningContent: "private-reasoning", ToolCalls: []schema.ToolCall{{ID: "call_1", Type: "function", Function: schema.FunctionCall{Name: "get_file_status", Arguments: arguments}}}, ResponseMeta: &schema.ResponseMeta{Usage: &schema.TokenUsage{PromptTokens: 10, CompletionTokens: 2, TotalTokens: 12}}}
}
func fixtureRequest(tools ...Tool) Request {
	return Request{History: []Message{{Role: "user", Content: "查看文件1状态"}}, Tools: tools}
}

func TestActualADKToolsHistoryUsageAndNoReasoningInTrace(t *testing.T) {
	var modelCalls, toolCalls atomic.Int32
	var events []Event
	fake := functionModel(func(ctx context.Context, messages []*schema.Message, _ ...model.Option) (*schema.Message, error) {
		if modelCalls.Add(1) == 1 {
			if messages[0].Role != schema.System || messages[len(messages)-1].Content != "查看文件1状态" {
				t.Error("ADK did not receive trusted instructions and user history")
			}
			return toolMessage(`{"fileId":1}`), nil
		}
		found := false
		for _, message := range messages {
			if message.Role == schema.Assistant && len(message.ToolCalls) > 0 && message.ReasoningContent != "private-reasoning" {
				t.Error("provider-required reasoning was not privately echoed during the tool turn")
			}
			if message.Role == schema.Tool {
				found = strings.Contains(message.Content, `"status":"active"`)
			}
		}
		if !found {
			t.Error("ADK did not feed actual tool result back to model")
		}
		return &schema.Message{Role: schema.Assistant, Content: "文件1正常。", ReasoningContent: "private-final-reasoning", ResponseMeta: &schema.ResponseMeta{Usage: &schema.TokenUsage{PromptTokens: 20, CompletionTokens: 4, TotalTokens: 24}}}, nil
	})
	r := fixtureRunner(t, fake, nil)
	result, err := r.Run(context.Background(), fixtureRequest(fixtureTool(func(ctx context.Context, args json.RawMessage) (ToolOutput, error) {
		toolCalls.Add(1)
		if string(args) != `{"fileId":1}` {
			t.Error("wrong tool arguments")
		}
		return ToolOutput{Content: `{"id":1,"status":"active"}`, Summary: "返回1条文件状态", Count: 1}, nil
	})), func(_ context.Context, event Event) error { events = append(events, event); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if result.FinalAnswer != "文件1正常。" || modelCalls.Load() != 2 || toolCalls.Load() != 1 || result.Usage.TotalTokens != 36 || len(result.ToolSummaries) != 1 || !result.ToolSummaries[0].Success {
		t.Fatalf("incorrect actual ADK result: %+v", result)
	}
	encoded, _ := json.Marshal(events)
	if len(events) != 6 || strings.Contains(string(encoded), "private-") || strings.Contains(string(encoded), "fileId") || strings.Contains(string(encoded), "active") {
		t.Fatalf("unsafe execution trace: %s", encoded)
	}
}

func TestInvalidBatchDoesNotInvokeAnyRegisteredTool(t *testing.T) {
	for _, variant := range []string{"unknown", "malformed", "array", "duplicate-key", "trailing-data"} {
		t.Run(variant, func(t *testing.T) {
			var toolCalls atomic.Int32
			fake := functionModel(func(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
				message := toolMessage(`{"fileId":1}`)
				bad := schema.ToolCall{ID: "call_2", Type: "function", Function: schema.FunctionCall{Name: "get_file_status", Arguments: `{"fileId":2}`}}
				switch variant {
				case "unknown":
					bad.Function.Name = "delete_user"
				case "malformed":
					bad.Function.Arguments = `{"fileId":`
				case "array":
					bad.Function.Arguments = `[]`
				case "duplicate-key":
					bad.Function.Arguments = `{"fileId":1,"fileId":2}`
				case "trailing-data":
					bad.Function.Arguments = `{} {}`
				}
				message.ToolCalls = append(message.ToolCalls, bad)
				return message, nil
			})
			r := fixtureRunner(t, fake, nil)
			_, err := r.Run(context.Background(), fixtureRequest(fixtureTool(func(context.Context, json.RawMessage) (ToolOutput, error) {
				toolCalls.Add(1)
				return ToolOutput{Content: `{}`}, nil
			})), nil)
			want := ErrToolArguments
			if variant == "unknown" {
				want = ErrUnknownTool
			}
			if !errors.Is(err, want) || toolCalls.Load() != 0 {
				t.Fatalf("bad batch invoked a capability: calls=%d err=%v", toolCalls.Load(), err)
			}
		})
	}
}

func TestToolFailureStopsNextModelAndKeepsRawErrorPrivate(t *testing.T) {
	var modelCalls atomic.Int32
	r := fixtureRunner(t, functionModel(func(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
		modelCalls.Add(1)
		return toolMessage(`{"fileId":1}`), nil
	}), nil)
	result, err := r.Run(context.Background(), fixtureRequest(fixtureTool(func(context.Context, json.RawMessage) (ToolOutput, error) {
		return ToolOutput{}, fmt.Errorf("permission revoked; secret=test-key")
	})), nil)
	if !errors.Is(err, ErrToolFailed) || strings.Contains(err.Error(), "test-key") || modelCalls.Load() != 1 || len(result.ToolSummaries) != 1 || result.ToolSummaries[0].Success {
		t.Fatalf("unsafe tool failure: %+v %v", result, err)
	}
}

func TestTypedArgumentRejectionRetainsSafeClassification(t *testing.T) {
	r := fixtureRunner(t, functionModel(func(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
		return toolMessage(`{"fileId":"wrong-type"}`), nil
	}), nil)
	_, err := r.Run(context.Background(), fixtureRequest(fixtureTool(func(context.Context, json.RawMessage) (ToolOutput, error) {
		return ToolOutput{}, fmt.Errorf("%w: raw rejected data", ErrToolArguments)
	})), nil)
	if !errors.Is(err, ErrToolArguments) || strings.Contains(err.Error(), "raw rejected data") {
		t.Fatal("typed argument error classification or privacy was lost")
	}
}

func TestStepAnswerTokenAndToolOutputBudgets(t *testing.T) {
	for _, variant := range []string{"steps", "answer", "tokens", "tool-output"} {
		t.Run(variant, func(t *testing.T) {
			var calls atomic.Int32
			fake := functionModel(func(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
				calls.Add(1)
				message := toolMessage(`{"fileId":1}`)
				if variant == "answer" {
					message.ToolCalls = nil
					message.Content = strings.Repeat("x", 17)
				}
				if variant == "tokens" {
					message.ResponseMeta.Usage.TotalTokens = 1000
				}
				return message, nil
			})
			r := fixtureRunner(t, fake, func(c *Config) {
				c.MaxSteps = 2
				c.MaxAnswerBytes = 16
				c.MaxToolOutputBytes = 16
				c.MaxTotalTokens = 100
			})
			output := `{}`
			if variant == "tool-output" {
				output = `{"long":"0123456789"}`
			}
			_, err := r.Run(context.Background(), fixtureRequest(fixtureTool(func(context.Context, json.RawMessage) (ToolOutput, error) { return ToolOutput{Content: output}, nil })), nil)
			if !errors.Is(err, ErrBudgetExceeded) || calls.Load() > 2 {
				t.Fatalf("budget was not enforced: calls=%d err=%v", calls.Load(), err)
			}
		})
	}
}

func TestCancellationAndTraceFailureStopExecution(t *testing.T) {
	started := make(chan struct{})
	r := fixtureRunner(t, functionModel(func(ctx context.Context, _ []*schema.Message, _ ...model.Option) (*schema.Message, error) {
		close(started)
		<-ctx.Done()
		return nil, ctx.Err()
	}), nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, err := r.Run(ctx, fixtureRequest(), nil); done <- err }()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancel did not return promptly")
	}
	var calls atomic.Int32
	r = fixtureRunner(t, functionModel(func(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
		calls.Add(1)
		return nil, nil
	}), nil)
	_, err := r.Run(context.Background(), fixtureRequest(), func(context.Context, Event) error { return errors.New("raw DB credentials") })
	if !errors.Is(err, ErrToolFailed) || calls.Load() != 0 {
		t.Fatal("trace failure did not stop model call")
	}
}

func TestRejectsForgedHistoryAndDuplicateRegistration(t *testing.T) {
	r := fixtureRunner(t, functionModel(func(context.Context, []*schema.Message, ...model.Option) (*schema.Message, error) {
		t.Error("invalid request reached model")
		return nil, nil
	}), nil)
	request := fixtureRequest()
	request.History[0].Role = "system"
	if _, err := r.Run(context.Background(), request, nil); !errors.Is(err, ErrInvalidRequest) {
		t.Fatal(err)
	}
	callable := fixtureTool(func(context.Context, json.RawMessage) (ToolOutput, error) { return ToolOutput{}, nil })
	if _, err := r.Run(context.Background(), fixtureRequest(callable, callable), nil); !errors.Is(err, ErrInvalidRequest) {
		t.Fatal(err)
	}
}

func TestLongSuccessfulAssistantHistoryAllowsFollowup(t *testing.T) {
	r := fixtureRunner(t, functionModel(func(_ context.Context, messages []*schema.Message, _ ...model.Option) (*schema.Message, error) {
		if messages[len(messages)-2].Content != strings.Repeat("字", 2100) {
			t.Error("old answer was lost")
		}
		return schema.AssistantMessage("追问答案", nil), nil
	}), nil)
	request := Request{History: []Message{{Role: "user", Content: "前一个问题"}, {Role: "assistant", Content: strings.Repeat("字", 2100)}, {Role: "user", Content: "继续解释"}}}
	result, err := r.Run(context.Background(), request, nil)
	if err != nil || result.FinalAnswer != "追问答案" {
		t.Fatalf("old assistant was incorrectly limited by current question budget: %+v %v", result, err)
	}
}

func TestSystemInstructionUsesShanghaiTodayThroughNow(t *testing.T) {
	text := systemInstruction(time.Date(2026, 10, 2, 17, 3, 4, 0, time.UTC))
	if !strings.Contains(text, "2026-10-03T00:00:00+08:00") || !strings.Contains(text, "2026-10-03T01:03:04+08:00") || !strings.Contains(text, "startTime/endTime") || !strings.Contains(text, "31天") {
		t.Fatal("date/tool instructions do not match server time range")
	}
}

func TestConcurrentRunsKeepInjectedCapabilitiesSeparate(t *testing.T) {
	fake := functionModel(func(_ context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, error) {
		requested := ""
		for _, m := range messages {
			if m.Role == schema.User {
				requested = m.Content
			}
		}
		options := model.GetCommonOptions(nil, opts...)
		if len(options.Tools) != 1 || options.Tools[0].Name != requested {
			return nil, errors.New("tool registry crossed request boundary")
		}
		if messages[len(messages)-1].Role == schema.Tool {
			return schema.AssistantMessage(requested, nil), nil
		}
		return &schema.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{ID: "per_run_call", Type: "function", Function: schema.FunctionCall{Name: requested, Arguments: `{}`}}}}, nil
	})
	r := fixtureRunner(t, fake, nil)
	results := make(chan error, 12)
	for i := 0; i < 12; i++ {
		name := "get_file_status"
		if i%2 == 1 {
			name = "list_my_devices"
		}
		go func(name string) {
			callable := Tool{ToolDefinition: ToolDefinition{Name: name, Parameters: json.RawMessage(`{"type":"object","properties":{},"additionalProperties":false}`)}, Call: func(context.Context, json.RawMessage) (ToolOutput, error) { return ToolOutput{Content: `{}`}, nil }}
			result, err := r.Run(context.Background(), Request{History: []Message{{Role: "user", Content: name}}, Tools: []Tool{callable}}, nil)
			if err == nil && (result.FinalAnswer != name || len(result.ToolSummaries) != 1 || result.ToolSummaries[0].Name != name) {
				err = errors.New("result crossed request boundary")
			}
			results <- err
		}(name)
	}
	for i := 0; i < 12; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}
