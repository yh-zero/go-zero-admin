package agenttools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"
)

func TestKbAnalyzeRecordedProjectEvidenceReachesModelAndKeepsCitationIdentity(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	raw, err := os.ReadFile(filepath.Join(root, "ai_document", "ai-rag", "business-evidence-fixtures.json"))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(raw)
	if hex.EncodeToString(digest[:]) != "29fefd71054f2ee2e8be6fb03cac8adc7b50e83931c1ee19012a42fc2c06ac2c" {
		t.Fatal("真实候选夹具发生漂移，须单独复核来源")
	}
	var fixture struct {
		Observation struct {
			SHA256 string `json:"sha256"`
			KBID   int64  `json:"kb_id"`
		} `json:"observation"`
		Documents []struct {
			Source string `json:"source"`
			Title  string `json:"title"`
			SHA256 string `json:"sha256"`
		} `json:"documents"`
		Scenarios []struct {
			ID    string `json:"id"`
			Query string `json:"query"`
		} `json:"scenarios"`
		Candidates []struct {
			Rank    int     `json:"rank"`
			ChunkID int64   `json:"chunk_id"`
			DocID   int64   `json:"doc_id"`
			Title   string  `json:"title"`
			Snippet string  `json:"snippet"`
			Score   float32 `json:"score"`
			SHA256  string  `json:"content_sha256"`
		} `json:"candidates"`
	}
	if json.Unmarshal(raw, &fixture) != nil || len(fixture.Candidates) != 14 || fixture.Observation.SHA256 != "a8bb00d74a2c3f515e2029299e75809728395c56a85a541ef212b68d8bda257d" {
		t.Fatal("录制来源或候选结构无效")
	}
	compact := func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, s)
	}
	contents := map[string]string{}
	for _, doc := range fixture.Documents {
		body, e := os.ReadFile(filepath.Join(root, doc.Source))
		if e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != doc.SHA256 {
			t.Fatal("冻结原文已变化")
		}
		contents[doc.Title] = compact(string(body))
	}
	hits := make([]KbSearchHit, 0, len(fixture.Candidates))
	byChunk := map[int64]int{}
	for i, c := range fixture.Candidates {
		sum := sha256.Sum256([]byte(c.Snippet))
		if c.Rank != i+1 || hex.EncodeToString(sum[:]) != c.SHA256 || !strings.Contains(contents[c.Title], compact(c.Snippet)) {
			t.Fatal("录制片段不是对应冻结文档正文")
		}
		byChunk[c.ChunkID] = i + 1
		hits = append(hits, KbSearchHit{ChunkID: c.ChunkID, KbID: fixture.Observation.KBID, DocID: c.DocID, DocTitle: c.Title, Snippet: c.Snippet, Score: c.Score})
	}
	for _, scenario := range fixture.Scenarios {
		t.Run(scenario.ID, func(t *testing.T) {
			s := &candidateAnalysisSearcher{hits: hits}
			_, c := analysisFixture()
			answers := []kbModelAnswer{}
			var want []int64
			if scenario.ID == "acceptance-dev-code-and-env" {
				answers = []kbModelAnswer{{Target: "源码产物和环境变量各自需要的处理", State: "supported", Text: "先编译宿主Linux二进制并构建镜像，构建失败停止；环境变量变更需重新创建相关容器。", Citations: []int{byChunk[31], byChunk[32], byChunk[24]}}, {Target: "更新后怎样确认实际运行的是新版本", State: "supported", Text: "核对启动命令、配置挂载和运行产物摘要，并验证受影响功能。", Citations: []int{byChunk[34], byChunk[35]}}}
				want = []int64{32, 34, 24, 35, 31}
			} else {
				// 另一原问复用上述真实候选桩，不声称这是其实际向量排名或模型原始回复。
				answers = []kbModelAnswer{{Target: "宿主", State: "supported", Text: "使用application/*/etc/*.yaml，连接宿主映射端口。", Citations: []int{byChunk[25]}}, {Target: "Compose", State: "supported", Text: "使用docker/etc/*.yaml并挂载到容器，连接容器服务名；AI/App通过env_file注入后端.env.local。", Citations: []int{byChunk[24]}}}
				want = []int64{25, 24}
			}
			response, _ := json.Marshal(struct {
				Answers []kbModelAnswer `json:"answers"`
			}{answers})
			c.response.Content = string(response)
			args, _ := json.Marshal(map[string]any{"kb_id": fixture.Observation.KBID, "query": scenario.Query})
			out, e := NewKbAnalyze(s, func(context.Context, int64, int64) error { return nil }, c, 32000).Execute(WithActor(context.Background(), 7), args)
			if e != nil {
				t.Fatal(e)
			}
			var sent struct {
				Sources []KbAnalysisSource `json:"sources"`
			}
			var got KbAnalysisResult
			if json.Unmarshal([]byte(c.messages[1].Content), &sent) != nil || json.Unmarshal([]byte(out), &got) != nil || len(sent.Sources) != 14 || s.analysisCalls != 1 || c.calls != 1 {
				t.Fatal("实际来源必须一次传入原有分析")
			}
			for id, text := range map[int64]string{31: "不会编译 Go 源码", 34: "比较容器二进制与本地产物摘要", 24: "docker/etc/*.yaml", 25: "application/*/etc/*.yaml"} {
				if !strings.Contains(sent.Sources[byChunk[id]-1].Snippet, text) {
					t.Fatalf("关键片段%d未进入实际送模上下文", id)
				}
			}
			ids := make([]int64, 0, len(got.Sources))
			for i, source := range got.Sources {
				ids = append(ids, source.ChunkID)
				if source.Index != i+1 || source.KbSearchHit != sent.Sources[byChunk[source.ChunkID]-1].KbSearchHit {
					t.Fatal("最终来源与实际送入片段不一致")
				}
			}
			if !slices.Equal(ids, want) {
				t.Fatalf("最终引用身份%v want%v", ids, want)
			}
		})
	}
}
