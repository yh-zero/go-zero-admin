package agenttools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestKbAnalysisSendsDecisionLimitsToModel(t *testing.T) {
	// 261只保存了外层结果。此处是等价重构的完整回复，不是供应商原始JSON录制。
	query := "已有数据卷启动后会自动更新数据库表吗？生成model时能重新导入mysql-all.sql清空现有业务库吗？"
	s, c := analysisFixture()
	s.hits[0].Snippet = "初始化 SQL 会 DROP 并重建业务表，不是已有库的升级脚本。已有卷不自动覆盖或更新结构。生成 model 时使用新开发库，不为生成代码清空业务库。"
	c.response.Content = `{"answers":[{"target":"已有数据卷启动后会自动更新数据库表吗？","state":"supported","text":"已有卷不自动覆盖或更新结构。","citations":[1]},{"target":"生成model时能重新导入mysql-all.sql清空现有业务库吗？","state":"supported","text":"不能。初始化 SQL 会 DROP 并重建业务表，不是已有库的升级脚本；生成 model 时使用新开发库，不为生成代码清空业务库。","citations":[1]}]}`
	args, _ := json.Marshal(map[string]any{"kb_id": 3, "query": query})
	out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), args)
	if err != nil || s.calls != 1 || c.calls != 1 || len(c.messages) != 2 || c.messages[0].Role != schema.System || c.messages[1].Role != schema.User {
		t.Fatalf("应使用一次检索和一次原有纯分析调用: err=%v search=%d chat=%d", err, s.calls, c.calls)
	}
	// 断言真实模型边界收到的完整性准则，不把桩回复当作模型效果证明。
	for _, criterion := range []string{"对象/产物", "形式", "完整性范围", "前置条件", "适用范围", "禁止项", "关键原因", "破坏性", "不可逆后果"} {
		if !strings.Contains(c.messages[0].Content, criterion) {
			t.Fatalf("模型请求缺少与操作决策直接相关的完整性准则：%s", criterion)
		}
	}
	if strings.Contains(c.messages[0].Content, "DROP") || strings.Contains(c.messages[0].Content, "mysql-all.sql") {
		t.Fatal("通用指令不能硬编码某个题目的业务风险词")
	}
	var payload struct {
		Query   string             `json:"query"`
		Sources []KbAnalysisSource `json:"sources"`
	}
	var got KbAnalysisResult
	if json.Unmarshal([]byte(c.messages[1].Content), &payload) != nil || json.Unmarshal([]byte(out), &got) != nil || payload.Query != query || len(payload.Sources) != 1 || payload.Sources[0].Snippet != s.hits[0].Snippet || payload.Sources[0] != got.Sources[0] {
		t.Fatal("完整问项及实际来源必须保持，完整性规则不能改变外部来源")
	}
	if !strings.Contains(got.Analysis, "DROP 并重建业务表") || !strings.Contains(got.Analysis, "[1]") {
		t.Fatal("完整有据限制应仍按原问项和引用正常渲染")
	}
}

