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
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/vectordb"
)

type coverageCandidate struct {
	id, kb, doc int64
	score       float32
	filtered    bool
}

type coverageStore struct {
	vectordb.Store
	candidates []coverageCandidate
	calls      int
	owner, kb  int64
	topK       int
}

func (s *coverageStore) Search(_ context.Context, _ []float32, owner, kb int64, topK int) ([]vectordb.Hit, error) {
	s.calls++
	s.owner, s.kb, s.topK = owner, kb, topK
	hits := make([]vectordb.Hit, 0, len(s.candidates))
	for _, item := range s.candidates {
		hits = append(hits, vectordb.Hit{ID: uint64(item.id), Score: item.score})
	}
	return hits, nil
}

type coverageChunks struct {
	model.AiKbChunkModel
	candidates []coverageCandidate
	calls      int
	owner, kb  int64
	ids        []uint64
}

func (m *coverageChunks) FindReadyByIDs(_ context.Context, ids []uint64, owner, kb int64) ([]*model.KbChunkRow, error) {
	m.calls++
	m.owner, m.kb, m.ids = owner, kb, slices.Clone(ids)
	rows := make([]*model.KbChunkRow, 0, len(m.candidates))
	seen := make(map[int64]bool)
	// 返回乱序、且只包含模型层已经复核可用的候选，覆盖选择不能重新加入已过滤ID。
	for i := len(m.candidates) - 1; i >= 0; i-- {
		item := m.candidates[i]
		if !item.filtered && !seen[item.id] {
			rows = append(rows, &model.KbChunkRow{Id: item.id, KbId: item.kb, DocId: item.doc, DocTitle: "资料", Content: "片段资料" + strings.Repeat("资", 4), TokenCount: 1})
			seen[item.id] = true
		}
	}
	return rows, nil
}

