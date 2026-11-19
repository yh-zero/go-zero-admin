package aiservicelogic

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/internal/model"
)

// 候选策略保留原文连续区间。块尾优先挑最后四分之一内的换行，单行太长时仍须硬切。
func chunkTextByLine(content string) []model.KbChunkRow {
	var rows []model.KbChunkRow
	runes := []rune(strings.TrimSpace(content))
	for start := 0; start < len(runes); {
		end := min(start+kbChunkTargetTokens, len(runes))
		if end < len(runes) && runes[end] != '\n' {
			for pos := end - 1; pos >= start+3*kbChunkTargetTokens/4; pos-- {
				if runes[pos] == '\n' {
					end = pos + 1
					break
				}
			}
		}
		part := strings.TrimSpace(string(runes[start:end]))
		if part != "" {
			rows = append(rows, model.KbChunkRow{Seq: int64(len(rows)), Content: part, TokenCount: int64(utf8.RuneCountInString(part))})
		}
		if end == len(runes) {
			break
		}
		start = end - kbChunkOverlapTokens
	}
	return rows
}

// 题目在运行两种策略前固定。每题的证据为对应原文中包含锚点的一整行，同题检索只用问题文本。
var chunkComparisonCases = []struct{ file, question, anchor string }{
	{"01", "宿主上要从向量检索服务的网页控制台切到 gRPC，两个映射端口分别是什么？", "| Qdrant |"},
	{"01", "在容器里连接数据库用哪个服务名，宿主连接同库应选哪个端口？", "| MySQL |"},
	{"02", "本地已存业务数据，模型生成前可以直接拿初始化脚本升级旧库吗？", "不为生成代码清空业务库"},
	{"03", "交接给同事运行前端，锁文件更新后用什么命令把依赖装齐？", "pnpm install --frozen-lockfile"},
	{"04", "回归测试结束怎样确保开发者原本启动的服务仍在运行？", "保留用户要求运行的项目栈"},
	{"05", "模型切换到聊天服务后，知识库检索还依赖什么凭据和向量模型？", "Embedding 独立使用 Qwen"},
	{"06", "批量上传被打断，之前已接收和未发出的文件各自要怎么核对？", "尚未发出的文件不会自动补传"},
	{"07", "一个带审批的流程被我取消后，前面已经调用模型产生的动作会撤销吗？", "已经发生的模型调用或外部操作不会撤回"},
	{"08", "我只想退出此设备，后台哪个参数决定是否也踢掉其他设备？", "默认撤销当前会话"},
}

type chunkComparisonDoc struct{ name, body string }
type chunkComparisonHit struct {
	doc, text string
	score     float64
}

func rankChunkComparison(query string, hits []chunkComparisonHit, vectors []map[[2]rune]float64) []chunkComparisonHit {
	q := chunkComparisonTerms(query)
	ranked := append([]chunkComparisonHit(nil), hits...)
	for j := range ranked {
		for term, count := range q {
			ranked[j].score += count * vectors[j][term]
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].score > ranked[j].score })
	return ranked
}

// 无供应商、缓存及向量库的确定性检索桩。字符二元组余弦只度量这组对照，不能推定模型的语义表现。
func chunkComparisonTerms(s string) map[[2]rune]float64 {
	runes := []rune(strings.ToLower(s))
	freq := make(map[[2]rune]float64)
	for i := 1; i < len(runes); i++ {
		freq[[2]rune{runes[i-1], runes[i]}]++
	}
	var sum float64
	for _, v := range freq {
		sum += v * v
	}
	if sum > 0 {
		norm := math.Sqrt(sum)
		for term := range freq {
			freq[term] /= norm
		}
	}
	return freq
}

func TestKbChunkStructureComparison(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..", "..", "..")
	manifest, err := os.ReadFile(filepath.Join(root, "ai_document", "ai-rag", "business-corpus-provenance.json"))
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Documents []struct {
			Source string
			SHA    string `json:"sha256"`
		}
	}
	if err = json.Unmarshal(manifest, &corpus); err != nil {
		t.Fatal(err)
	}
	var docs []chunkComparisonDoc
	for _, entry := range corpus.Documents {
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.Source)))
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(raw)) != entry.SHA {
			t.Fatalf("冻结资料已变化: %s", entry.Source)
		}
		docs = append(docs, chunkComparisonDoc{filepath.Base(entry.Source)[:2], string(raw)})
	}
	if len(docs) != 8 {
		t.Fatalf("语料数量 %d", len(docs))
	}
	for _, strategy := range []struct {
		name  string
		split func(string) []model.KbChunkRow
	}{{"current", chunkText}, {"line_end", chunkTextByLine}} {
		var hits []chunkComparisonHit
		var vectors []map[[2]rune]float64
		var runes, batches, completeLines, totalLines int
		for _, doc := range docs {
			rows := strategy.split(doc.body)
			batches += (len(rows) + 9) / 10 // 固定模型每篇按每批10块计，不作真实供应商调用。
			for _, row := range rows {
				hits = append(hits, chunkComparisonHit{doc: doc.name, text: row.Content})
				vectors = append(vectors, chunkComparisonTerms(row.Content))
				runes += utf8.RuneCountInString(row.Content)
			}
			for _, line := range strings.Split(doc.body, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				totalLines++
				for _, row := range rows {
					if strings.Contains(row.Content, line) {
						completeLines++
						break
					}
				}
			}
		}
		var durations []time.Duration
		var recall, evidence int
		for _, c := range chunkComparisonCases {
			line := ""
			for _, doc := range docs {
				if doc.name != c.file {
					continue
				}
				for _, l := range strings.Split(doc.body, "\n") {
					if strings.Contains(l, c.anchor) {
						line = strings.TrimSpace(l)
						break
					}
				}
			}
			if line == "" {
				t.Fatalf("冻结证据缺失：%s", c.anchor)
			}
			var ranked []chunkComparisonHit
			// Windows 时钟可能低于单次检索分辨率。按 64 次无缓存请求计时，报告各批平均耗时的 P95。
			const requestsPerSample = 64
			for i := 0; i < 4; i++ {
				start := time.Now()
				for j := 0; j < requestsPerSample; j++ {
					ranked = rankChunkComparison(c.question, hits, vectors)
				}
				durations = append(durations, time.Since(start)/requestsPerSample)
			}
			foundDoc, foundLine := false, false
			for _, h := range ranked[:min(len(ranked), 5)] {
				if h.doc == c.file {
					foundDoc = true
					foundLine = foundLine || strings.Contains(h.text, line)
				}
			}
			if foundDoc {
				recall++
			}
			if foundLine {
				evidence++
			}
		}
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		t.Logf("%s: docs=%d cases=%d chunks=%d complete_source_lines=%d/%d embedded_runes=%d embedding_batches=%d doc_recall@5=%d/%d complete_line@5=%d/%d local_uncached_batch_mean_p95=%s", strategy.name, len(docs), len(chunkComparisonCases), len(hits), completeLines, totalLines, runes, batches, recall, len(chunkComparisonCases), evidence, len(chunkComparisonCases), durations[(95*len(durations)+99)/100-1])
	}
}
