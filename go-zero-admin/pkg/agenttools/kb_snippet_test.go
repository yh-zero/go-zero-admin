package agenttools

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestKbAnalyzeBudgetKeepsRelevantTailAndItsConditions(t *testing.T) {
	const evidence = "只有审批通过，才允许账单导出；审批未通过时，不得账单导出。"
	hits := analysisCandidateHits(20)
	for i := range hits {
		hits[i].Snippet = strings.Repeat("这段介绍页面布局和常规使用方式。", 36) + evidence
	}
	tool := &kbAnalyzeTool{maxContextTokens: 32000}
	messages, sources, err := tool.fitSources("账单导出有什么条件？", 3, hits, 2048)
	if err != nil || len(sources) != 20 {
		t.Fatalf("候选与原问题应在预算内保留: count=%d err=%v", len(sources), err)
	}
	for _, source := range sources {
		if !strings.Contains(source.Snippet, evidence) || !strings.Contains(hits[source.Index-1].Snippet, source.Snippet) {
			t.Fatalf("预算压缩丢失尾部事实、否定或条件: %q", source.Snippet)
		}
		if source.ChunkID != hits[source.Index-1].ChunkID || source.Score != hits[source.Index-1].Score {
			t.Fatal("片段选择改变了来源身份或分数")
		}
	}
	encoded, _ := json.Marshal(messages)
	if (len(encoded)+2)/3+2048 > 32000 || !strings.Contains(messages[1].Content, "账单导出有什么条件？") || analysisLargestSourceSetBytes(sources) > kbAnalysisSourceBytes {
		t.Fatal("相关片段不能挤占完整问题、回答及来源预算")
	}
	t.Logf("同语料、同问题、同字节额度：旧前缀含完整条件=%t，相关窗口=%t，候选=%d，单片段=%d字节，模型输入=%d字节", strings.Contains(truncateUTF8(hits[0].Snippet, len(sources[0].Snippet)), evidence), strings.Contains(sources[0].Snippet, evidence), len(sources), len(sources[0].Snippet), len(encoded))
}

func TestKbAnalyzeRelevantWindowsRespectEscapedJSONAndSmallBudget(t *testing.T) {
	hits := analysisCandidateHits(20)
	for i := range hits {
		hits[i].Snippet = strings.Repeat("无关前言。", 90) + "若 AI.MaxContextTokens 未配置，不应据此关闭服务。" + strings.Repeat("\"\\\n", 250)
	}
	tool := &kbAnalyzeTool{maxContextTokens: 2300}
	messages, sources, err := tool.fitSources("AI.MaxContextTokens 未配置怎么办？", 3, hits, 500)
	if err != nil || len(sources) == 0 || len(sources) >= 20 {
		t.Fatalf("小预算应保留完整问题并减少候选: count=%d err=%v", len(sources), err)
	}
	encoded, _ := json.Marshal(messages)
	if (len(encoded)+2)/3+500 > 2300 || !strings.Contains(messages[1].Content, "AI.MaxContextTokens 未配置怎么办？") {
		t.Fatal("转义后的完整请求超过预算或问题被截断")
	}
	for _, source := range sources {
		if !utf8.ValidString(source.Snippet) || !strings.Contains(hits[source.Index-1].Snippet, source.Snippet) {
			t.Fatal("小预算片段必须仍为连续UTF8原文")
		}
	}
	tool.maxContextTokens = 100
	if _, _, err := tool.fitSources("完整问题不得被裁掉", 3, hits, 25); err == nil {
		t.Fatal("必要输入都放不下时应失败，不能抹掉问题")
	}
}

func TestKbAnalyzeRelevantWindowDoesNotMistakeWhitespaceForBudgetFailure(t *testing.T) {
	hits := analysisCandidateHits(1)
	hits[0].Snippet = "alpha" + strings.Repeat(" ", 160) + "beta"
	tool := &kbAnalyzeTool{maxContextTokens: 32000}
	_, sources, err := tool.fitSources("alpha beta", 3, hits, 2048)
	if err != nil || len(sources) != 1 || sources[0].Snippet != hits[0].Snippet {
		t.Fatalf("小窗口中的空白不能令二分错过放得下的真实来源: %v", err)
	}
}

func TestRelevantKbSnippetKeepsLocalEvidenceWithoutInventingText(t *testing.T) {
	cases := []struct {
		name, query, evidence string
	}{
		{"中文", "账单导出有什么条件？", "只有审批通过，才允许账单导出；未审批时不得账单导出。"},
		{"英文", "When is billing export allowed?", "Billing export is allowed only after approval; without approval it must not run."},
		{"配置标识符", "AI.MaxContextTokens 未配置怎么办？", "若 AI.MaxContextTokens 未配置，沿用默认预算，不关闭服务。"},
		{"路径标识符", "internal/config/runtime.yaml controls which limit?", "Only internal/config/runtime.yaml controls this limit; a sample file does not."},
		{"单字", "税", "税仅按本段注明的条件计算，不得套用其他情况。"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content := strings.Repeat("前言与无关说明。", 50) + tc.evidence + strings.Repeat("相邻补充说明。", 15)
			got := RelevantKbSnippet(content, tc.query, 420)
			if !strings.Contains(got, tc.evidence) || !strings.Contains(content, got) || !utf8.ValidString(got) || len(got) > 420 {
				t.Fatalf("须保留真实连续的局部事实及条件: %q", got)
			}
		})
	}
}

