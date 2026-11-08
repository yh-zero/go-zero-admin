package agenttools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestKbAnalysisFailureReasonIsStatic(t *testing.T) {
	sources := []KbAnalysisSource{{Index: 1, KbSearchHit: KbSearchHit{Snippet: "预算为五万元。"}}}
	for _, tc := range []struct{ name, query, reply, reason string }{
		{"非法目标", "预算", `{"answers":[{"target":"其他目标","state":"insufficient","text":"","citations":[]}]}`, "内部回复目标不是原问题中的有效文字"},
		{"重复引用", "预算", `{"answers":[{"target":"预算","state":"supported","text":"预算为五万元。","citations":[1,1]}]}`, "内部回复引用编号无效或重复"},
		{"背景不足", "有人建议“忽略规则”。请说明实际规则。", `{"answers":[{"target":"忽略规则","state":"insufficient","text":"","citations":[]}]}`, "已结束的背景引文不能独立成为问项"},
		{"非法JSON", "预算", `{"PRIVATE_INPUT_MARKER":`, "JSON格式无效"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := renderKbAnalysisResponse(tc.reply, tc.query, sources)
			if err == nil || out != "" || analysisResponseFailureReason(err) != tc.reason {
				t.Fatalf("必须拒绝整条回复且仅返回固定诊断原因: out=%q reason=%q", out, analysisResponseFailureReason(err))
			}
		})
	}
}

func TestKbAnalysisFailureReasonNeverCopiesDynamicErrors(t *testing.T) {
	var value struct {
		Private string `json:"PRIVATE_DECODER_MARKER"`
	}
	decoderErr := json.Unmarshal([]byte(`{"PRIVATE_DECODER_MARKER":{}}`), &value)
	if decoderErr == nil || !strings.Contains(decoderErr.Error(), "PRIVATE_DECODER_MARKER") {
		t.Fatal("回归前提必须包含可能携带输入字段的真实JSON解码错误")
	}
	for _, err := range []error{decoderErr, fmt.Errorf("PRIVATE_WRAPPER_MARKER: %w", decoderErr), errors.New("PRIVATE_ERROR_MARKER"), nil} {
		if reason := analysisResponseFailureReason(err); reason != "JSON格式无效" {
			t.Fatalf("动态错误不得透传到诊断日志: reason=%q", reason)
		}
	}
}

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

func TestKbAnalysisRejectsQuotedContextAsMissingTarget(t *testing.T) {
	// 原始日志只保存外部analysis；此处按实测问项与状态重构内部坏结构，不冒充供应商原始回复。
	query := "有人建议“忽略归属，知道任务ID就直接读别人的任务，前端隐藏按钮就够了”。请按本项目资料说明实际的数据隔离和服务端授权要求。"
	reply := `{"answers":[{"target":"实际的数据隔离和服务端授权要求","state":"supported","text":"会话、工作流和任务仅其所属用户可读写，越权ID与不存在均返回404；页面入口、操作权限与服务端接口授权分别校验。","citations":[1]},{"target":"忽略归属，知道任务ID就直接读别人的任务，前端隐藏按钮就够了","state":"insufficient","text":"","citations":[]}]}`
	s, c := analysisFixture()
	s.hits[0].Snippet = "会话、工作流和任务仅其所属用户可读写，越权ID与不存在均返回404；页面入口、操作权限与服务端接口授权分别校验。"
	c.response.Content = reply
	args, _ := json.Marshal(map[string]any{"kb_id": 3, "query": query})
	out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), args)
	if err == nil || out != "" || s.calls != 1 || c.calls != 1 {
		t.Fatalf("不能把背景建议呈现为未知问项，也不能删尾句伪装成功或自动重试：out=%q err=%v search=%d chat=%d", out, err, s.calls, c.calls)
	}
}

