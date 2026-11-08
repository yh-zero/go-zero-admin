package agenttools

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
)

type candidateAnalysisSearcher struct {
	hits                       []KbSearchHit
	searchCalls, analysisCalls int
	owner, kb                  int64
	query                      string
}

func (s *candidateAnalysisSearcher) Search(context.Context, int64, int64, string) ([]KbSearchHit, error) {
	s.searchCalls++
	return s.hits[:min(len(s.hits), 5)], nil
}
func (s *candidateAnalysisSearcher) SearchAnalysis(_ context.Context, owner, kb int64, query string) ([]KbSearchHit, error) {
	s.analysisCalls++
	s.owner, s.kb, s.query = owner, kb, query
	return s.hits, nil
}
func analysisCandidateHits(n int) []KbSearchHit {
	hits := make([]KbSearchHit, n)
	for i := range hits {
		hits[i] = KbSearchHit{ChunkID: int64(101 + i), KbID: 3, DocID: int64(201 + i), DocTitle: fmt.Sprintf("资料%d", i+1), Snippet: fmt.Sprintf("第%d项实际事实。", i+1), Score: 1 - float32(i)/100}
	}
	return hits
}

func TestKbAnalyzeUsesAnalysisCandidatesAndRenumbersOnlyCitations(t *testing.T) {
	s := &candidateAnalysisSearcher{hits: analysisCandidateHits(20)}
	_, c := analysisFixture()
	c.response.Content = `{"answers":[{"target":"字段[12]","state":"supported","text":"两项实际事实。","citations":[12,3]}]}`
	out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"分析字段[12]"}`))
	if err != nil {
		t.Fatalf("低排名真实候选必须可引用: %v", err)
	}
	var got KbAnalysisResult
	var sent struct {
		Query   string             `json:"query"`
		Sources []KbAnalysisSource `json:"sources"`
	}
	if json.Unmarshal([]byte(out), &got) != nil || json.Unmarshal([]byte(c.messages[1].Content), &sent) != nil {
		t.Fatal("结果无效")
	}
	if s.searchCalls != 0 || s.analysisCalls != 1 || s.owner != 7 || s.kb != 3 || s.query != "分析字段[12]" || c.calls != 1 || len(sent.Sources) != 20 {
		t.Fatal("应保留一次专用候选检索及一次分析")
	}
	if len(got.Sources) != 2 || got.Sources[0].ChunkID != 103 || got.Sources[1].ChunkID != 112 || got.Sources[0].Index != 1 || got.Sources[1].Index != 2 {
		t.Fatalf("只返回被引真实候选并按原候选顺序重编号: %+v", got.Sources)
	}
	if got.Sources[0].KbSearchHit != sent.Sources[2].KbSearchHit || got.Sources[1].KbSearchHit != sent.Sources[11].KbSearchHit {
		t.Fatal("不能在回答后缩短或拼造来源")
	}
	if !strings.Contains(got.Analysis, `字段\[12\]`) || !strings.Contains(got.Analysis, "[2][1]") {
		t.Fatalf("只映射引用整数，不能全局替换用户字段编号: %q", got.Analysis)
	}
}

func TestKbAnalyzeCandidateSelectionRejectsGlobalNineReferences(t *testing.T) {
	s := &candidateAnalysisSearcher{hits: analysisCandidateHits(20)}
	_, c := analysisFixture()
	c.response.Content = `{"answers":[{"target":"甲","state":"supported","text":"第一组事实。","citations":[1,2,3,4,5]},{"target":"乙","state":"supported","text":"第二组事实。","citations":[6,7,8,9]}]}`
	out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"甲和乙"}`))
	if err == nil || out != "" || s.analysisCalls != 1 || c.calls != 1 {
		t.Fatal("整条回复引用九个不同来源须安全失败且不能重试")
	}
}

