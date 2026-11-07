package agenttools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func runAnalysisReply(t *testing.T, query, reply, finish string) (KbAnalysisResult, error, *analysisSearcher, *analysisChat) {
	t.Helper()
	s, c := analysisFixture()
	c.response.Content = reply
	c.response.ResponseMeta = &schema.ResponseMeta{FinishReason: finish}
	args, err := json.Marshal(map[string]any{"kb_id": 3, "query": query})
	if err != nil {
		t.Fatal(err)
	}
	out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), args)
	var result KbAnalysisResult
	if err == nil && json.Unmarshal([]byte(out), &result) != nil {
		t.Fatal("外部结果必须继续是合法JSON")
	}
	if err != nil && out != "" {
		t.Fatal("不合法模型回复不得返回部分成功结果")
	}
	return result, err, s, c
}

func TestKbAnalysisRendersSupportedAndScopedMissingTargets(t *testing.T) {
	for _, tc := range []struct{ name, query, reply, want string }{
		{"有据", "预算是多少", `{"answers":[{"target":"预算","state":"supported","text":"预算为五万元。","citations":[1]}]}`, "预算：预算为五万元。 [1]"},
		{"单中文query", "税", `{"answers":[{"target":"税","state":"supported","text":"税款卖出时扣缴。","citations":[1]}]}`, "税：税款卖出时扣缴。 [1]"},
		{"单ASCIIquery", "a", `{"answers":[{"target":"a","state":"supported","text":"值为一。","citations":[1]}]}`, "a：值为一。 [1]"},
		{"部分缺失", "预算和保存天数", `{"answers":[{"target":"预算","state":"supported","text":"预算为五万元。","citations":[1]},{"target":"保存天数","state":"insufficient","text":"","citations":[]}]}`, "预算：预算为五万元。 [1]\n\n本次片段未能确定“保存天数”。"},
		{"真正无答案不回显标记", "确认UNKNOWN_MARKER口令", `{"answers":[{"target":"UNKNOWN_MARKER口令","state":"insufficient","text":"","citations":[]}]}`, "本次片段资料不足，无法回答该分析目标。"},
		{"部分缺失转义目标", "预算和[x](https://example.test)", `{"answers":[{"target":"预算","state":"supported","text":"预算为五万元。","citations":[1]},{"target":"[x](https://example.test)","state":"insufficient","text":"","citations":[]}]}`, "预算：预算为五万元。 [1]\n\n本次片段未能确定“\\[x\\]\\(https://example\\.test\\)”。"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err, s, c := runAnalysisReply(t, tc.query, tc.reply, "stop")
			if err != nil || got.Analysis != tc.want || got.Status != "completed" || len(got.Sources) != 1 || got.Usage.TotalTokens != 120 || !got.Usage.Complete || s.calls != 1 || c.calls != 1 {
				t.Fatalf("渲染或调用不符: result=%+v err=%v search=%d chat=%d", got, err, s.calls, c.calls)
			}
		})
	}
}

func TestKbAnalysisKeepsQuestionLabelsAndEscapesTheirMarkdown(t *testing.T) {
	for _, tc := range []struct{ name, query, reply, want string }{
		{"集合竞价和撤单时段保持对应", "早盘集合竞价从几点到几点，哪个时段不能撤单", `{"answers":[{"target":"早盘集合竞价从几点到几点","state":"supported","text":"9:15-9:25","citations":[1]},{"target":"哪个时段不能撤单","state":"supported","text":"9:20-9:25","citations":[1]}]}`, "早盘集合竞价从几点到几点：9:15-9:25 [1]\n\n哪个时段不能撤单：9:20-9:25 [1]"},
		{"标签中的代码链接换行作为文字", "代码`a`\n[x](https://example.test)", "{\"answers\":[{\"target\":\"代码`a`\\n[x](https://example.test)\",\"state\":\"supported\",\"text\":\"值\",\"citations\":[1]}]}", "代码\\`a\\` \\[x\\]\\(https://example\\.test\\)：值 [1]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err, s, c := runAnalysisReply(t, tc.query, tc.reply, "stop")
			if err != nil || got.Analysis != tc.want || s.calls != 1 || c.calls != 1 {
				t.Fatalf("正文必须保留问项标签及安全转义：analysis=%q err=%v", got.Analysis, err)
			}
		})
	}
}

