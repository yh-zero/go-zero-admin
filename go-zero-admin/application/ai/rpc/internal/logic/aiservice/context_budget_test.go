package aiservicelogic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/xerr"
)

type budgetResultTool struct{}

func (budgetResultTool) Name() string        { return "budget_result" }
func (budgetResultTool) Description() string { return "本地预算回归工具" }
func (budgetResultTool) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{}
}
func (budgetResultTool) Execute(context.Context, json.RawMessage) (string, error) {
	data, _ := json.Marshal(map[string]any{"hits": []any{map[string]any{"doc_id": 1, "snippet": strings.Repeat("\"资料", 12000)}}})
	return string(data), nil
}

func TestContextBudgetReservesInitialSpaceAndKeepsCurrentQuestion(t *testing.T) {
	messages := make([]*schema.Message, 0, 81)
	for i := 0; i < 40; i++ {
		messages = append(messages, schema.UserMessage(strings.Repeat("旧", 20)), schema.AssistantMessage(strings.Repeat("答", 20), nil))
	}
	messages = append(messages, schema.UserMessage("当前问题"))
	fit, err := (&contextBudget{maxTokens: 1000}).fit(messages, []byte("[]"), true)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(fit)
	if len(data)+2 > 1500 || fit[len(fit)-1].Content != "当前问题" || fit[0].Role != schema.User {
		t.Fatal("历史未为检索、工具和回答预留空间或破坏完整历史轮次")
	}
}

