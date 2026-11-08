package vectordb

import (
	"context"
	"time"
)

// RecoveringStore 保留已配置客户端；集合暂不可用时有限退避，后续请求重新探测。
// 不持有后台协程，单实例并发请求只进行一次初始化探测。
type RecoveringStore struct {
	Store
	gate      chan struct{}
	ready     bool
	lastError error
	retryAt   time.Time
	backoff   time.Duration
	now       func() time.Time
}

func NewRecoveringStore(store Store) *RecoveringStore {
	return &RecoveringStore{Store: store, gate: make(chan struct{}, 1), backoff: time.Second, now: time.Now}
}

// Ready 允许调用方在付费Embedding或领取任务前核对启动恢复状态。
func (s *RecoveringStore) Ready(ctx context.Context) error { return s.EnsureCollection(ctx) }

func (s *RecoveringStore) EnsureCollection(ctx context.Context) error {
	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.ready {
		return nil
	}
	if s.lastError != nil && s.now().Before(s.retryAt) {
		return s.lastError
	}
	err := s.Store.EnsureCollection(ctx)
	if err == nil {
		s.ready, s.lastError = true, nil
		s.backoff = time.Second
		return nil
	}
	s.lastError = err
	s.retryAt = s.now().Add(s.backoff)
	s.backoff = min(s.backoff*2, 30*time.Second)
	return err
}

func (s *RecoveringStore) Upsert(ctx context.Context, points []Point) error {
	if err := s.EnsureCollection(ctx); err != nil {
		return err
	}
	return s.Store.Upsert(ctx, points)
}

func (s *RecoveringStore) Search(ctx context.Context, vector []float32, ownerID, kbID int64, topK int) ([]Hit, error) {
	if err := s.EnsureCollection(ctx); err != nil {
		return nil, err
	}
	return s.Store.Search(ctx, vector, ownerID, kbID, topK)
}

func (s *RecoveringStore) Delete(ctx context.Context, ids []uint64) error {
	if err := s.EnsureCollection(ctx); err != nil {
		return err
	}
	return s.Store.Delete(ctx, ids)
}

func (s *RecoveringStore) DeleteByKb(ctx context.Context, kbID int64) error {
	if err := s.EnsureCollection(ctx); err != nil {
		return err
	}
	return s.Store.DeleteByKb(ctx, kbID)
}

func (s *RecoveringStore) AllIDs(ctx context.Context) (map[uint64]struct{}, error) {
	if err := s.EnsureCollection(ctx); err != nil {
		return nil, err
	}
	return s.Store.AllIDs(ctx)
}
