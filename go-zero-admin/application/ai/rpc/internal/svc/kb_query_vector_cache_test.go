package svc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go-zero-admin/application/ai/rpc/internal/config"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/vectordb"
)

type queryVectorMemoryCache struct {
	mu      sync.Mutex
	values  map[string]string
	err     error
	writes  int
	seconds int
}

func (c *queryVectorMemoryCache) GetCtx(_ context.Context, key string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.values[key], c.err
}

func (c *queryVectorMemoryCache) SetexCtx(_ context.Context, key, value string, seconds int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	if c.values == nil {
		c.values = map[string]string{}
	}
	c.values[key] = value
	c.writes++
	c.seconds = seconds
	return nil
}

type queryVectorUsage struct {
	mu       sync.Mutex
	attempts []llm.UsageAttempt
	finished int
}

func (h *queryVectorUsage) Begin(_ context.Context, a llm.UsageAttempt) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.attempts = append(h.attempts, a)
	return nil
}
func (*queryVectorUsage) Dispatched(context.Context, string) error { return nil }
func (h *queryVectorUsage) Finish(context.Context, string, llm.UsageObservation) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.finished++
	return nil
}

func queryVectorFixture(t *testing.T, handler http.HandlerFunc, hook llm.UsageHook) (*llm.Embedder, config.ConfigAI) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	embedder, err := llm.NewEmbedder(llm.EmbedConfig{BaseURL: server.URL, APIKey: "local-test", Model: "fixture", Dims: 2, UsageHook: hook})
	if err != nil {
		t.Fatal(err)
	}
	return embedder, config.ConfigAI{EmbeddingProvider: "qwen", EmbeddingBaseURL: server.URL, EmbeddingModel: "fixture", EmbeddingDimensions: 2}
}