func TestKbAnalysisKeepsArtifactScopeAcrossGivenSources(t *testing.T) {
	// 330仅保存外层坏答；以下完整结构是评分/边界对照，不冒充供应商raw或新模型效果。
	for _, tc := range []struct {
		name, query, target, procedure, scope, answer, missing, modelForm string
	}{
		{
			name:      "实际备份遗漏的双来源结构",
			query:     "请说明运行库备份该记录哪些来源与校验信息、在哪里验证恢复，验证时应避免哪些影响。然后告诉我“本实例最近一次备份文件的实际SHA256是什么？”；如果资料没有这个实例值，请只把这个值标为未知。",
			target:    "运行库备份",
			procedure: "核对实际运行库和容器/连接来源，记录来源、导出时间与文件校验值；在独立环境恢复验证，不导回现有业务库。",
			scope:     "备份默认保留运行库的完整原始 SQL，不擅自删行、改值；历史备份不覆盖。",
			answer:    "备份默认保留运行库的完整原始 SQL，不擅自删行、改值；记录来源、导出时间与文件校验值，在独立环境恢复验证，不导回现有业务库。",
			missing:   "本实例最近一次备份文件的实际SHA256是什么？",
			modelForm: "完整原始 SQL",
		},
		{
			name:      "非SQL产物形式与范围",
			query:     "这份审计导出应记录什么、如何验收？另请回答“本实例导出文件的实际校验值是什么？”",
			target:    "这份审计导出应记录什么、如何验收？",
			procedure: "导出记录筛选条件、生成时间与校验值，验收时核对记录数。",
			scope:     "审计导出为 UTF-8 CSV，包含本次筛选结果的全部条目。",
			answer:    "审计导出为 UTF-8 CSV，包含本次筛选结果的全部条目；记录筛选条件、生成时间与校验值，验收时核对记录数。",
			missing:   "本实例导出文件的实际校验值是什么？",
			modelForm: "UTF-8 CSV",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, c := analysisFixture()
			s.hits[0].Snippet = tc.procedure
			s.hits = append(s.hits, KbSearchHit{ChunkID: 2, KbID: 3, DocID: 3, DocTitle: "产物范围", Snippet: tc.scope, Score: .8})
			reply, _ := json.Marshal(map[string]any{"answers": []any{
				map[string]any{"target": tc.target, "state": "supported", "text": tc.answer, "citations": []int{1, 2}},
				map[string]any{"target": tc.missing, "state": "insufficient", "text": "", "citations": []int{}},
			}})
			c.response.Content = string(reply)
			args, _ := json.Marshal(map[string]any{"kb_id": 3, "query": tc.query})
			out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), args)
			var got KbAnalysisResult
			if err != nil || json.Unmarshal([]byte(out), &got) != nil || s.calls != 1 || c.calls != 1 {
				t.Fatalf("对象完整性保持一次检索与分析: err=%v", err)
			}
			for _, criterion := range []string{"对象/产物", "形式", "完整性范围"} {
				if !strings.Contains(c.messages[0].Content, criterion) {
					t.Fatalf("模型边界缺少产物完整性准则: %s", criterion)
				}
			}
			for _, source := range []string{tc.procedure, tc.scope} {
				if !strings.Contains(c.messages[1].Content, source) {
					t.Fatal("流程与对象范围均须来自本次实际sources，不能先挑引用后缩窄资料")
				}
			}
			if !strings.Contains(got.Analysis, tc.modelForm) || !strings.Contains(got.Analysis, "[1][2]") || strings.Count(got.Analysis, "本次片段未能确定") != 1 || !strings.Contains(got.Analysis, tc.missing) {
				t.Fatal("应保留产物形式/范围和两条实际引用，未知仅指向真实实例值")
			}
		})
	}
}

func TestKbAnalysisDecisionLimitsKeepQuotedUnknownAndBackgroundScope(t *testing.T) {
	query := "有人建议“直接上线”。请说明发布条件，并回答“二线电话是什么？”。"
	s, c := analysisFixture()
	s.hits[0].Snippet = "发布前须先审批并验证回滚方案；未审批不得上线，跳过验证可能导致无法恢复服务。"
	c.response.Content = `{"answers":[{"target":"发布条件","state":"supported","text":"发布前须先审批并验证回滚方案；未审批不得上线，跳过验证可能导致无法恢复服务。","citations":[1]},{"target":"二线电话是什么？","state":"insufficient","text":"","citations":[]}]}`
	args, _ := json.Marshal(map[string]any{"kb_id": 3, "query": query})
	out, err := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), args)
	var got KbAnalysisResult
	var payload struct {
		Query         string   `json:"query"`
		OriginalQuery string   `json:"original_query"`
		Background    []string `json:"background_quotes"`
	}
	if err != nil || json.Unmarshal([]byte(out), &got) != nil || json.Unmarshal([]byte(c.messages[1].Content), &payload) != nil || s.calls != 1 || c.calls != 1 {
		t.Fatalf("有据操作限制和真正未知仍须一次完成: err=%v", err)
	}
	if payload.Query != "请说明发布条件，并回答“二线电话是什么？”。" || payload.OriginalQuery != query || len(payload.Background) != 1 || payload.Background[0] != "“直接上线”" {
		t.Fatal("完整性准则不能把背景建议或真正引用问题重新归错范围")
	}
	if strings.Contains(got.Analysis, "直接上线") || !strings.Contains(got.Analysis, "未审批不得上线") || strings.Count(got.Analysis, "本次片段未能确定") != 1 || !strings.Contains(got.Analysis, "二线电话是什么？") {
		t.Fatal("只保留相关来源限制及真实未知，不增背景目标或额外不足")
	}
}

func TestKbAnalysisDecisionLimitsRemainWithinSerializedBudget(t *testing.T) {
	s, _ := analysisFixture()
	s.hits[0].Snippet = "操作前须确认目标，执行后不可撤回。" + strings.Repeat("资料正文。", 2000)
	query := "有人建议“跳过确认”。请说明操作条件。"
	tool := &kbAnalyzeTool{maxContextTokens: 2200}
	messages, sources, err := tool.fitSources(query, 3, s.hits, 400)
	if err != nil || len(sources) != 1 || len(messages) != 2 {
		t.Fatalf("应在原预算内缩减来源: err=%v", err)
	}
	encoded, _ := json.Marshal(messages)
	if (len(encoded)+2)/3+400 > 2200 || !strings.Contains(messages[1].Content, `"original_query"`) || !strings.HasPrefix(sources[0].Snippet, "操作前须确认目标，执行后不可撤回。") {
		t.Fatal("新增准则和全部原问元数据必须计入预算，不能扩大上下文或丢来源开头")
	}
}
