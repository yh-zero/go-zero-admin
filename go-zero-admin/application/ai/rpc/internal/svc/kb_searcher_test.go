package svc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/vectordb"
)

func TestKbSearcherSkipsEmbeddingWhenRequestBudgetExhausted(t *testing.T) {
	// 不提供 embedder/store，额度耗尽必须在任何外部访问前结束。
	searcher := &kbSearcher{snippetBudget: 8000}
	hits, err := searcher.Search(agenttools.WithKbSearchBudget(context.Background(), 0), 7, 0, "资料")
	if err != nil || len(hits) != 0 {
		t.Fatal("耗尽预算不应访问 Embedding 或返回资料")
	}
}

type recallStore struct {
	vectordb.Store
	owner, kb int64
	topK      int
}

func (s *recallStore) Search(_ context.Context, _ []float32, owner, kb int64, topK int) ([]vectordb.Hit, error) {
	s.owner, s.kb, s.topK = owner, kb, topK
	return []vectordb.Hit{{ID: 9, Score: .8}, {ID: 6, Score: .8}, {ID: 3, Score: .8}, {ID: 8, Score: .8}, {ID: 7, Score: .8}, {ID: 2, Score: .9}, {ID: 1, Score: 1}}, nil
}

type recallChunks struct {
	model.AiKbChunkModel
	owner, kb int64
}

func (m *recallChunks) FindReadyByIDs(_ context.Context, ids []uint64, owner, kb int64) ([]*model.KbChunkRow, error) {
	m.owner, m.kb = owner, kb
	rows := make([]*model.KbChunkRow, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, &model.KbChunkRow{Id: int64(id), KbId: 2, DocId: 20, DocTitle: "文档", Content: string(rune('A'+id)) + strings.Repeat("资", 4), TokenCount: 1})
	}
	return rows, nil
}

func TestKbSearcherSortsTiesAndKeepsTopFiveWithinBudget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			t.Error("unexpected provider path")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"index": 0, "embedding": []float32{1, 0}}}})
	}))
	defer server.Close()
	embedder, err := llm.NewEmbedder(llm.EmbedConfig{BaseURL: server.URL, APIKey: "local-test", Model: "local-test", Dims: 2})
	if err != nil {
		t.Fatal(err)
	}
	store, chunks := &recallStore{}, &recallChunks{}
	searcher := &kbSearcher{embedder: embedder, store: store, chunks: chunks, snippetBudget: 8000}
	hits, err := searcher.Search(context.Background(), 7, 2, "资料")
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 5 {
		t.Fatalf("topK=%d want 5", len(hits))
	}
	for i, id := range []int64{1, 2, 3, 6, 7} {
		if hits[i].ChunkID != id || !strings.HasPrefix(hits[i].Snippet, string(rune('A'+id))) {
			t.Fatalf("unstable ranking at %d, got=%+v", i, hits)
		}
	}
	if store.owner != 7 || store.kb != 2 || store.topK != 20 || chunks.owner != 7 || chunks.kb != 2 {
		t.Fatal("owner/kb/topK filter was lost")
	}
	searcher.snippetBudget = 2
	hits, err = searcher.Search(context.Background(), 7, 2, "资料")
	if err != nil {
		t.Fatal(err)
	}
	var bytes int
	for _, h := range hits {
		bytes += len(h.Snippet)
		if !utf8.ValidString(h.Snippet) {
			t.Fatal("truncated invalid UTF-8")
		}
	}
	if bytes > 6 {
		t.Fatalf("budget bytes=%d want <=6", bytes)
	}
}