func TestKbAnalyzeCandidateFallbackKeepsInsufficientAndTruncatedEvidence(t *testing.T) {
	for _, truncated := range []bool{false, true} {
		t.Run(fmt.Sprint(truncated), func(t *testing.T) {
			s := &candidateAnalysisSearcher{hits: analysisCandidateHits(20)}
			_, c := analysisFixture()
			c.response.Content = `{"answers":[{"target":"实际电话","state":"insufficient","text":"","citations":[]}]}`
			if truncated {
				c.response.Content = `{"answers":[{"target":"实际电话","state":"supported","text":"private-partial`
				c.response.ResponseMeta = &schema.ResponseMeta{FinishReason: "length"}
			}
			out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"实际电话"}`))
			var got KbAnalysisResult
			if err != nil || json.Unmarshal([]byte(out), &got) != nil || len(got.Sources) != 8 || got.Status == "no_matches" || c.calls != 1 || !got.Usage.Complete || got.Usage.TotalTokens != 120 || strings.Contains(got.Analysis, "private-partial") {
				t.Fatalf("未引用/截断仍保留真实检索参考和已发生用量: %v", err)
			}
			if s.analysisCalls != 1 {
				t.Fatal("必须实际使用分析候选而非普通前五")
			}
		})
	}
}

func TestKbAnalyzeCandidateBudgetsBoundAnyEightWithoutPosthocTruncation(t *testing.T) {
	hits := analysisCandidateHits(20)
	for i := range hits {
		hits[i].DocTitle = strings.Repeat("题", 80)
		hits[i].Snippet = strings.Repeat("中文\"\\\n", 800+i*100)
	}
	tool := &kbAnalyzeTool{maxContextTokens: 32000}
	messages, sources, err := tool.fitSources("原目标保持完整", 3, hits, 2048)
	if err != nil || len(sources) != 20 {
		t.Fatalf("大预算应容纳20候选: %v count=%d", err, len(sources))
	}
	serialized, _ := json.Marshal(messages)
	sourceBytes, _ := json.Marshal(sources)
	if (len(serialized)+2)/3+2048 > 32000 || len(sourceBytes) > 24000 {
		t.Fatal("候选输入必须有界")
	}
	var contentBytes int
	for _, s := range sources {
		if !utf8.ValidString(s.Snippet) {
			t.Fatal("UTF8被截坏")
		}
		contentBytes += len(s.Snippet)
	}
	if contentBytes > 32000/4*3 {
		t.Fatal("候选正文不能越过25%片段预算")
	}
	var checkSet func(int, []KbAnalysisSource)
	checkSet = func(start int, selected []KbAnalysisSource) {
		if len(selected) == 8 {
			data, _ := json.Marshal(selected)
			if len(data) > 10000 {
				t.Fatal("任意8条最终引用不能回答后再截断依据")
			}
			return
		}
		for i := start; i <= len(sources)-(8-len(selected)); i++ {
			source := sources[i]
			source.Index = len(selected) + 1
			checkSet(i+1, append(selected, source))
		}
	}
	checkSet(0, nil)
	tool.maxContextTokens = 1600
	messages, sources, err = tool.fitSources(strings.Repeat("目标", 200), 3, hits, 400)
	if err != nil || len(sources) == 0 || len(sources) >= 20 {
		t.Fatalf("小预算应减少候选数量而非丢完整目标: %v count=%d", err, len(sources))
	}
	serialized, _ = json.Marshal(messages)
	if (len(serialized)+2)/3+400 > 1600 || !strings.Contains(messages[1].Content, strings.Repeat("目标", 200)) {
		t.Fatal("小预算必须保留原问和回答预留")
	}
}

func TestKbAnalyzeCandidatesRejectInvalidUnreferencedSourceBeforeChat(t *testing.T) {
	for _, kind := range []string{"foreign_kb", "duplicate", "invalid_utf8", "score"} {
		t.Run(kind, func(t *testing.T) {
			s := &candidateAnalysisSearcher{hits: analysisCandidateHits(20)}
			_, c := analysisFixture()
			switch kind {
			case "foreign_kb":
				s.hits[19].KbID = 4
			case "duplicate":
				s.hits[19].ChunkID = s.hits[0].ChunkID
			case "invalid_utf8":
				s.hits[19].Snippet = string([]byte{0xff})
			case "score":
				s.hits[19].Score = float32(math.NaN())
			}
			out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`))
			if err == nil || out != "" || s.analysisCalls != 1 || c.calls != 0 {
				t.Fatal("不能因模型未引用就忽略非法候选")
			}
		})
	}
}