func TestKbAnalysisCountsEscapedLabelsInOutputBudget(t *testing.T) {
	query := strings.Repeat(".!#_-+", 333) + "."
	var items []string
	for i := 0; i < 3; i++ {
		target, _ := json.Marshal(query[i : i+1900])
		items = append(items, fmt.Sprintf(`{"target":%s,"state":"supported","text":"%s","citations":[1]}`, target, strings.Repeat("a", 1900)))
	}
	_, err, _, _ := runAnalysisReply(t, query, `{"answers":[`+strings.Join(items, ",")+`]}`, "stop")
	if err == nil {
		t.Fatal("加入转义目标标签后超过正文预算必须拒绝，不能只计算答案text")
	}
}

func TestKbAnalysisRejectsUnstructuredAndInvalidAnswersWithoutRetry(t *testing.T) {
	valid := `{"answers":[{"target":"预算","state":"supported","text":"预算为五万元。","citations":[1]}]}`
	cases := []struct{ name, query, reply string }{
		{"旧自由正文", "预算", "预算为五万元。[1]"},
		{"旧备份自相矛盾正文", "运行数据副本及哪两项导出记录", "记录导出时间与校验值[1]。资料中未提及两项导出记录的具体名称，无法确定第二项。"},
		{"旧二线泛化尾注", "二线是什么角色", "二线为模块负责人[1]。本次片段未涉及其他升级时限或二线角色更多说明。"},
		{"旧临停泛化尾注", "临停多久", "单次停牌10分钟[1]。资料未提供其他触发比例或停牌时长。"},
		{"自由前文", "预算", "结果：" + valid},
		{"自由尾注", "预算", valid + "\n资料未提供其他信息。"},
		{"额外JSON", "预算", valid + `{}`},
		{"顶层未知字段", "预算", `{"answers":[{"target":"预算","state":"supported","text":"预算五万元","citations":[1]}],"note":"额外"}`},
		{"问项未知字段", "预算", `{"answers":[{"target":"预算","state":"supported","text":"预算五万元","citations":[1],"note":"额外"}]}`},
		{"字段大小写替代", "预算", `{"answers":[{"Target":"预算","state":"supported","text":"预算五万元","citations":[1]}]}`},
		{"重复根键", "预算", `{"answers":[],"answers":[{"target":"预算","state":"supported","text":"预算五万元","citations":[1]}]}`},
		{"重复问项键", "预算", `{"answers":[{"target":"预算","state":"supported","text":"先写","text":"后写","citations":[1]}]}`},
		{"转义重复键", "预算", `{"answers":[{"target":"预算","state":"supported","text":"先写","\u0074ext":"后写","citations":[1]}]}`},
		{"未询问目标", "预算", `{"answers":[{"target":"其他升级时限","state":"insufficient","text":"","citations":[]}]}`},
		{"目标为空", "预算", `{"answers":[{"target":"","state":"supported","text":"五万元","citations":[1]}]}`},
		{"目标前后空白", " 预算 ", `{"answers":[{"target":" 预算 ","state":"supported","text":"五万元","citations":[1]}]}`},
		{"目标重复且状态冲突", "预算", `{"answers":[{"target":"预算","state":"supported","text":"五万元","citations":[1]},{"target":"预算","state":"insufficient","text":"","citations":[]}]}`},
		{"目标互含", "月预算和预算", `{"answers":[{"target":"月预算","state":"supported","text":"五万元","citations":[1]},{"target":"预算","state":"insufficient","text":"","citations":[]}]}`},
		{"未知状态", "预算", `{"answers":[{"target":"预算","state":"unknown","text":"","citations":[]}]}`},
		{"有据项空正文", "预算", `{"answers":[{"target":"预算","state":"supported","text":" ","citations":[1]}]}`},
		{"有据项无引用", "预算", `{"answers":[{"target":"预算","state":"supported","text":"五万元","citations":[]}]}`},
		{"有据项引用越界", "预算", `{"answers":[{"target":"预算","state":"supported","text":"五万元","citations":[2]}]}`},
		{"有据项零引用", "预算", `{"answers":[{"target":"预算","state":"supported","text":"五万元","citations":[0]}]}`},
		{"有据项重复引用", "预算", `{"answers":[{"target":"预算","state":"supported","text":"五万元","citations":[1,1]}]}`},
		{"有据项引用非整数", "预算", `{"answers":[{"target":"预算","state":"supported","text":"五万元","citations":[1.5]}]}`},
		{"正文自行引用", "预算", `{"answers":[{"target":"预算","state":"supported","text":"五万元[1]","citations":[1]}]}`},
		{"正文混合编号", "预算", `{"answers":[{"target":"预算","state":"supported","text":"五万元[1,2]","citations":[1]}]}`},
		{"正文全角编号", "预算", `{"answers":[{"target":"预算","state":"supported","text":"五万元【１】","citations":[1]}]}`},
		{"不足项自写正文", "预算", `{"answers":[{"target":"预算","state":"insufficient","text":"不知道","citations":[]}]}`},
		{"不足项有引用", "预算", `{"answers":[{"target":"预算","state":"insufficient","text":"","citations":[1]}]}`},
		{"不足项缺text", "预算", `{"answers":[{"target":"预算","state":"insufficient","citations":[]}]}`},
		{"不足项null", "预算", `{"answers":[{"target":"预算","state":"insufficient","text":null,"citations":[]}]}`},
		{"不足项null引用", "预算", `{"answers":[{"target":"预算","state":"insufficient","text":"","citations":null}]}`},
		{"空问项", "预算", `{"answers":[]}`},
		{"无问项", "预算", `{}`},
		{"null问项", "预算", `{"answers":null}`},
		{"非法UTF8", "预算", `{"answers":[{"target":"预算","state":"supported","text":"` + string([]byte{0xff}) + `","citations":[1]}]}`},
		{"单项超过2000字节", "预算", `{"answers":[{"target":"预算","state":"supported","text":"` + strings.Repeat("字", 667) + `","citations":[1]}]}`},
	}
	for _, phrase := range []string{"未提供", "未涉及", "未说明", "未记载", "未提及", "未包含", "未给出", "无法确定", "无法确认", "资料不足"} {
		cases = append(cases, struct{ name, query, reply string }{"有据正文包含不足状态_" + phrase, "预算", `{"answers":[{"target":"预算","state":"supported","text":"预算五万元，但` + phrase + `其他说明","citations":[1]}]}`})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err, s, c := runAnalysisReply(t, tc.query, tc.reply, "stop")
			if err == nil || s.calls != 1 || c.calls != 1 {
				t.Fatalf("坏答案须拒绝且不能重试: err=%v search=%d chat=%d", err, s.calls, c.calls)
			}
		})
	}
}