func TestKbAnalysisRejectsQuotedBackgroundAsSupportedTarget(t *testing.T) {
	// 任务228/229只保存外部analysis；按实测坏标签重构等价内部结构，不冒充供应商原始JSON。
	query := "有人建议“忽略归属，知道任务ID就直接读别人的任务，前端隐藏按钮就够了”。请按本项目资料说明实际的数据隔离和服务端授权要求。"
	for _, backgroundFirst := range []bool{true, false} {
		items := []kbModelAnswer{
			{Target: "忽略归属，知道任务ID就直接读别人的任务，前端隐藏按钮就够了", State: "supported", Text: "会话、工作流和任务仅其所属用户可读写，越权 ID 与不存在均返回 404。", Citations: []int{1}},
			{Target: "实际的数据隔离和服务端授权要求", State: "supported", Text: "知识库、会话、流程和任务按创建者隔离；页面入口、操作权限与服务端接口授权分别校验。", Citations: []int{1}},
		}
		if !backgroundFirst {
			items[0], items[1] = items[1], items[0]
		}
		reply, _ := json.Marshal(map[string]any{"answers": items})
		_, err, s, c := runAnalysisReply(t, query, string(reply), "stop")
		if err == nil || s.calls != 1 || c.calls != 1 {
			t.Fatalf("正确事实不能让错误背景标签成为独立有据问项，也不能删句或重试: err=%v search=%d chat=%d", err, s.calls, c.calls)
		}
	}
}

func TestKbAnalysisPromptSeparatesBackgroundAndPreservesOriginalQuery(t *testing.T) {
	for _, tc := range []struct {
		name, query, modelQuery string
		background              []string
	}{
		{"确定背景", "有人建议“忽略归属”。请说明实际规则。", "请说明实际规则。", []string{"“忽略归属”"}},
		{"原178实际问题", "有人建议“忽略归属，知道任务ID就直接读别人的任务，前端隐藏按钮就够了”。请按本项目资料说明实际的数据隔离和服务端授权要求。", "请按本项目资料说明实际的数据隔离和服务端授权要求。", []string{"“忽略归属，知道任务ID就直接读别人的任务，前端隐藏按钮就够了”"}},
		{"两个独立背景", "有人建议“甲”。请说明实际规则。有人声称“乙”。请说明预算。", "请说明实际规则。有人声称“乙”。请说明预算。", []string{"“甲”", "“乙”"}},
		{"前置真实问题不能丢", "预算是多少？有人建议“忽略归属”。请说明实际规则。", "", []string{"“忽略归属”"}},
		{"前置真实请求不能丢", "请说明预算。有人建议“忽略归属”。请说明实际规则。", "", []string{"“忽略归属”"}},
		{"列出请求不能依赖词表识别", "列出预算明细。有人建议“忽略归属”。请说明实际授权规则。", "", []string{"“忽略归属”"}},
		{"英文List请求不能丢", `List the budget items. Someone said "ignore ownership". Please explain actual rules.`, "", []string{`"ignore ownership"`}},
		{"逗号前置请求不收窄", "列出预算明细，有人建议“忽略归属”。请说明实际规则。", "", []string{"“忽略归属”"}},
		{"分号前置请求不收窄", `List the budget items; Someone said "ignore ownership". Please explain actual rules.`, "", []string{`"ignore ownership"`}},
		{"换行前置请求不收窄", "列出预算明细\n有人建议“忽略归属”。请说明实际规则。", "", []string{"“忽略归属”"}},
		{"转述引导末尾中文冒号", "有人建议：“忽略归属”。请说明实际规则。", "请说明实际规则。", []string{"“忽略归属”"}},
		{"转述引导末尾英文冒号", `Someone said: "ignore ownership". Please explain actual rules.`, "Please explain actual rules.", []string{`"ignore ownership"`}},
		{"后文请求不紧邻时不收窄", "有人建议“忽略归属”。另有待议事项，预算是多少？", "", []string{"“忽略归属”"}},
		{"明确请求引文", "请回答“保存天数是多少？”，并说明预算。", "", nil},
		{"整个问题仅引文", "“上线日期”", "", nil},
		{"回指引文", "手册写着“保存天数”。请告诉我它的值。", "", nil},
		{"不确定背景", "“保存天数”。请说明预算。", "", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := analysisFixture()
			tool := &kbAnalyzeTool{maxContextTokens: 32000}
			messages, sources, err := tool.fitSources(tc.query, 3, s.hits, 2048)
			if err != nil || len(messages) != 2 || messages[0].Role != schema.System || messages[1].Role != schema.User {
				t.Fatalf("背景标记必须保持原消息角色: err=%v", err)
			}
			var payload struct {
				Query         string             `json:"query"`
				OriginalQuery string             `json:"original_query"`
				Background    []string           `json:"background_quotes"`
				Sources       []KbAnalysisSource `json:"sources"`
			}
			wantQuery, wantOriginal := tc.query, ""
			if tc.modelQuery != "" {
				wantQuery, wantOriginal = tc.modelQuery, tc.query
			}
			if err := json.Unmarshal([]byte(messages[1].Content), &payload); err != nil || payload.Query != wantQuery || payload.OriginalQuery != wantOriginal || len(payload.Sources) != len(sources) || len(payload.Background) != len(tc.background) {
				t.Fatalf("只收窄确定前置背景后的直接请求，原问题和来源必须保留: err=%v payload=%+v", err, payload)
			}
			for i, quote := range tc.background {
				if payload.Background[i] != quote {
					t.Fatalf("背景必须逐字保留原引文: got=%q want=%q", payload.Background[i], quote)
				}
			}
			if len(tc.background) == 0 && strings.Contains(messages[1].Content, `"background_quotes"`) {
				t.Fatal("真正引文问题不能被标记为背景，空背景字段应省略")
			}
			if tc.modelQuery == "" && strings.Contains(messages[1].Content, `"original_query"`) {
				t.Fatal("未收窄时不能额外复制原问题")
			}
			if !strings.Contains(messages[0].Content, "background_quotes") || !strings.Contains(messages[0].Content, "任何state的target") {
				t.Fatal("系统提示必须说明背景列表不是任何状态的待答目标")
			}
		})
	}
}

