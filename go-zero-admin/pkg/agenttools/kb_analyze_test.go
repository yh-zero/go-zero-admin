package agenttools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
)

type analysisSearcher struct {
	calls int
	hits  []KbSearchHit
	err   error
}

func (s *analysisSearcher) Search(_ context.Context, owner, kb int64, query string) ([]KbSearchHit, error) {
	s.calls++
	if owner != 7 || kb != 3 || strings.TrimSpace(query) == "" {
		panic("身份或目标错误")
	}
	return s.hits, s.err
}

type analysisChat struct {
	calls     int
	messages  []*schema.Message
	maxTokens int
	response  *schema.Message
	usage     *schema.TokenUsage
	err       error
}

func (c *analysisChat) ChatBounded(_ context.Context, m []*schema.Message, maxTokens int) (*schema.Message, *schema.TokenUsage, error) {
	c.calls++
	c.messages = m
	c.maxTokens = maxTokens
	return c.response, c.usage, c.err
}
func analysisFixture() (*analysisSearcher, *analysisChat) {
	return &analysisSearcher{hits: []KbSearchHit{{ChunkID: 1, KbID: 3, DocID: 2, DocTitle: "资料", Snippet: "预算为五万元。", Score: .9}}}, &analysisChat{response: schema.AssistantMessage("预算为五万元。[1]", nil), usage: &schema.TokenUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120}}
}

