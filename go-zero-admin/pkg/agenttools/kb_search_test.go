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

type documentSearchFixture struct {
	budgetTestSearcher
	owner, kb, doc int64
	returnedDoc    int64
}

func (s *documentSearchFixture) SearchDocument(_ context.Context, owner, kb, doc int64, _ string) ([]KbSearchHit, error) {
	s.owner, s.kb, s.doc = owner, kb, doc
	return []KbSearchHit{{KbID: kb, DocID: s.returnedDoc, ChunkID: 3, DocVersion: 4, Snippet: "住宿上限 300 元"}}, nil
}

func TestKbSearchDocumentScopeFailsClosed(t *testing.T) {
	ctx := WithActor(context.Background(), 9)
	collector := &KbSourceCollector{}
	ctx = WithKbSources(WithKbDocumentScope(ctx, 11, 17), collector)
	args := json.RawMessage(`{"query":"住宿上限","kb_id":99}`)
	s := &documentSearchFixture{returnedDoc: 17}
	if _, err := NewKbSearch(s).Execute(ctx, args); err != nil || s.owner != 9 || s.kb != 11 || s.doc != 17 || s.calls != 0 {
		t.Fatal("文档范围须取自已授权的服务端上下文", err)
	}
	if hits := collector.Sources(); len(hits) != 1 || hits[0].DocVersion != 4 {
		t.Fatal("来源快照丢失版本")
	}
	s.returnedDoc++
	if _, err := NewKbSearch(s).Execute(ctx, args); err == nil {
		t.Fatal("禁止向模型返回其他资料")
	}
	broadOnly := &budgetTestSearcher{}
	if _, err := NewKbSearch(broadOnly).Execute(ctx, args); err == nil || broadOnly.calls != 0 {
		t.Fatal("缺少文档检索能力须拒绝，不能扩大到整个库")
	}
	if _, err := NewKbSearch(s).Execute(WithKbDocumentScope(ctx, 0, 0), args); err == nil {
		t.Fatal("无效文档范围须明确失败")
	}
}

func TestKbCitationsKeepTheirIndexAcrossSearches(t *testing.T) {
	c := &KbSourceCollector{}
	makeHit := func(id int64, snippet string) KbSearchHit {
		return KbSearchHit{KbID: 1, DocID: 1, ChunkID: id, Snippet: snippet}
	}
	for round, hits := range [][]KbSearchHit{
		{makeHit(7, "证据 A"), makeHit(6, "证据 B")},
		{makeHit(6, "已改写 B"), makeHit(5, "证据 C"), makeHit(7, "已改写 A"), makeHit(5, "重复 C")},
	} {
		result, err := c.render(hits, 16000)
		var material struct{ Hits []numberedKbHit }
		if err != nil || json.Unmarshal([]byte(result), &material) != nil {
			t.Fatal("引用材料不可读")
		}
		want := [][]int{{1, 2}, {2, 3, 1}}[round]
		if len(material.Hits) != len(want) {
			t.Fatalf("round %d: hits=%d", round, len(material.Hits))
		}
		for i, hit := range material.Hits {
			if hit.Index != want[i] || c.Sources()[hit.Index-1] != hit.KbSearchHit || !strings.HasPrefix(hit.Snippet, "证据") {
				t.Fatalf("round %d: 引用与首次展示或持久化内容不符", round)
			}
		}
	}
	limits := []int{2, 80, 300, 600}
	for _, limit := range limits {
		partial := &KbSourceCollector{}
		result, err := partial.render([]KbSearchHit{makeHit(1, strings.Repeat("文", 999)), makeHit(2, "two")}, limit)
		var material struct{ Hits []numberedKbHit }
		if err != nil || len(result) > limit || json.Unmarshal([]byte(result), &material) != nil || len(material.Hits) != len(partial.Sources()) {
			t.Fatalf("字节预算 %d 内出现无效或不可见的来源", limit)
		}
	}
}

type scopedChatSearcher struct{ seenKb int64 }

type citationSearchResults []KbSearchHit

func (s *citationSearchResults) Search(context.Context, int64, int64, string) ([]KbSearchHit, error) {
	return *s, nil
}

