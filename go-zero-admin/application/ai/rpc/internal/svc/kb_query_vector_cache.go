package svc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"
	"time"

	"go-zero-admin/application/ai/rpc/internal/config"
	"go-zero-admin/pkg/llm"
)

type kbQueryVectorStore interface {
	GetCtx(context.Context, string) (string, error)
	SetexCtx(context.Context, string, string, int) error
}

// 只复用查询向量；语料变化、软删和归属仍由每次索引查询与MySQL复核处理。
type kbQueryVectorCache struct {
	store       kbQueryVectorStore
	embedder    *llm.Embedder
	fingerprint [32]byte
	mu          sync.Mutex
	flights     map[string]*kbQueryVectorFlight
}

type kbQueryVectorFlight struct {
	done     chan struct{}
	cancel   context.CancelFunc
	waiters  int
	finished bool
	vector   []float32
	err      error
}

func newKbQueryVectorCache(store kbQueryVectorStore, embedder *llm.Embedder, ai config.ConfigAI) *kbQueryVectorCache {
	// 不用密钥或正文作命名空间，端点、模型、维度必须随实际配置变化。
	identity, _ := json.Marshal([]any{ai.EmbeddingProvider, ai.EmbeddingBaseURL, ai.EmbeddingModel, ai.EmbeddingDimensions})
	return &kbQueryVectorCache{store: store, embedder: embedder, fingerprint: sha256.Sum256(identity), flights: make(map[string]*kbQueryVectorFlight)}
}

func (c *kbQueryVectorCache) Embed(ctx context.Context, ownerID int64, query string) ([]float32, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if ownerID <= 0 || c.embedder == nil {
		return nil, errors.New("查询向量缺少有效归属或客户端")
	}
	if identity, ok := llm.UsageIdentityFrom(ctx); ok && identity.UserID != ownerID {
		return nil, errors.New("查询向量计量身份与归属不一致")
	}
	key := fmt.Sprintf("ai:kb:query-vector:v1:%d:%x:%x", ownerID, c.fingerprint, sha256.Sum256([]byte(query)))
	if vector, ok := c.cachedVector(ctx, key); ok {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return vector, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	flight := c.flights[key]
	if flight == nil {
		// 保留发起者的计量身份；任一等待者取消不终止其他请求，全部离开才取消外呼。
		callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 90*time.Second)
		flight = &kbQueryVectorFlight{done: make(chan struct{}), cancel: cancel}
		c.flights[key] = flight
		go c.fill(callCtx, key, query, flight)
	}
	flight.waiters++
	c.mu.Unlock()
	defer c.leave(key, flight)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-flight.done:
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return slices.Clone(flight.vector), flight.err
	}
}

func (c *kbQueryVectorCache) fill(ctx context.Context, key, query string, flight *kbQueryVectorFlight) {
	defer flight.cancel()
	var vector []float32
	var err error
	// 首次读缓存到进入flight期间可能已有另一个调用完成，冷miss再读一次避免重复计费。
	cached, ok := c.cachedVector(ctx, key)
	if ok {
		vector = cached
	} else {
		var vectors [][]float32
		vectors, err = c.embedder.Embed(ctx, []string{query})
		if err == nil {
			if len(vectors) != 1 || !validKbQueryVector(vectors[0], c.embedder.Dims()) {
				err = errors.New("查询向量返回数量、维度或数值无效")
			} else {
				vector = vectors[0]
			}
		}
	}
	if err == nil {
		err = ctx.Err()
	}
	if err == nil && !ok && c.store != nil {
		if raw, encodeErr := json.Marshal(vector); encodeErr == nil {
			writeCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
			// 缓存是可选加速；读写错误不改变已取得的有效向量，也不记录动态键或查询。
			_ = c.store.SetexCtx(writeCtx, key, string(raw), 300)
			cancel()
		}
	}
	c.mu.Lock()
	flight.vector, flight.err, flight.finished = vector, err, true
	close(flight.done)
	if c.flights[key] == flight {
		delete(c.flights, key)
	}
	c.mu.Unlock()
}

func (c *kbQueryVectorCache) cachedVector(ctx context.Context, key string) ([]float32, bool) {
	if c.store == nil {
		return nil, false
	}
	readCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()
	raw, err := c.store.GetCtx(readCtx, key)
	// null解到float32会被静默置零，用指针保留缺失元素并明确拒绝。
	var values []*float32
	if err != nil || len(raw) > c.embedder.Dims()*32+2 || json.Unmarshal([]byte(raw), &values) != nil || len(values) != c.embedder.Dims() {
		return nil, false
	}
	vector := make([]float32, len(values))
	for i, value := range values {
		if value == nil {
			return nil, false
		}
		vector[i] = *value
	}
	if validKbQueryVector(vector, c.embedder.Dims()) {
		return vector, true
	}
	return nil, false
}

func (c *kbQueryVectorCache) leave(key string, flight *kbQueryVectorFlight) {
	c.mu.Lock()
	defer c.mu.Unlock()
	flight.waiters--
	if flight.waiters == 0 && !flight.finished {
		if c.flights[key] == flight {
			delete(c.flights, key)
		}
		flight.cancel()
	}
}

func validKbQueryVector(vector []float32, dimensions int) bool {
	if len(vector) != dimensions {
		return false
	}
	nonzero := false
	for _, value := range vector {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return false
		}
		nonzero = nonzero || value != 0
	}
	return nonzero
}