func queryVectorReply(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1,0]}],"usage":{"prompt_tokens":1,"total_tokens":1}}`))
}

func TestKbQueryVectorCacheReusesOnlyOwnerModelAndExactQuery(t *testing.T) {
	var calls atomic.Int32
	embedder, ai := queryVectorFixture(t, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); queryVectorReply(w) }, nil)
	cache := &queryVectorMemoryCache{}
	load := newKbQueryVectorCache(cache, embedder, ai)
	for _, input := range []struct {
		owner int64
		query string
	}{{7, "原始 Query"}, {7, "原始 Query"}, {8, "原始 Query"}, {7, "原始 query"}, {7, "原始  Query"}} {
		got, err := load.Embed(context.Background(), input.owner, input.query)
		if err != nil || len(got) != 2 || got[0] != 1 {
			t.Fatalf("向量读取失败: %v %v", got, err)
		}
		got[0] = 99
	}
	if calls.Load() != 4 || cache.writes != 4 || cache.seconds != 300 {
		t.Fatalf("精确命中只省一次外呼: calls=%d writes=%d ttl=%d", calls.Load(), cache.writes, cache.seconds)
	}
	for key, value := range cache.values {
		if strings.Contains(key, "原始") || strings.Contains(value, "Query") {
			t.Fatal("缓存不能保存查询明文")
		}
	}
	for _, field := range []string{"model", "endpoint", "dimensions"} {
		changed := ai
		switch field {
		case "model":
			changed.EmbeddingModel = "other"
		case "endpoint":
			changed.EmbeddingBaseURL += "/other"
		case "dimensions":
			changed.EmbeddingDimensions = 3
		}
		other := newKbQueryVectorCache(cache, embedder, changed)
		_, _ = other.Embed(context.Background(), 7, "原始 Query")
	}
	if calls.Load() != 7 {
		t.Fatal("模型、端点和维度必须各自隔离缓存")
	}
}

func TestKbQueryVectorCacheRejectsCorruptValuesAndFallsBackOnCacheFailure(t *testing.T) {
	var calls atomic.Int32
	embedder, ai := queryVectorFixture(t, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); queryVectorReply(w) }, nil)
	for _, corrupt := range []string{`[1]`, `[1,0,0]`, `["bad",0]`, `[1e100,0]`, `null`, `[null,1]`, `[null,null]`, `[0,0]`} {
		t.Run(corrupt, func(t *testing.T) {
			cache := &queryVectorMemoryCache{}
			load := newKbQueryVectorCache(cache, embedder, ai)
			if _, err := load.Embed(context.Background(), 7, "query"); err != nil {
				t.Fatal(err)
			}
			for key := range cache.values {
				cache.values[key] = corrupt
			}
			before := calls.Load()
			if vector, err := load.Embed(context.Background(), 7, "query"); err != nil || len(vector) != 2 || calls.Load() != before+1 {
				t.Fatalf("损坏缓存必须重新校验向量: %v %v", vector, err)
			}
		})
	}
	cache := &queryVectorMemoryCache{err: errors.New("isolated cache failure")}
	load := newKbQueryVectorCache(cache, embedder, ai)
	if vector, err := load.Embed(context.Background(), 7, "cache unavailable"); err != nil || len(vector) != 2 {
		t.Fatal("缓存故障不能代替正常供应商结果", err)
	}
}

func TestKbQueryVectorCacheMergesConcurrentRequestsAndKeepsOneUsageIdentity(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	hook := &queryVectorUsage{}
	embedder, ai := queryVectorFixture(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		calls.Add(1)
		close(started)
		select {
		case <-release:
			queryVectorReply(w)
		case <-r.Context().Done():
		}
	}, hook)
	load := newKbQueryVectorCache(&queryVectorMemoryCache{}, embedder, ai)
	first := llm.WithUsageIdentity(context.Background(), llm.UsageIdentity{UserID: 7, Source: "kb_search", RequestID: "first", SourceID: 3, StepIndex: -1})
	leader, cancelLeader := context.WithCancel(first)
	firstResult := make(chan error, 1)
	go func() { _, err := load.Embed(leader, 7, "same"); firstResult <- err }()
	<-started
	second := llm.WithUsageIdentity(context.Background(), llm.UsageIdentity{UserID: 7, Source: "evaluation", RequestID: "second", SourceID: 4, StepIndex: -1})
	secondResult := make(chan error, 1)
	go func() { _, err := load.Embed(second, 7, "same"); secondResult <- err }()
	waitForQueryVectorWaiters(t, load, 2)
	cancelLeader()
	if err := <-firstResult; !errors.Is(err, context.Canceled) {
		t.Fatalf("第一个等待者应独立取消: %v", err)
	}
	close(release)
	if err := <-secondResult; err != nil {
		t.Fatal("其他等待者仍应获得同次调用结果", err)
	}
	hook.mu.Lock()
	defer hook.mu.Unlock()
	if calls.Load() != 1 || len(hook.attempts) != 1 || hook.finished != 1 || hook.attempts[0].RequestID != "first" || hook.attempts[0].UserID != 7 {
		t.Fatalf("合并只能保留一个真实调用及最初计量身份: calls=%d attempts=%+v finished=%d", calls.Load(), hook.attempts, hook.finished)
	}
}

func waitForQueryVectorWaiters(t *testing.T, cache *kbQueryVectorCache, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		cache.mu.Lock()
		count := 0
		for _, flight := range cache.flights {
			count += flight.waiters
		}
		cache.mu.Unlock()
		if count == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("并发请求未进入同一个向量调用")
}

func TestKbQueryVectorCacheCancelsProviderWhenAllWaitersLeave(t *testing.T) {
	started, cancelled := make(chan struct{}), make(chan struct{})
	embedder, ai := queryVectorFixture(t, func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}, nil)
	cache := &queryVectorMemoryCache{}
	load := newKbQueryVectorCache(cache, embedder, ai)
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := load.Embed(ctx, 7, "cancel"); result <- err }()
	<-started
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("全部等待者离开后仍占用供应商连接")
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.writes != 0 {
		t.Fatal("取消请求不能写入缓存")
	}
}

func TestKbQueryVectorCacheDoesNotCacheOrRetryProviderFailure(t *testing.T) {
	var calls atomic.Int32
	embedder, ai := queryVectorFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}, nil)
	load := newKbQueryVectorCache(&queryVectorMemoryCache{}, embedder, ai)
	for range 2 {
		if _, err := load.Embed(context.Background(), 7, "failure"); err == nil {
			t.Fatal("供应商失败不能伪装成功")
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("每个显式请求只能尝试一次，不得缓存错误: %d", calls.Load())
	}
}

type queryVectorSearchStore struct {
	vectordb.Store
	readyErr error
	calls    int
}

func (s *queryVectorSearchStore) Ready(context.Context) error { return s.readyErr }
func (s *queryVectorSearchStore) Search(context.Context, []float32, int64, int64, int) ([]vectordb.Hit, error) {
	s.calls++
	return []vectordb.Hit{{ID: 1, Score: .8}}, nil
}

func TestKbSearcherCachedVectorStillChecksReadinessAndFactSource(t *testing.T) {
	var calls atomic.Int32
	embedder, ai := queryVectorFixture(t, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); queryVectorReply(w) }, nil)
	store := &queryVectorSearchStore{}
	chunks := &coverageChunks{candidates: []coverageCandidate{{id: 1, kb: 3, doc: 4, score: .8}}}
	searcher := &kbSearcher{store: store, embedder: embedder, chunks: chunks, snippetBudget: 1000, queryVectors: newKbQueryVectorCache(&queryVectorMemoryCache{}, embedder, ai)}
	for range 2 {
		if hits, err := searcher.Search(context.Background(), 7, 3, "same"); err != nil || len(hits) != 1 {
			t.Fatal(hits, err)
		}
	}
	if calls.Load() != 1 || store.calls != 2 || chunks.calls != 2 {
		t.Fatal("只允许省查询向量化，不能省索引查询或事实源复核")
	}
	chunks.candidates[0].filtered = true
	if hits, err := searcher.Search(context.Background(), 7, 3, "same"); err != nil || len(hits) != 0 {
		t.Fatal("缓存不得恢复已软删或失效片段", hits, err)
	}
	store.readyErr = errors.New("index unavailable")
	if _, err := searcher.Search(context.Background(), 7, 3, "same"); err == nil {
		t.Fatal("命中缓存仍必须先核对索引就绪")
	}
	if calls.Load() != 1 || store.calls != 3 || chunks.calls != 3 {
		t.Fatal("未就绪请求不应继续执行检索")
	}
}

func TestKbQueryVectorCachePayloadContainsOnlyVector(t *testing.T) {
	embedder, ai := queryVectorFixture(t, func(w http.ResponseWriter, _ *http.Request) { queryVectorReply(w) }, nil)
	cache := &queryVectorMemoryCache{}
	if _, err := newKbQueryVectorCache(cache, embedder, ai).Embed(context.Background(), 7, "private query"); err != nil {
		t.Fatal(err)
	}
	for _, raw := range cache.values {
		var got []float32
		if json.Unmarshal([]byte(raw), &got) != nil || len(got) != 2 {
			t.Fatal("缓存只保存数值向量")
		}
	}
}

type queryVectorDelayedReadKey struct{}

type queryVectorDelayedMiss struct {
	queryVectorMemoryCache
	blocked chan struct{}
	release chan struct{}
	delayed atomic.Bool
}

func (c *queryVectorDelayedMiss) GetCtx(ctx context.Context, key string) (string, error) {
	raw, err := c.queryVectorMemoryCache.GetCtx(ctx, key)
	if ctx.Value(queryVectorDelayedReadKey{}) != nil && c.delayed.CompareAndSwap(false, true) {
		close(c.blocked)
		select {
		case <-c.release:
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	return raw, err
}

func TestKbQueryVectorCacheRechecksDelayedMissBeforeStartingAnotherProviderCall(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	embedder, ai := queryVectorFixture(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			close(started)
			<-release
		}
		queryVectorReply(w)
	}, nil)
	cache := &queryVectorDelayedMiss{blocked: make(chan struct{}), release: make(chan struct{})}
	load := newKbQueryVectorCache(cache, embedder, ai)
	first, second := make(chan error, 1), make(chan error, 1)
	go func() { _, err := load.Embed(context.Background(), 7, "same"); first <- err }()
	<-started
	go func() {
		ctx := context.WithValue(context.Background(), queryVectorDelayedReadKey{}, true)
		_, err := load.Embed(ctx, 7, "same")
		second <- err
	}()
	<-cache.blocked
	close(release)
	if err := <-first; err != nil {
		t.Fatal(err)
	}
	close(cache.release)
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatalf("先前flight已写缓存，迟到miss不能再发供应商调用: %d", calls.Load())
	}
}

func TestKbQueryVectorCacheRejectsMismatchedUsageOwnerEvenOnCacheHit(t *testing.T) {
	var calls atomic.Int32
	embedder, ai := queryVectorFixture(t, func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); queryVectorReply(w) }, nil)
	load := newKbQueryVectorCache(&queryVectorMemoryCache{}, embedder, ai)
	if _, err := load.Embed(context.Background(), 7, "same"); err != nil {
		t.Fatal(err)
	}
	ctx := llm.WithUsageIdentity(context.Background(), llm.UsageIdentity{UserID: 8, Source: "kb_search", SourceID: 3, StepIndex: -1})
	if _, err := load.Embed(ctx, 7, "same"); err == nil {
		t.Fatal("计量身份不能被另一归属的缓存键替代")
	}
	if calls.Load() != 1 {
		t.Fatal("身份不一致不能触发供应商调用")
	}
}