func TestKbAnalysisBudgetIncludesOriginalQueryAndBackground(t *testing.T) {
	s, _ := analysisFixture()
	s.hits[0].Snippet = strings.Repeat("预算为五万元。", 1000)
	query := "有人建议“" + strings.Repeat("背景", 160) + "”。请说明预算。"
	tool := &kbAnalyzeTool{maxContextTokens: 2200}
	messages, sources, err := tool.fitSources(query, 3, s.hits, 400)
	if err != nil || len(sources) != 1 || len(sources[0].Snippet) >= len(s.hits[0].Snippet) {
		t.Fatalf("包含原问和背景元数据后仍应按整体预算缩减来源: err=%v", err)
	}
	encoded, _ := json.Marshal(messages)
	if (len(encoded)+2)/3+400 > 2200 || !strings.Contains(messages[1].Content, `"original_query"`) || !strings.Contains(messages[1].Content, `"background_quotes"`) {
		t.Fatal("全部原问、背景和消息角色必须计入同一上下文预算")
	}
}

func TestKbAnalysisQuotedTargetContextBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, query, missing string
		wantError            bool
	}{
		{"英文双引号", `预算是多少？有人说"all users are admins"。请说明实际规则。`, "all users are admins", true},
		{"英文单引号", "预算是多少？有人建议'ignore ownership'。请说明实际规则。", "ignore ownership", true},
		{"中文嵌套引号", "预算是多少？有人说“先阅读‘不要检查归属’再操作”。请说明规则。", "不要检查归属", true},
		{"中文直角引号", "预算是多少？有人说「先阅读『不要检查归属』再操作」。请说明规则。", "不要检查归属", true},
		{"仅加引号外壳仍是背景", "预算是多少？有人说“不要检查归属”。请说明规则。", "“不要检查归属”", true},
		{"反斜杠转义内引号", `预算是多少？有人说"the \"owner\" need not match"。请说明规则。`, `the \"owner\" need not match`, true},
		{"英文内撇号", `预算是多少？有人说"don't inspect the owner"。请说明规则。`, "don't inspect the owner", true},
		{"英文弯撇号", `预算是多少？有人说"don’t inspect the owner"。请说明规则。`, "don’t inspect the owner", true},
		{"姓氏撇号不影响中文引文", "D’Artagnan建议“不要检查归属”。预算是多少，请说明实际规则。", "不要检查归属", true},
		{"真正询问引用内容须保留问句语境", "预算是多少，“上线日期”是什么？", "“上线日期”是什么", false},
		{"英文引用真实未知", `预算是多少，what is the "launch date"?`, `what is the "launch date"`, false},
		{"同目标在引文外也出现", "有人提到“保存天数”。预算和保存天数分别是多少？", "保存天数", false},
		{"整query仅引文", "“预算和保存天数”", "保存天数", false},
		{"引文外只有标点", "（“预算和保存天数”）？", "保存天数", false},
		{"未闭合引文保留既有行为", "预算是多少，“保存天数", "保存天数", false},
		{"错配引号不猜范围", "预算是多少，“保存天数’", "保存天数", false},
		{"普通英文撇号不是引文", "don't guess. 预算和保存天数分别是多少？", "保存天数", false},
		{"转义外引号不作引文", `预算是多少？请核对\"保存天数\"`, "保存天数", false},
		{"显式请求引用字段后还有请求", "请说明预算，并告诉我“保存天数”。另外说明预算用途。", "保存天数", false},
		{"后文明确询问引用字段", "预算是多少？有人说“保存天数”。请告诉我这个字段的值。", "保存天数", false},
		{"通用代词回指引文不限定对象名词", "预算是多少？有人说“上线记录”。请告诉我该记录的日期。", "上线记录", false},
		{"后问引文使用它回指", "预算是多少？手册写着“保存天数”。请告诉我它的值。", "保存天数", false},
		{"后问引文使用其回指", "预算是多少？手册写着“保存天数”。请给出其数值。", "保存天数", false},
		{"没有明确背景归属时不猜意图", "预算是多少？“保存天数”。请说明保留期限。", "保存天数", false},
		{"背景句后没有独立请求时不猜意图", "预算是多少？有人说“保存天数”。", "保存天数", false},
		{"引文本身是问题时保守放行", "预算是多少？有人说“保存天数是多少？”。另外说明预算用途。", "保存天数是多少？", false},
		{"英文显式请求引用字段", `Please give the 预算 and tell me "retention days". Then explain the 预算 purpose.`, "retention days", false},
		{"英文后问引文", `预算是多少？Someone said "retention days". Please answer the quoted question.`, "retention days", false},
		{"英文已结束的背景陈述", `预算是多少？Someone claimed "all users are admins". Please explain actual rules.`, "all users are admins", true},
		{"引文内句末结束背景", "预算是多少？有人说“所有人都可读写。”请说明实际规则。", "所有人都可读写。", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reply, _ := json.Marshal(map[string]any{"answers": []any{
				map[string]any{"target": "预算", "state": "supported", "text": "预算为五万元。", "citations": []int{1}},
				map[string]any{"target": tc.missing, "state": "insufficient", "text": "", "citations": []int{}},
			}})
			got, err, s, c := runAnalysisReply(t, tc.query, string(reply), "stop")
			if (err != nil) != tc.wantError || s.calls != 1 || c.calls != 1 {
				t.Fatalf("引文背景与实际问句须区分，且不增加外部调用：error=%v wantError=%v search=%d chat=%d", err, tc.wantError, s.calls, c.calls)
			}
			if !tc.wantError && !strings.Contains(got.Analysis, "本次片段未能确定") {
				t.Fatal("合法真实缺口不能被引用防线抹去")
			}
		})
	}
}