func TestKbAnalyzeRejectsArgumentsBeforeDependencies(t *testing.T) {
	for _, raw := range []string{`{}`, `{"query":"目标"}`, `{"kb_id":0,"query":"目标"}`, `{"kb_id":-1,"query":"目标"}`, `{"kb_id":1.5,"query":"目标"}`, `{"kb_id":3,"query":"  "}`, `{"kb_id":3,"query":null}`, `{"kb_id":3,"query":"目标","owner_id":7}`, `[]`, `null`, `{"kb_id":3,"query":"` + strings.Repeat("中", 667) + `"}`, string([]byte{'{', 0xff, '}'})} {
		t.Run(raw[:min(len(raw), 30)], func(t *testing.T) {
			s, c := analysisFixture()
			owned := 0
			tool := NewKbAnalyze(s, func(context.Context, int64, int64) error { owned++; return nil }, c, 32000)
			out, err := tool.Execute(WithActor(context.Background(), 7), []byte(raw))
			if err == nil || out != "" || s.calls != 0 || c.calls != 0 || owned != 0 {
				t.Fatalf("非法参数必须零外部调用: err=%v owned=%d search=%d chat=%d", err, owned, s.calls, c.calls)
			}
		})
	}
}
func TestKbAnalyzeOwnershipPrecedesSearchAndNoMatchesSkipsChat(t *testing.T) {
	s, c := analysisFixture()
	owned := 0
	tool := NewKbAnalyze(s, func(_ context.Context, kb, owner int64) error {
		owned++
		if kb != 3 || owner != 7 {
			t.Fatal("归属身份错误")
		}
		return errors.New("private vendor detail")
	}, c, 32000)
	if _, err := tool.Execute(context.Background(), []byte(`{"kb_id":3,"query":"目标"}`)); err == nil || owned != 0 {
		t.Fatal("缺失身份必须拒绝")
	}
	if _, err := tool.Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`)); err == nil || strings.Contains(err.Error(), "private") || s.calls != 0 || c.calls != 0 {
		t.Fatal("越权必须在检索前安全失败")
	}
	s.hits = nil
	tool = NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000)
	out, err := tool.Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`))
	var result KbAnalysisResult
	if err != nil || json.Unmarshal([]byte(out), &result) != nil || result.Status != "no_matches" || len(result.Sources) != 0 || s.calls != 1 || c.calls != 0 || result.Usage.Complete {
		t.Fatalf("无结果不得聊天: %s %v", out, err)
	}
}
func TestKbAnalyzeSingleSearchChatAndUsage(t *testing.T) {
	for _, known := range []bool{true, false} {
		t.Run(map[bool]string{true: "known", false: "missing"}[known], func(t *testing.T) {
			s, c := analysisFixture()
			if !known {
				c.usage = nil
			}
			owned := 0
			tool := NewKbAnalyze(s, func(context.Context, int64, int64) error { owned++; return nil }, c, 32000)
			out, err := tool.Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"分析预算"}`))
			var result KbAnalysisResult
			if err != nil || json.Unmarshal([]byte(out), &result) != nil {
				t.Fatalf("%v", err)
			}
			if owned != 1 || s.calls != 1 || c.calls != 1 || result.Status != "completed" || result.Version != 1 || result.Type != "kb_analysis" || len(result.Sources) != 1 || result.Sources[0].Index != 1 || result.Usage.Scope != "chat" || result.Usage.Complete != known {
				t.Fatalf("调用与结果不符: %s", out)
			}
			if known && result.Usage.TotalTokens != 120 {
				t.Fatal("用量须来自供应商")
			}
			if c.maxTokens != 2048 {
				t.Fatal("回答须限量")
			}
			if !strings.Contains(c.messages[0].Content, "不可信") || !strings.Contains(c.messages[0].Content, "资料不足") || !strings.Contains(c.messages[1].Content, result.Sources[0].Snippet) {
				t.Fatal("提示词与来源不符")
			}
		})
	}
}
func TestKbAnalyzeBudgetPreservesQueryAndSources(t *testing.T) {
	s, c := analysisFixture()
	s.hits[0].Snippet = strings.Repeat("中文\"\\\n", 6000)
	s.hits[0].DocTitle = strings.Repeat("题", 4000)
	query := strings.Repeat("目标", 200)
	tool := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 1600)
	args, _ := json.Marshal(map[string]any{"kb_id": 3, "query": query})
	out, err := tool.Execute(WithActor(context.Background(), 7), args)
	if err != nil {
		t.Fatal(err)
	}
	var result KbAnalysisResult
	if json.Unmarshal([]byte(out), &result) != nil || !utf8.ValidString(out) || len(out) > 24000 || strings.Contains(out, "\n") {
		t.Fatal("结果必须为有界单行 UTF8 JSON")
	}
	encoded, _ := json.Marshal(c.messages)
	if (len(encoded)+2)/3+c.maxTokens > 1600 {
		t.Fatal("输入须预留回答预算")
	}
	if !strings.Contains(c.messages[1].Content, query) {
		t.Fatal("不能截断目标")
	}
	if len(result.Sources) != 1 || !strings.Contains(c.messages[1].Content, `"snippet":`) {
		t.Fatal("来源丢失")
	}
	var input struct {
		Sources []KbAnalysisSource `json:"sources"`
	}
	if json.Unmarshal([]byte(c.messages[1].Content), &input) != nil || input.Sources[0] != result.Sources[0] {
		t.Fatal("保存的来源必须与实际送模型完全一致")
	}
}

func TestKbAnalyzeAcceptsSingleChineseRuneAndRejectsInsufficientBudget(t *testing.T) {
	s, c := analysisFixture()
	s.hits[0].Snippet = "中"
	if _, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`)); err != nil {
		t.Fatalf("单字资料仍须分析: %v", err)
	}
	s, c = analysisFixture()
	if _, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 10).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`)); err == nil || c.calls != 0 {
		t.Fatal("预算不足必须在聊天前失败")
	}
}
func TestKbAnalyzeFailsSafelyAndMarksLength(t *testing.T) {
	for _, kind := range []string{"search", "chat", "empty", "invalid_utf8", "tool_calls", "oversize", "filter", "usage"} {
		t.Run(kind, func(t *testing.T) {
			s, c := analysisFixture()
			switch kind {
			case "search":
				s.err = errors.New("vendor private")
			case "chat":
				c.err = errors.New("vendor private")
			case "empty":
				c.response.Content = " "
			case "invalid_utf8":
				c.response.Content = string([]byte{0xff})
			case "tool_calls":
				c.response.ToolCalls = []schema.ToolCall{{ID: "fake"}}
			case "oversize":
				c.response.Content = strings.Repeat("x", 13000)
			case "filter":
				c.response.ResponseMeta = &schema.ResponseMeta{FinishReason: "content_filter"}
			case "usage":
				c.usage.TotalTokens = -1
			}
			out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`))
			if err == nil || out != "" || strings.Contains(err.Error(), "private") {
				t.Fatalf("失败不能伪装成功或泄漏: %q %v", out, err)
			}
		})
	}
	s, c := analysisFixture()
	c.response.ResponseMeta = &schema.ResponseMeta{FinishReason: "length"}
	out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`))
	var result KbAnalysisResult
	if err != nil || json.Unmarshal([]byte(out), &result) != nil || result.Status != "truncated" {
		t.Fatal("达到模型长度上限须明确标记")
	}
}
func TestWorkflowOnlyToolRemainsListedButNotOffered(t *testing.T) {
	s, c := analysisFixture()
	tool := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000)
	previous, had := registry[tool.Name()]
	registry[tool.Name()] = tool
	t.Cleanup(func() {
		if had {
			registry[tool.Name()] = previous
		} else {
			delete(registry, tool.Name())
		}
	})
	if _, ok := Get(tool.Name()); !ok {
		t.Fatal("工作流须能取工具")
	}
	found := false
	for _, v := range List() {
		if v.Name() == tool.Name() {
			found = true
		}
	}
	if !found {
		t.Fatal("目录须能取工具")
	}
	for _, v := range LLMTools() {
		if v.Name == tool.Name() {
			t.Fatal("工作流专用工具不能提供给聊天模型")
		}
	}
	if !IsWorkflowOnly(tool) {
		t.Fatal("工作流标记缺失")
	}
}