func TestRelevantKbSnippetNoMatchFallsBackToPrefixAndHonorsLimits(t *testing.T) {
	content := "普通说明😀与中文。" + strings.Repeat("资料", 30)
	for _, query := range []string{"完全无关", "unrelated_identifier", ""} {
		for limit := -1; limit <= len(content)+1; limit++ {
			got := RelevantKbSnippet(content, query, limit)
			if limit <= 0 {
				if got != "" {
					t.Fatal("非正预算应返回空片段")
				}
				continue
			}
			if !strings.HasPrefix(content, got) || !utf8.ValidString(got) || len(got) > limit {
				t.Fatalf("无匹配必须保持UTF8原前缀: limit=%d got=%q", limit, got)
			}
		}
	}
	if got := RelevantKbSnippet(content, "普通说明", len(content)); got != content {
		t.Fatal("全文放得下时不能移动窗口或改变正文")
	}
}

func TestRelevantKbSnippetPrefersDistinctLocalTermsAndWholeEnglishWords(t *testing.T) {
	content := strings.Repeat("The limit is unlimited; this is only a generic overview. ", 30) + "A request quota can change only after approval."
	got := RelevantKbSnippet(content, "request quota approval", 180)
	if !strings.Contains(got, "A request quota can change only after approval.") {
		t.Fatalf("重复常见词不应挤掉多个实际问词的局部证据: %q", got)
	}
	content = strings.Repeat("unlimited ", 70) + "trailing note"
	if got := RelevantKbSnippet(content, "limit", 90); got != content[:90] {
		t.Fatal("英文子串误匹配不应移动无关资料窗口")
	}
}

func TestRelevantKbSnippetWindowsRemainNestedForSerializedBudgetSearch(t *testing.T) {
	content := strings.Repeat("前言\"\\\n😀。", 80) + "若 ai.maxcontexttokens 不存在，则保留默认值。" + strings.Repeat("尾注\"\\\n。", 80)
	query := "AI.MaxContextTokens"
	previous := ""
	previousJSONBytes := 0
	for limit := 1; limit <= len(content); limit++ {
		got := RelevantKbSnippet(content, query, limit)
		encoded, _ := json.Marshal(got)
		if !utf8.ValidString(got) || len(got) > limit || !strings.Contains(content, got) || !strings.Contains(got, previous) || len(encoded) < previousJSONBytes {
			t.Fatalf("窗口扩大须嵌套且序列化大小单调: limit=%d", limit)
		}
		previous, previousJSONBytes = got, len(encoded)
	}
}

func TestRelevantKbSnippetSmallWindowsKeepLongSentenceEvidence(t *testing.T) {
	const evidence = "Billing export requires approval"
	for _, tc := range []struct {
		name, sentence string
	}{
		{"sentence_start", evidence + " and " + strings.Repeat("ordinary narrative ", 19) + "."},
		{"sentence_end", strings.Repeat("ordinary narrative ", 19) + evidence + "."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			content := tc.sentence + " " + strings.Repeat("other material. ", 50)
			for _, limit := range []int{60, 100, 150, 240} {
				got := RelevantKbSnippet(content, "billing export approval", limit)
				if !strings.Contains(got, evidence) || !strings.Contains(content, got) || len(got) > limit {
					t.Fatalf("长句小窗口不能移动到无关句中而丢失完整条件: limit=%d snippet=%q", limit, got)
				}
			}
		})
	}
}

func TestRelevantKbSnippetSmallWindowAnchorsOnAMatchedTerm(t *testing.T) {
	content := "quota " + strings.Repeat(" ", 150) + "approval requires review. " + strings.Repeat("other material. ", 50)
	got := RelevantKbSnippet(content, "quota approval", 60)
	if !strings.Contains(got, "approval") && !strings.Contains(got, "quota") {
		t.Fatalf("命中词相距较远时，小窗口不能落在两词间空白: %q", got)
	}
}

func BenchmarkRelevantKbSnippet(b *testing.B) {
	content := strings.Repeat("前言与常规说明。", 35) + "若 AI.MaxContextTokens 未配置，保留默认预算，不关闭服务。"
	for b.Loop() {
		RelevantKbSnippet(content, "AI.MaxContextTokens 未配置如何处理？", 600)
	}
}
