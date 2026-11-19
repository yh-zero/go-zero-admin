package aiservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
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

func TestContextBudgetPreservesNumberedSearchEvidence(t *testing.T) {
	search := schema.ToolCall{ID: "search", Function: schema.FunctionCall{Name: "kb_search", Arguments: "{}"}}
	other := schema.ToolCall{ID: "stats", Function: schema.FunctionCall{Name: "text_stats", Arguments: "{}"}}
	evidence := strings.Repeat("依据", 100)
	input := []*schema.Message{schema.UserMessage("问题"), schema.AssistantMessage("", []schema.ToolCall{search, other}), schema.ToolMessage(evidence, search.ID), schema.ToolMessage(strings.Repeat("大", 5000), other.ID)}
	fit, err := (&contextBudget{maxTokens: 1000}).fit(input, []byte("[]"), false)
	if err != nil || fit[2].Content != evidence || fit[3].Content == input[3].Content {
		t.Fatal("其它工具结果挤压预算时须完整保留已编号的摘录")
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

func (s *budgetKbSearcher) Search(_ context.Context, _ int64, _ int64, query string) ([]agenttools.KbSearchHit, error) {
	s.calls++
	hit := func(id int64) agenttools.KbSearchHit {
		return agenttools.KbSearchHit{ChunkID: id, KbID: 1, DocID: 2, Snippet: fmt.Sprintf("摘录%d", id)}
	}
	switch query {
	case "first":
		return []agenttools.KbSearchHit{hit(41), hit(63)}, nil
	case "second":
		return []agenttools.KbSearchHit{hit(63), hit(8)}, nil
	case "failed":
		return nil, errors.New("本地检索桩故障")
	case "quota":
		return nil, fmt.Errorf("检索未准入: %w", xerr.NewErrCodeMsg(429, "每日供应商请求额度已用完"))
	}
	return []agenttools.KbSearchHit{{KbID: 1, DocID: 2, DocTitle: "资料", Snippet: strings.Repeat("资", 900), Score: 1}}, nil
}

func TestAIStreamNumberedKbToolRounds(t *testing.T) {
	agenttools.RegisterIfAbsent(agenttools.NewKbSearch(&budgetKbSearcher{}))
	for _, tc := range []struct {
		query    string
		wantCode uint32
	}{{"second", 0}, {"failed", 503}, {"quota", 429}} {
		t.Run(tc.query, func(t *testing.T) {
			round := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				round++
				var body struct{ Messages []*schema.Message }
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				if round > 1 {
					var material struct {
						Hits []struct {
							Index   int
							ChunkID int64 `json:"chunk_id"`
						}
					}
					last := body.Messages[len(body.Messages)-1]
					want := [][]int64{{41, 63}, {63, 8}}[round-2]
					if last.Role != schema.Tool || json.Unmarshal([]byte(last.Content), &material) != nil || len(material.Hits) != 2 {
						t.Error("模型未收到本轮引用")
						return
					}
					for i, hit := range material.Hits {
						if hit.Index != round-1+i || hit.ChunkID != want[i] {
							t.Error("模型收到的引用编号与检索命中不一致")
						}
					}
				}
				w.Header().Set("Content-Type", "text/event-stream")
				delta, reason := map[string]any{"content": "见 [2] 和 [3]。"}, "stop"
				if round < 3 {
					query := []string{"first", tc.query}[round-1]
					delta = map[string]any{"tool_calls": []any{map[string]any{"index": 0, "id": fmt.Sprintf("call-%d", round), "type": "function", "function": map[string]any{"name": "kb_search", "arguments": fmt.Sprintf(`{"query":"%s"}`, query)}}}}
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
			collector := &agenttools.KbSourceCollector{}
			ctx := agenttools.WithKbSources(agenttools.WithActor(context.Background(), 7), collector)
			answer, _, err := askModelStream(ctx, client, []*schema.Message{schema.UserMessage("查找资料")}, func(*aipb.StreamChunk) error { return nil }, func(int64, []schema.ToolCall, bool) error { return nil }, 32000)
			if tc.wantCode != 0 {
				code, ok := xerr.AsCodeError(err)
				if !ok || code.GetErrCode() != tc.wantCode || round != 2 || answer != "" {
					t.Fatalf("预期检索终止并返回 %d，实际错误 %v，模型调用 %d 次", tc.wantCode, err, round)
				}
				return
			}
			record := callRecord{Version: 1, Sources: collector.Sources()}
			saved := messageProto(&model.MessageRow{Content: answer, Role: "assistant", Status: "succeeded", ToolCalls: record.marshal()})
			if err != nil || answer != "见 [2] 和 [3]。" || len(saved.Sources) != 3 || saved.Sources[1].ChunkId != 63 || saved.Sources[2].ChunkId != 8 {
				t.Fatal("消息读回的编号或摘录发生变化")
			}
		})
	}
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