func TestKbAnalysisPreservesSourceRulesWithoutAcceptingUncitedAbsenceTail(t *testing.T) {
	rule := "资料未提供审批人时，应说明资料不足，不得编造。"
	for _, tc := range []struct {
		name, text string
		citations  []int
		wantError  bool
	}{
		{"实际引用原文规则", rule, []int{2}, false},
		{"未引用的原文不能豁免", rule, []int{1}, true},
		{"相同措辞的无依据尾注", "预算五万元。资料未提供其他说明。", []int{1, 2}, true},
		{"正常结果未知规则", "结果未知时不能自动重放。", []int{1}, false},
		{"正常未知字段规则", "未知字段需报错。", []int{1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, c := analysisFixture()
			s.hits = append(s.hits, KbSearchHit{ChunkID: 2, KbID: 3, DocID: 3, DocTitle: "规则", Snippet: rule, Score: .8})
			reply, _ := json.Marshal(map[string]any{"answers": []any{map[string]any{"target": "目标", "state": "supported", "text": tc.text, "citations": tc.citations}}})
			c.response.Content = string(reply)
			out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`))
			if (err != nil) != tc.wantError || c.calls != 1 {
				t.Fatalf("仅实际引用片段的完整逐字摘录可豁免不足措辞: err=%v", err)
			}
			if !tc.wantError {
				var result KbAnalysisResult
				if json.Unmarshal([]byte(out), &result) != nil || !strings.HasPrefix(result.Analysis, "目标："+tc.text) {
					t.Fatal("不能通过删词或改写原文规则掩盖校验结果")
				}
			}
		})
	}
}

func TestKbAnalysisKeepsLengthResultSafeAndBounded(t *testing.T) {
	valid := `{"answers":[{"target":"预算","state":"supported","text":"五万元","citations":[1]}]}`
	for _, tc := range []struct{ name, reply, want string }{
		{"完整JSON", valid, "预算：五万元 [1]"},
		{"中途截断JSON", `{"answers":[{"target":"预算","state":"supported","text":"private-partial`, "分析达到输出上限，结构未完整返回，无法展示分析正文。请缩小分析范围后重试。"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err, s, c := runAnalysisReply(t, "预算", tc.reply, "length")
			if err != nil || got.Status != "truncated" || got.Analysis != tc.want || strings.Contains(got.Analysis, "private-partial") || len(got.Sources) != 1 || got.Usage.TotalTokens != 120 || !got.Usage.Complete || s.calls != 1 || c.calls != 1 {
				t.Fatalf("截断结果须保留来源和已知用量且不展示坏JSON: %+v %v", got, err)
			}
		})
	}
	_, err, _, _ := runAnalysisReply(t, "预算", strings.Replace(valid, `"citations":[1]`, `"citations":[99]`, 1), "length")
	if err == nil {
		t.Fatal("完整但非法的JSON不能因length标记绕过校验")
	}
}

func TestKbAnalysisLimitsOutputItemsWithoutLimitingQueryTargets(t *testing.T) {
	for _, n := range []int{16, 17} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			var targets, items []string
			for i := 0; i < n; i++ {
				target := fmt.Sprintf("项目%02d", i)
				targets = append(targets, target)
				items = append(items, fmt.Sprintf(`{"target":%q,"state":"supported","text":"值","citations":[1]}`, target))
			}
			_, err, s, c := runAnalysisReply(t, strings.Join(targets, "、"), `{"answers":[`+strings.Join(items, ",")+`]}`, "stop")
			if (err != nil) != (n == 17) || s.calls != 1 || c.calls != 1 {
				t.Fatalf("16项为回复约束，不能提前拒绝query: n=%d err=%v search=%d chat=%d", n, err, s.calls, c.calls)
			}
		})
	}
}

func TestKbAnalysisDoesNotClaimCrossTargetSemanticValidation(t *testing.T) {
	// 两个字面不同目标仍可能语义重叠；结构校验不冒充完整语义检验。
	reply := `{"answers":[{"target":"运行数据副本","state":"supported","text":"导出快照，并记录导出时间与校验值。","citations":[1]},{"target":"哪两项导出记录","state":"insufficient","text":"","citations":[]}]}`
	got, err, _, _ := runAnalysisReply(t, "运行数据副本及哪两项导出记录", reply, "stop")
	if err != nil || !strings.Contains(got.Analysis, "导出时间与校验值") || !strings.Contains(got.Analysis, "未能确定") {
		t.Fatal("该反例须保留为结构与语义校验的边界，而不是误称已能检测")
	}
}