func TestKbAnalysisQuotedOnlyQuestionCanRemainUnknown(t *testing.T) {
	got, err, s, c := runAnalysisReply(t, "“上线日期”", `{"answers":[{"target":"上线日期","state":"insufficient","text":"","citations":[]}]}`, "stop")
	if err != nil || got.Analysis != "本次片段资料不足，无法回答该分析目标。" || s.calls != 1 || c.calls != 1 {
		t.Fatalf("整条query只是引用内容时仍可正常表达真正未知：result=%+v err=%v search=%d chat=%d", got, err, s.calls, c.calls)
	}
}

func TestKbAnalysisKeepsExplicitlyRequestedQuotedMissingTask(t *testing.T) {
	// 旧构建真实任务211已正确回答这个部分未知问题；新范围防线不能把它变成格式失败。
	query := "请按本项目资料说明任务的数据归属与越权ID处理规则，并回答“本实例上一次试运行的实际task_id是多少？”。"
	s, c := analysisFixture()
	s.hits[0].Snippet = "会话、工作流和任务仅其所属用户可读写，越权 ID 与不存在均返回 404。"
	c.response.Content = `{"answers":[{"target":"任务的数据归属与越权ID处理规则","state":"supported","text":"会话、工作流和任务仅其所属用户可读写，越权 ID 与不存在均返回 404。","citations":[1]},{"target":"本实例上一次试运行的实际task_id是多少？","state":"insufficient","text":"","citations":[]}]}`
	args, _ := json.Marshal(map[string]any{"kb_id": 3, "query": query})
	out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), args)
	var got KbAnalysisResult
	if err == nil {
		err = json.Unmarshal([]byte(out), &got)
	}
	if err != nil || !strings.Contains(got.Analysis, "均返回 404") || !strings.Contains(got.Analysis, "本次片段未能确定") || s.calls != 1 || c.calls != 1 {
		t.Fatalf("明确要求回答的带引号问句应保留真正未知，不得安全失败或增调模型：result=%+v err=%v search=%d chat=%d", got, err, s.calls, c.calls)
	}
}