func TestContextBudgetDropsWholeOlderToolRoundAndKeepsLatestPairs(t *testing.T) {
	call := func(id, args string) schema.ToolCall {
		return schema.ToolCall{ID: id, Type: "function", Function: schema.FunctionCall{Name: "text_stats", Arguments: args}}
	}
	messages := []*schema.Message{
		schema.UserMessage("当前问题"),
		schema.AssistantMessage("", []schema.ToolCall{call("old", strings.Repeat("x", 2500))}), schema.ToolMessage("旧资料", "old"),
		schema.AssistantMessage("", []schema.ToolCall{call("new-a", "{}"), call("new-b", "{}")}), schema.ToolMessage(`{"result":"a"}`, "new-a"), schema.ToolMessage(`{"result":"b"}`, "new-b"),
	}
	fit, err := (&contextBudget{maxTokens: 1000}).fit(messages, []byte("[]"), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(fit) != 4 || fit[0].Content != "当前问题" || len(fit[1].ToolCalls) != 2 || fit[2].ToolCallID != "new-a" || fit[3].ToolCallID != "new-b" {
		t.Fatal("裁剪产生孤立工具调用/结果，或丢弃当前问题")
	}
}

func TestContextBudgetFailsOnlyWhenMandatoryInputCannotFit(t *testing.T) {
	for _, test := range []struct{ name, question, definitions, args string }{
		{"current-question", strings.Repeat("长", 1000), "[]", ""},
		{"tool-definitions", "问题", strings.Repeat("x", 3000), ""},
		{"mandatory-tool-call", "问题", "[]", strings.Repeat("x", 3000)},
	} {
		t.Run(test.name, func(t *testing.T) {
			messages := []*schema.Message{schema.UserMessage(test.question)}
			if test.args != "" {
				messages = append(messages, schema.AssistantMessage("", []schema.ToolCall{{ID: "call", Type: "function", Function: schema.FunctionCall{Name: "text_stats", Arguments: test.args}}}), schema.ToolMessage(`{"ok":true}`, "call"))
			}
			_, err := (&contextBudget{maxTokens: 1000}).fit(messages, []byte(test.definitions), false)
			ce, ok := xerr.AsCodeError(err)
			if !ok || ce.GetErrCode() != 400 {
				t.Fatalf("不可裁剪输入必须明确返回预算错误，got %v", err)
			}
		})
	}
}

// 旧历史挤满预算时，仍须为工具结果留空间并保留合法 JSON 和调用配对。
func TestAIStreamBudgetLongHistoryAndLargeJSONTool(t *testing.T) {
	agenttools.RegisterIfAbsent(budgetResultTool{})
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body struct {
			Messages []*schema.Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if requests == 2 {
			if len(body.Messages) < 3 || body.Messages[len(body.Messages)-3].Content != "当前问题" {
				t.Error("当前问题丢失")
				return
			}
			assistant, result := body.Messages[len(body.Messages)-2], body.Messages[len(body.Messages)-1]
			if len(assistant.ToolCalls) != 1 || result.ToolCallID != assistant.ToolCalls[0].ID {
				t.Error("工具调用和结果失配")
			}
			if !json.Valid([]byte(result.Content)) {
				t.Error("结构化工具结果被截成非法 JSON")
			}
			if len(result.Content) > 12000 {
				t.Error("工具结果未按动态预算压缩")
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if requests == 1 {
			writeModelFrame(t, w, map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": "budget-call", "type": "function", "function": map[string]any{"name": "budget_result", "arguments": "{}"}}}}, "finish_reason": "tool_calls"}}})
		} else {
			writeModelFrame(t, w, map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"content": "预算回答"}, "finish_reason": "stop"}}})
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	client, err := llm.New(llm.Config{BaseURL: server.URL + "/v1", APIKey: "local-test-only", Model: "local-test", Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	messages := []*schema.Message{schema.UserMessage(strings.Repeat("旧", 6000)), schema.AssistantMessage("旧回答", nil), schema.UserMessage("当前问题")}
	answer, _, err := askModelStream(context.Background(), client, messages, func(*aipb.StreamChunk) error { return nil }, func(int64, []schema.ToolCall, bool) error { return nil }, 4000)
	if err != nil || answer != "预算回答" || requests != 2 {
		t.Fatalf("长历史+大工具结果应正常回答：answer=%q requests=%d err=%v", answer, requests, err)
	}
}

type budgetKbSearcher struct{ calls int }

func (s *budgetKbSearcher) Search(context.Context, int64, int64, string) ([]agenttools.KbSearchHit, error) {
	s.calls++
	return []agenttools.KbSearchHit{{KbID: 1, DocID: 2, DocTitle: "资料", Snippet: strings.Repeat("资", 900), Score: 1}}, nil
}

// 同一次发送的多轮检索必须共享额度，不能每次重新取得 25%。
func TestAIStreamBudgetRepeatedKbSearchSharesAllowance(t *testing.T) {
	searcher := &budgetKbSearcher{}
	agenttools.RegisterIfAbsent(agenttools.NewKbSearch(searcher))
	requests, totalResultBytes := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var body struct {
			Messages []*schema.Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if requests > 1 {
			totalResultBytes += len(body.Messages[len(body.Messages)-1].Content)
		}
		encoded, _ := json.Marshal(body.Messages)
		definitions, _ := json.Marshal(agenttools.LLMTools())
		if len(encoded)+len(definitions) > 10500 {
			t.Error("模型请求未给回答预留空间")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		delta := map[string]any{"content": "检索回答"}
		reason := "stop"
		if requests <= 3 {
			delta = map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("kb-%d", requests), "type": "function", "function": map[string]any{"name": "kb_search", "arguments": `{"query":"资料"}`}}}}
			reason = "tool_calls"
		}
		writeModelFrame(t, w, map[string]any{"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": reason}}})
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	client, err := llm.New(llm.Config{BaseURL: server.URL + "/v1", APIKey: "local-test-only", Model: "local-test", Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	messages := []*schema.Message{schema.UserMessage(strings.Repeat("旧", 6000)), schema.AssistantMessage("旧回答", nil), schema.UserMessage("当前问题")}
	answer, _, err := askModelStream(agenttools.WithActor(context.Background(), 7), client, messages, func(*aipb.StreamChunk) error { return nil }, func(int64, []schema.ToolCall, bool) error { return nil }, 4000)
	if err != nil || answer != "检索回答" {
		t.Fatalf("重复检索应安全缩减并继续回答：%v", err)
	}
	// 结果消息中的预算耗尽提示允许少量额外字节，资料结果总额仍不得超过 3000 字节。
	if totalResultBytes > 3300 {
		t.Fatalf("三次检索未共享 25%% 预算：结果累计 %d 字节", totalResultBytes)
	}
}