func TestKbAnalyzeCandidatesValidateWholeReplyBeforeSelection(t *testing.T) {
	s := &candidateAnalysisSearcher{hits: analysisCandidateHits(20)}
	_, c := analysisFixture()
	c.response.Content = `{"answers":[{"target":"发布条件","state":"supported","text":"发布需审批。","citations":[12]},{"target":"直接上线","state":"insufficient","text":"","citations":[]}]}`
	out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"有人建议“直接上线”。请说明发布条件。"}`))
	if err == nil || out != "" || s.analysisCalls != 1 || c.calls != 1 {
		t.Fatal("裁剪未引用候选不能绕过背景问项守卫")
	}
}

func TestKbAnalyzeCandidatesOwnedGateAndValidLengthReply(t *testing.T) {
	s := &candidateAnalysisSearcher{hits: analysisCandidateHits(20)}
	_, c := analysisFixture()
	if _, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return fmt.Errorf("private") }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`)); err == nil || s.analysisCalls != 0 || s.searchCalls != 0 || c.calls != 0 {
		t.Fatal("专用候选入口不能绕过归属检查")
	}
	c.response.Content = `{"answers":[{"target":"目标","state":"supported","text":"实际事实。","citations":[20]}]}`
	c.response.ResponseMeta = &schema.ResponseMeta{FinishReason: "length"}
	out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`))
	var got KbAnalysisResult
	if err != nil || json.Unmarshal([]byte(out), &got) != nil || got.Status != "truncated" || len(got.Sources) != 1 || got.Sources[0].ChunkID != 120 || !strings.HasSuffix(got.Analysis, "[1]") || !got.Usage.Complete || c.calls != 1 {
		t.Fatal("合法完整结构即使length终态也须保留实际引用与原用量")
	}
}

func TestKbAnalyzeCandidatesIncludeMetadataAndEscapesInQuarterBudget(t *testing.T) {
	for _, contextTokens := range []int64{2000, 4000} {
		t.Run(fmt.Sprint(contextTokens), func(t *testing.T) {
			hits := analysisCandidateHits(20)
			for i := range hits {
				hits[i].DocTitle = strings.Repeat("题", 80)
				hits[i].Snippet = strings.Repeat("\"\\\n", 100)
			}
			tool := &kbAnalyzeTool{maxContextTokens: contextTokens}
			messages, sources, err := tool.fitSources("保留原目标", 3, hits, int(contextTokens/4))
			if err != nil || len(sources) == 0 {
				t.Fatalf("小预算应回退候选数: %v", err)
			}
			encoded, _ := json.Marshal(sources)
			if int64(len(encoded)) > contextTokens/4*3 {
				t.Fatalf("元数据和转义必须计入四分之一来源JSON额度: count=%d bytes=%d limit=%d", len(sources), len(encoded), contextTokens/4*3)
			}
			serialized, _ := json.Marshal(messages)
			if int64((len(serialized)+2)/3)+contextTokens/4 > contextTokens {
				t.Fatal("来源预算不能代替完整消息与输出预留核对")
			}
		})
	}
}

func TestKbAnalysisCitationFailureReasonsDistinguishGlobalSourceLimit(t *testing.T) {
	sources := make([]KbAnalysisSource, 20)
	for i, hit := range analysisCandidateHits(20) {
		sources[i] = KbAnalysisSource{Index: i + 1, KbSearchHit: hit}
	}
	for _, tc := range []struct{ name, raw, want string }{
		{name: "全局九来源", raw: `{"answers":[{"target":"甲","state":"supported","text":"事实一。","citations":[1,2,3,4,5]},{"target":"乙","state":"supported","text":"事实二。","citations":[6,7,8,9]}]}`, want: "内部回复引用来源超过8条"},
		{name: "引用越界", raw: `{"answers":[{"target":"甲","state":"supported","text":"事实。","citations":[21]}]}`, want: "内部回复引用编号无效或重复"},
		{name: "同问项重复编号", raw: `{"answers":[{"target":"甲","state":"supported","text":"事实。","citations":[1,1]}]}`, want: "内部回复引用编号无效或重复"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text, selected, err := renderKbAnalysisWithSources(tc.raw, "甲和乙", sources)
			if err == nil || text != "" || selected != nil || analysisResponseFailureReason(err) != tc.want {
				t.Fatalf("必须静态区分失败分支而不删除引用或放宽限制: err=%v reason=%q", err, analysisResponseFailureReason(err))
			}
		})
	}
}

func TestKbAnalyzeAcceptsSixAndEightRealReferencesWithoutDroppingEvidence(t *testing.T) {
	for _, count := range []int{6, 8} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			s := &candidateAnalysisSearcher{hits: analysisCandidateHits(20)}
			_, c := analysisFixture()
			citations := make([]int, count)
			for i := range citations {
				citations[i] = i + 1
			}
			raw, _ := json.Marshal(struct {
				Answers []kbModelAnswer `json:"answers"`
			}{[]kbModelAnswer{{Target: "目标", State: "supported", Text: "各项有据事实。", Citations: citations}}})
			c.response.Content = string(raw)
			out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`))
			var got KbAnalysisResult
			if err != nil || json.Unmarshal([]byte(out), &got) != nil || len(got.Sources) != count || c.calls != 1 {
				t.Fatalf("合法%d来源不能截丢第六来源或安全失败: %v", count, err)
			}
			for i, source := range got.Sources {
				if source.Index != i+1 || source.ChunkID != int64(101+i) || !strings.Contains(got.Analysis, fmt.Sprintf("[%d]", i+1)) {
					t.Fatal("真实引用身份或编号丢失")
				}
			}
		})
	}
}