func TestKbSearchCompactionDoesNotCarryOverOptionalDocVersion(t *testing.T) {
	first := KbSearchHit{KbID: 1, DocID: 2, ChunkID: 1, DocVersion: 73, Snippet: strings.Repeat("a", 600)}
	second := KbSearchHit{KbID: 1, DocID: 3, ChunkID: 2, Snippet: strings.Repeat("b", 900)}
	const tokens = 396
	c := &KbSourceCollector{}
	ctx := WithKbSources(WithKbSearchBudget(WithActor(context.Background(), 1), tokens), c)
	results := citationSearchResults{first}
	tool := NewKbSearch(&results)
	args := json.RawMessage(`{"query":"检索"}`)
	initial, err := tool.Execute(ctx, args)
	if err != nil || len(c.Sources()) != 1 || c.Sources()[0] != first {
		t.Fatalf("首轮未按原文登记版本 73: %v", err)
	}
	results = citationSearchResults{first, second}
	raw, err := tool.Execute(ctx, args)
	var result kbSearchResult
	if err != nil || json.Unmarshal([]byte(raw), &result) != nil || !result.Truncated || len(result.Hits) != 1 {
		t.Fatalf("第二轮应只展示新分块的有界摘录: %+v %v", result, err)
	}
	hit := result.Hits[0]
	if hit.Index != 2 || hit.ChunkID != second.ChunkID || hit.DocVersion != 0 || hit.Snippet == "" || !strings.HasPrefix(second.Snippet, hit.Snippet) {
		t.Fatalf("新分块引用不符: index=%d chunk=%d version=%d bytes=%d", hit.Index, hit.ChunkID, hit.DocVersion, len(hit.Snippet))
	}
	sources := c.Sources()
	if len(sources) != 2 || sources[0] != first || sources[1] != hit.KbSearchHit || len(initial)+len(raw) > tokens*3 {
		t.Fatalf("展示内容、来源登记或总预算不符: sources=%d bytes=%d", len(sources), len(initial)+len(raw))
	}
}

func TestKbSearchRepeatPreservesVisibleSnapshotWithinBudget(t *testing.T) {
	first := KbSearchHit{KbID: 1, DocID: 2, ChunkID: 3, DocVersion: 4, DocTitle: "原始标题", Snippet: strings.Repeat("证", 300)}
	changed := first
	changed.DocVersion, changed.DocTitle, changed.Snippet = 5, "新版标题", "改写摘录"
	next := KbSearchHit{KbID: 1, DocID: 2, ChunkID: 8, Snippet: "新证据"}
	for _, tc := range []struct {
		name        string
		tokens      int64
		second      citationSearchResults
		want        []numberedKbHit
		sourceCount int
		truncated   bool
	}{
		{"可完整复用", 1500, citationSearchResults{changed, next, next}, []numberedKbHit{{first, 1}, {next, 2}}, 2, false},
		{"预算只能容纳新命中", 627, citationSearchResults{changed, next, next}, []numberedKbHit{{next, 2}}, 2, true},
		{"预算无法重发已有命中", 627, citationSearchResults{changed}, nil, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := &KbSourceCollector{}
			ctx := WithKbSources(WithKbSearchBudget(WithActor(context.Background(), 1), tc.tokens), c)
			results := citationSearchResults{first}
			tool := NewKbSearch(&results)
			query := json.RawMessage(`{"query":"检索"}`)
			used := 0
			for round, want := range [][]numberedKbHit{{{first, 1}}, tc.want} {
				raw, err := tool.Execute(ctx, query)
				used += len(raw)
				var result struct {
					Hits      []numberedKbHit
					Truncated bool
				}
				if err != nil || json.Unmarshal([]byte(raw), &result) != nil || len(result.Hits) != len(want) {
					t.Fatalf("第 %d 轮返回 %d 条，期望 %d 条: %v", round+1, len(result.Hits), len(want), err)
				}
				if used > int(tc.tokens*3) || result.Truncated != (round == 1 && tc.truncated) {
					t.Fatalf("第 %d 轮预算或截断标记不符: bytes=%d truncated=%t", round+1, used, result.Truncated)
				}
				for i, hit := range result.Hits {
					if hit != want[i] || c.Sources()[hit.Index-1] != hit.KbSearchHit {
						t.Fatalf("第 %d 轮引用 %d 与首次展示的快照不符", round+1, hit.Index)
					}
				}
				results = tc.second
			}
			sources := c.Sources()
			if len(sources) != tc.sourceCount || sources[0] != first {
				t.Fatal("保存了未展示的引用或覆盖了首轮快照")
			}
			results = nil
			if raw, err := tool.Execute(ctx, query); err != nil || !strings.Contains(raw, "未检索到") || len(c.Sources()) != len(sources) {
				t.Fatalf("空召回改变了来源或工具回执: %s %v", raw, err)
			}
		})
	}
}

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