func TestKbSearcherDocumentCoverage(t *testing.T) {
	tests := []struct {
		name       string
		kb         int64
		candidates []coverageCandidate
		want       []int64
		budget     int64
	}{
		{name: "crowded_cross_document", kb: 6, candidates: []coverageCandidate{
			// 固定失败问题的真实候选排名，回归本身不再调用供应商。
			{id: 38, kb: 6, doc: 21, score: .672353}, {id: 42, kb: 6, doc: 23, score: .67178404},
			{id: 37, kb: 6, doc: 21, score: .63907075}, {id: 40, kb: 6, doc: 22, score: .61936486},
			{id: 39, kb: 6, doc: 22, score: .6162329}, {id: 41, kb: 6, doc: 23, score: .5622389},
			{id: 45, kb: 6, doc: 24, score: .55551547}, {id: 46, kb: 6, doc: 24, score: .5202169},
			{id: 43, kb: 6, doc: 24, score: .4750797}, {id: 44, kb: 6, doc: 24, score: .43743837},
		}, want: []int64{38, 42, 37, 40, 45}},
		{name: "dominant_four_of_five_keeps_permission_context", kb: 6, candidates: []coverageCandidate{
			// 原178的前五及首轮覆盖结果中观察到的低分文档代表。
			{id: 46, kb: 6, doc: 24, score: .6324503}, {id: 43, kb: 6, doc: 24, score: .6016855},
			{id: 44, kb: 6, doc: 24, score: .5508078}, {id: 45, kb: 6, doc: 24, score: .55040085},
			{id: 42, kb: 6, doc: 23, score: .4860351}, {id: 38, kb: 6, doc: 21, score: .44723088},
			{id: 39, kb: 6, doc: 22, score: .43500143},
		}, want: []int64{46, 43, 44, 45, 42}},
		{name: "dominant_three_of_five_keeps_permission_context", kb: 6, candidates: []coverageCandidate{
			{id: 43, kb: 6, doc: 24, score: .6459849}, {id: 46, kb: 6, doc: 24, score: .62369126},
			{id: 45, kb: 6, doc: 24, score: .60238063}, {id: 42, kb: 6, doc: 23, score: .57905424},
			{id: 41, kb: 6, doc: 23, score: .54597926}, {id: 39, kb: 6, doc: 22, score: .54231334},
			{id: 38, kb: 6, doc: 21, score: .5104296},
		}, want: []int64{43, 46, 45, 42, 41}},
		{name: "short_candidates_keep_every_ranked_chunk", kb: 6, candidates: []coverageCandidate{
			{id: 4, kb: 6, doc: 23, score: .6}, {id: 3, kb: 6, doc: 24, score: .7},
			{id: 2, kb: 6, doc: 24, score: .8}, {id: 1, kb: 6, doc: 24, score: .9},
		}, want: []int64{1, 2, 3, 4}},
		{name: "single_document_preserves_top_five", kb: 6, candidates: []coverageCandidate{
			{id: 7, kb: 6, doc: 21, score: .3}, {id: 6, kb: 6, doc: 21, score: .4}, {id: 5, kb: 6, doc: 21, score: .5},
			{id: 4, kb: 6, doc: 21, score: .6}, {id: 3, kb: 6, doc: 21, score: .7}, {id: 2, kb: 6, doc: 21, score: .8}, {id: 1, kb: 6, doc: 21, score: .9},
		}, want: []int64{1, 2, 3, 4, 5}},
		{name: "six_documents_choose_best_five", kb: 6, candidates: []coverageCandidate{
			{id: 1, kb: 6, doc: 1, score: 1}, {id: 2, kb: 6, doc: 1, score: .99}, {id: 3, kb: 6, doc: 2, score: .98},
			{id: 4, kb: 6, doc: 3, score: .97}, {id: 5, kb: 6, doc: 4, score: .96}, {id: 6, kb: 6, doc: 5, score: .95}, {id: 7, kb: 6, doc: 6, score: .94},
		}, want: []int64{1, 3, 4, 5, 6}},
		{name: "ties_keep_id_order", kb: 6, candidates: []coverageCandidate{
			{id: 9, kb: 6, doc: 1, score: .8}, {id: 6, kb: 6, doc: 2, score: .8}, {id: 3, kb: 6, doc: 3, score: .8},
			{id: 8, kb: 6, doc: 4, score: .8}, {id: 7, kb: 6, doc: 5, score: .8}, {id: 2, kb: 6, doc: 1, score: .8}, {id: 1, kb: 6, doc: 1, score: .8},
		}, want: []int64{1, 3, 6, 7, 8}},
		{name: "model_filtered_candidates_stay_absent", kb: 6, candidates: []coverageCandidate{
			{id: 99, kb: 6, doc: 99, score: 1, filtered: true}, {id: 1, kb: 6, doc: 21, score: .9}, {id: 2, kb: 6, doc: 21, score: .8},
			{id: 3, kb: 6, doc: 22, score: .7}, {id: 4, kb: 6, doc: 23, score: .6}, {id: 5, kb: 6, doc: 24, score: .5},
		}, want: []int64{1, 2, 3, 4, 5}},
		{name: "all_kbs_use_kb_and_doc_pair", kb: 0, candidates: []coverageCandidate{
			{id: 1, kb: 2, doc: 10, score: 1}, {id: 2, kb: 2, doc: 10, score: .9}, {id: 3, kb: 2, doc: 11, score: .85},
			{id: 4, kb: 3, doc: 10, score: .8}, {id: 5, kb: 3, doc: 11, score: .79}, {id: 6, kb: 3, doc: 12, score: .78},
		}, want: []int64{1, 3, 4, 5, 6}},
		{name: "duplicate_vector_ids_keep_unique_model_rows", kb: 6, candidates: []coverageCandidate{
			{id: 1, kb: 6, doc: 21, score: .9}, {id: 2, kb: 6, doc: 21, score: .8}, {id: 2, kb: 6, doc: 21, score: .8},
			{id: 3, kb: 6, doc: 22, score: .7}, {id: 4, kb: 6, doc: 23, score: .6}, {id: 5, kb: 6, doc: 24, score: .5},
		}, want: []int64{1, 2, 3, 4, 5}},
		{name: "small_budget_keeps_utf8_bound", kb: 6, candidates: []coverageCandidate{
			{id: 1, kb: 6, doc: 21, score: .9}, {id: 2, kb: 6, doc: 22, score: .8},
		}, budget: 2, want: []int64{1}},
		{name: "all_filtered_return_empty", kb: 6, candidates: []coverageCandidate{{id: 99, kb: 6, doc: 99, score: 1, filtered: true}}},
		{name: "no_vector_hits_skip_model", kb: 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var embeddingCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				embeddingCalls.Add(1)
				if r.URL.Path != "/embeddings" {
					t.Error("检索只能调用本地Embedding桩")
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"index": 0, "embedding": []float32{1, 0}}}})
			}))
			defer server.Close()
			embedder, err := llm.NewEmbedder(llm.EmbedConfig{BaseURL: server.URL, APIKey: "local-test", Model: "local-test", Dims: 2})
			if err != nil {
				t.Fatal(err)
			}
			store := &coverageStore{candidates: tt.candidates}
			chunks := &coverageChunks{candidates: tt.candidates}
			budget := tt.budget
			if budget == 0 {
				budget = 8000
			}
			searcher := &kbSearcher{store: store, embedder: embedder, chunks: chunks, snippetBudget: budget}
			hits, err := searcher.Search(context.Background(), 7, tt.kb, "同时核对配置与权限")
			if err != nil {
				t.Fatal(err)
			}
			ids := make([]int64, 0, len(hits))
			totalBytes := 0
			for _, hit := range hits {
				ids = append(ids, hit.ChunkID)
				totalBytes += len(hit.Snippet)
				if !utf8.ValidString(hit.Snippet) {
					t.Fatal("片段必须保留有效UTF-8")
				}
			}
			if !slices.Equal(ids, tt.want) {
				t.Fatalf("selected ids=%v want=%v", ids, tt.want)
			}
			if totalBytes > int(budget*3) {
				t.Fatalf("片段字节%d超出预算%d", totalBytes, budget*3)
			}
			if embeddingCalls.Load() != 1 || store.calls != 1 || store.owner != 7 || store.kb != tt.kb || store.topK != 20 {
				t.Fatal("覆盖选择不能增加Embedding/向量调用或放宽过滤")
			}
			wantModelCalls := 1
			if len(tt.candidates) == 0 {
				wantModelCalls = 0
			}
			if chunks.calls != wantModelCalls {
				t.Fatalf("model calls=%d want=%d", chunks.calls, wantModelCalls)
			}
			if wantModelCalls == 1 {
				if chunks.owner != 7 || chunks.kb != tt.kb || len(chunks.ids) != len(tt.candidates) {
					t.Fatal("MySQL复核须使用原候选全集及身份范围")
				}
			}
		})
	}
}
