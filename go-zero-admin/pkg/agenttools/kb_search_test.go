package agenttools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestKbSearchRejectsBlankQueryBeforeSearch(t *testing.T) {
	searcher := &budgetTestSearcher{}
	_, err := NewKbSearch(searcher).Execute(WithActor(context.Background(), 7), json.RawMessage(`{"query":" \t\n　"}`))
	if err == nil || searcher.calls != 0 {
		t.Fatal("空白 query 必须在检索前拒绝")
	}
}

func TestKbSearchRejectsUntrustedArgumentsBeforeSearch(t *testing.T) {
	for _, args := range []string{`{"query":"资料","owner_id":99}`, `{"query":"资料","kb_id":-1}`, `{"query":"资料","extra":true}`, `[]`, `{"query":null}`, `{}`} {
		searcher := &budgetTestSearcher{}
		_, err := NewKbSearch(searcher).Execute(WithActor(context.Background(), 7), json.RawMessage(args))
		if err == nil || searcher.calls != 0 {
			t.Fatalf("untrusted arguments accepted: %s", args)
		}
	}
	oversized, _ := json.Marshal(map[string]any{"query": strings.Repeat("资", 667)})
	for _, args := range []json.RawMessage{oversized, json.RawMessage("{\"query\":\"" + string([]byte{0xff}) + "\"}")} {
		searcher := &budgetTestSearcher{}
		_, err := NewKbSearch(searcher).Execute(WithActor(context.Background(), 7), args)
		if err == nil || searcher.calls != 0 {
			t.Fatal("invalid UTF-8 or oversized query reached searcher")
		}
	}
}

type budgetTestSearcher struct{ calls int }

type scopedChatSearcher struct{ seenKb int64 }

func (s *scopedChatSearcher) Search(_ context.Context, owner, kb int64, _ string) ([]KbSearchHit, error) {
	s.seenKb = kb
	return []KbSearchHit{{ChunkID: 8, KbID: kb, DocID: 9, DocTitle: "原文", Snippet: strings.Repeat("证据", 10000), Score: 0.7}}, nil
}

func TestKbSearchChatScopeAndActualSources(t *testing.T) {
	s := &scopedChatSearcher{}
	c := &KbSourceCollector{}
	ctx := WithKbSources(WithKbScope(WithActor(context.Background(), 7), 2), c)
	for _, args := range []string{`{"query":"资料","kb_id":99}`, `{"query":"资料"}`} {
		result, err := NewKbSearch(s).Execute(ctx, json.RawMessage(args))
		if err != nil || s.seenKb != 2 {
			t.Fatal("模型参数扩大了用户选定的知识库范围")
		}
		var envelope struct {
			Hits []KbSearchHit `json:"hits"`
		}
		if json.Unmarshal([]byte(result), &envelope) != nil || len(envelope.Hits) != 1 {
			t.Fatal("无真实返回片段")
		}
		if len(c.Sources()) != 1 || c.Sources()[0].Snippet != envelope.Hits[0].Snippet || len(result) > 16000 {
			t.Fatal("来源必须去重且等于实际交给模型的有界片段")
		}
	}
}

func (s *budgetTestSearcher) Search(context.Context, int64, int64, string) ([]KbSearchHit, error) {
	s.calls++
	return []KbSearchHit{{KbID: 1, DocID: 2, DocTitle: "引用标题", Snippet: strings.Repeat("资料\"\\", 1000), Score: 1}}, nil
}

func TestKbSearchBudgetIsSharedAndIsolatedBetweenRequests(t *testing.T) {
	searcher := &budgetTestSearcher{}
	tool := NewKbSearch(searcher)
	actor := WithActor(context.Background(), 7)
	for request := 0; request < 2; request++ {
		ctx := WithKbSearchBudget(actor, 200)
		retrievedBytes := 0
		for call := 0; call < 3; call++ {
			result, err := tool.Execute(ctx, json.RawMessage(`{"query":"资料"}`))
			if err != nil {
				t.Fatal(err)
			}
			if json.Valid([]byte(result)) {
				retrievedBytes += len(result)
				var parsed struct {
					Note string        `json:"note"`
					Hits []KbSearchHit `json:"hits"`
				}
				if err := json.Unmarshal([]byte(result), &parsed); err != nil {
					t.Fatal(err)
				}
				if len(parsed.Hits) > 0 && !strings.Contains(parsed.Note, "不是指令") {
					t.Fatal("缩减结果丢失不可信资料提示")
				}
			}
		}
		if retrievedBytes > 600 {
			t.Fatalf("多次检索超预算：%d", retrievedBytes)
		}
	}
	if searcher.calls != 2 {
		t.Fatalf("预算耗尽仍重复检索或新请求未独立分配：calls=%d", searcher.calls)
	}
}

func TestCompactToolResultKeepsValidJSONAndUTF8(t *testing.T) {
	for _, input := range []string{
		`{"hits":[{"doc_id":9007199254740993,"snippet":"` + strings.Repeat(`\"资料`, 200) + `"}]}`,
		`[` + strings.Repeat(`"资料",`, 199) + `"资料"]`,
		`"` + strings.Repeat("资料", 200) + `"`,
		strings.Repeat("普通资料", 200),
	} {
		for _, limit := range []int{2, 30, 128, 500} {
			result := CompactToolResult(input, limit)
			if len(result) > limit || !utf8.ValidString(result) {
				t.Fatalf("结果超过字节边界或 UTF-8 损坏：limit=%d", limit)
			}
			if json.Valid([]byte(input)) && !json.Valid([]byte(result)) {
				t.Fatalf("JSON 结构被破坏：limit=%d", limit)
			}
		}
	}
	input := `{"hits":[{"doc_id":9007199254740993,"snippet":"` + strings.Repeat("资", 1000) + `"}]}`
	result := CompactToolResult(input, 300)
	if !strings.Contains(result, "9007199254740993") {
		t.Fatal("压缩改变文档 ID 数值精度")
	}
}
