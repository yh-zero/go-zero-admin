package svc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"go-zero-admin/application/ai/rpc/internal/config"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/vectordb"
)

type vectorReadinessFixture struct {
	vectordb.Store
	err error
}

func (s *vectorReadinessFixture) Ready(context.Context) error { return s.err }
func (s *vectorReadinessFixture) Search(context.Context, []float32, int64, int64, int) ([]vectordb.Hit, error) {
	return nil, nil
}

func TestSearchDoesNotEmbedUntilVectorDependencyReady(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"index":0,"embedding":[1,0,0,0]}]}`)
	}))
	defer server.Close()
	embedder, err := llm.NewEmbedder(llm.EmbedConfig{BaseURL: server.URL, APIKey: "local-test-only", Model: "fixture", Dims: 4})
	if err != nil {
		t.Fatal(err)
	}
	unavailable := errors.New("index not ready")
	store := &vectorReadinessFixture{err: unavailable}
	searcher := &kbSearcher{store: store, embedder: embedder, snippetBudget: 1000}
	if _, err := searcher.Search(context.Background(), 7, 5, "fixture"); !errors.Is(err, unavailable) {
		t.Errorf("索引未就绪应失败: %v", err)
	}
	if calls.Load() != 0 {
		t.Errorf("索引未就绪产生了%d次Embedding", calls.Load())
	}
	store.err = nil
	if _, err := searcher.Search(context.Background(), 7, 5, "fixture"); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("依赖恢复后仅执行一次检索Embedding: %d", calls.Load())
	}
}

func TestConfiguredVectorStoreSurvivesStartupOutage(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	var c config.Config
	c.Qdrant.Address = address
	c.AI.EmbeddingProvider = "qwen"
	c.AI.EmbeddingModel = "fixture"
	c.AI.EmbeddingDimensions = 4
	store := newVectorStore(c)
	if store == nil {
		t.Fatal("已配置的向量存储遇到启动暂态故障应保留恢复入口")
	}
	t.Cleanup(func() { _ = store.Close() })
}
