package svc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/llm"
)

func TestKbSearcherAnalysisCandidatesKeepOriginalRanksAndCallCounts(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"index": 0, "embedding": []float32{1, 0}}}})
	}))
	defer server.Close()
	embed, err := llm.NewEmbedder(llm.EmbedConfig{BaseURL: server.URL, APIKey: "local-test", Model: "local-test", Dims: 2})
	if err != nil {
		t.Fatal(err)
	}
	candidates := []coverageCandidate{{id: 32, kb: 5, doc: 19, score: .7295935}, {id: 33, kb: 5, doc: 19, score: .6954123}, {id: 28, kb: 5, doc: 18, score: .69326115}, {id: 25, kb: 5, doc: 17, score: .664142}, {id: 26, kb: 5, doc: 17, score: .64852214}, {id: 34, kb: 5, doc: 19, score: .6473211}, {id: 24, kb: 5, doc: 17, score: .6429526}, {id: 35, kb: 5, doc: 19, score: .6378398}, {id: 23, kb: 5, doc: 17, score: .59021354}, {id: 29, kb: 5, doc: 18, score: .58928967}, {id: 27, kb: 5, doc: 18, score: .58179605}, {id: 31, kb: 5, doc: 19, score: .5597898}, {id: 36, kb: 5, doc: 20, score: .54100186}, {id: 30, kb: 5, doc: 18, score: .44942948}, {id: 99, kb: 5, doc: 99, score: .4, filtered: true}}
	store, chunks := &coverageStore{candidates: candidates}, &coverageChunks{candidates: candidates}
	searcher := &kbSearcher{store: store, embedder: embed, chunks: chunks, snippetBudget: 8000}
	hits, err := searcher.SearchAnalysis(context.Background(), 7, 5, "同时核对两类配置与版本")
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]int64, 0, len(hits))
	for _, hit := range hits {
		ids = append(ids, hit.ChunkID)
	}
	want := []int64{32, 33, 28, 25, 26, 34, 24, 35, 23, 29, 27, 31, 36, 30}
	if !slices.Equal(ids, want) || calls.Load() != 1 || store.calls != 1 || chunks.calls != 1 || store.owner != 7 || chunks.owner != 7 || store.kb != 5 || chunks.kb != 5 || store.topK != 20 {
		t.Fatalf("分析须一次原有检索/复核保留低排名真实候选: %v", ids)
	}
	for i, hit := range hits {
		if hit.Score != candidates[i].score {
			t.Fatal("不得重写实际向量score")
		}
	}
	hits, err = searcher.SearchAnalysis(agenttools.WithKbSearchBudget(context.Background(), 0), 7, 5, "没有预算")
	if err != nil || len(hits) != 0 || calls.Load() != 1 || store.calls != 1 || chunks.calls != 1 {
		t.Fatal("耗尽预算仍须零额外调用")
	}
}

type analysisPrefixChunks struct {
	model.AiKbChunkModel
	content string
}

func (m *analysisPrefixChunks) FindReadyByIDs(context.Context, []uint64, int64, int64) ([]*model.KbChunkRow, error) {
	return []*model.KbChunkRow{{Id: 1, KbId: 5, DocId: 10, Content: m.content, TokenCount: 500, DocTitle: "JSON原文"}}, nil
}

func TestKbSearcherAnalysisTruncatesLiteralPrefixRatherThanRewritingJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"index": 0, "embedding": []float32{1, 0}}}})
	}))
	defer server.Close()
	embed, err := llm.NewEmbedder(llm.EmbedConfig{BaseURL: server.URL, APIKey: "local-test", Model: "local-test", Dims: 2})
	if err != nil {
		t.Fatal(err)
	}
	store := &coverageStore{candidates: []coverageCandidate{{id: 1, kb: 5, doc: 10, score: .9}}}
	chunks := &analysisPrefixChunks{content: `["中文中文"]`}
	s := &kbSearcher{store: store, embedder: embed, chunks: chunks, snippetBudget: 2}
	hits, err := s.SearchAnalysis(context.Background(), 7, 5, "原始片段")
	if err != nil || len(hits) != 1 || len(hits[0].Snippet) > 6 || !strings.HasPrefix(chunks.content, hits[0].Snippet) || !strings.Contains(hits[0].Snippet, "中") {
		t.Fatalf("分析资料只能截实际UTF8前缀，不能把原JSON改写成另一JSON: %+v %v", hits, err)
	}
}
