package svc

import (
	"context"
	"errors"
	"slices"
	"testing"

	"go-zero-admin/pkg/vectordb"
)

type reconcileVectorStore struct {
	vectordb.Store
	ids          map[uint64]struct{}
	snapshotErr  error
	deleteErr    error
	snapshotDone bool
	snapshots    int
	deletes      [][]uint64
	maxBatch     int
	onSnapshot   func()
	onDelete     func()
}

func (s *reconcileVectorStore) AllIDs(ctx context.Context) (map[uint64]struct{}, error) {
	s.snapshots++
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make(map[uint64]struct{}, len(s.ids))
	for id := range s.ids {
		out[id] = struct{}{}
	}
	s.snapshotDone = true
	if s.onSnapshot != nil {
		s.onSnapshot()
	}
	return out, s.snapshotErr
}

func (s *reconcileVectorStore) Delete(ctx context.Context, ids []uint64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.deletes = append(s.deletes, append([]uint64(nil), ids...))
	s.maxBatch = max(s.maxBatch, len(ids))
	if s.deleteErr != nil {
		return s.deleteErr
	}
	for _, id := range ids {
		delete(s.ids, id)
	}
	if s.onDelete != nil {
		s.onDelete()
	}
	return nil
}

type reconcileFactSource struct {
	store        *reconcileVectorStore
	keep         map[uint64]struct{}
	err          error
	queries      int
	maxBatch     int
	onRead       func()
	ignoreCancel bool
}

func (s *reconcileFactSource) FindRetainedIndexIDs(ctx context.Context, ids []uint64) (map[uint64]struct{}, error) {
	if !s.store.snapshotDone {
		panic("事实源不得先于完整索引快照读取")
	}
	s.queries++
	s.maxBatch = max(s.maxBatch, len(ids))
	if s.onRead != nil {
		s.onRead()
	}
	if err := ctx.Err(); err != nil && !s.ignoreCancel {
		return nil, err
	}
	kept := map[uint64]struct{}{}
	for _, id := range ids {
		if _, ok := s.keep[id]; ok {
			kept[id] = struct{}{}
		}
	}
	return kept, s.err
}

func TestKbIndexReconcileKeepsPendingAndReadyWithBoundedBatches(t *testing.T) {
	store := &reconcileVectorStore{ids: map[uint64]struct{}{}}
	facts := &reconcileFactSource{store: store, keep: map[uint64]struct{}{}}
	for id := uint64(1); id <= 1201; id++ {
		store.ids[id] = struct{}{}
		if id <= 601 {
			// 保留集合包含 pending/indexing 和 ready；具体状态过滤另由真实SQL回归覆盖。
			facts.keep[id] = struct{}{}
		}
	}
	stats, err := reconcileKbIndex(context.Background(), store, facts)
	if err != nil || stats.Scanned != 1201 || stats.Deleted != 600 || facts.queries != 3 || facts.maxBatch > 500 || store.maxBatch > 500 || store.snapshots != 1 {
		t.Fatalf("每轮应一次快照、3次有界SQL并仅删600孤立ID: stats=%+v queries=%d max=%d/%d err=%v", stats, facts.queries, facts.maxBatch, store.maxBatch, err)
	}
	if len(store.ids) != 601 {
		t.Fatal("未保留全部正在摄取及已就绪索引")
	}
	for id := range facts.keep {
		if _, ok := store.ids[id]; !ok {
			t.Fatalf("仍有效索引被误删: %d", id)
		}
	}
}

func TestKbIndexReconcileSnapshotOrSQLFailureNeverMeansEmptyFacts(t *testing.T) {
	broken := errors.New("local failure")
	for _, where := range []string{"snapshot", "sql"} {
		t.Run(where, func(t *testing.T) {
			store := &reconcileVectorStore{ids: map[uint64]struct{}{1: {}, 2: {}}}
			facts := &reconcileFactSource{store: store}
			if where == "snapshot" {
				store.snapshotErr = broken
			} else {
				facts.err = broken
			}
			_, err := reconcileKbIndex(context.Background(), store, facts)
			if !errors.Is(err, broken) || len(store.deletes) != 0 || len(store.ids) != 2 || where == "snapshot" && facts.queries != 0 {
				t.Fatal("部分索引快照或SQL失败不得据空集合删除任何ID")
			}
		})
	}
}

func TestKbIndexReconcileDeleteFailureIsRetriedFromNextActualSnapshot(t *testing.T) {
	broken := errors.New("local delete failure")
	store := &reconcileVectorStore{ids: map[uint64]struct{}{1: {}, 2: {}, 3: {}}, deleteErr: broken}
	facts := &reconcileFactSource{store: store, keep: map[uint64]struct{}{2: {}}}
	if _, err := reconcileKbIndex(context.Background(), store, facts); !errors.Is(err, broken) || len(store.ids) != 3 {
		t.Fatal("失败删除必须留待下轮，不能视为已完成")
	}
	store.deleteErr = nil
	stats, err := reconcileKbIndex(context.Background(), store, facts)
	if err != nil || stats.Deleted != 2 || store.snapshots != 2 || len(store.ids) != 1 || len(store.deletes) != 2 {
		t.Fatalf("后续周期必须重新读取实际索引并补偿: %+v err=%v", stats, err)
	}
	if _, ok := store.ids[2]; !ok {
		t.Fatal("保留索引被重试误删")
	}
	first, second := append([]uint64(nil), store.deletes[0]...), append([]uint64(nil), store.deletes[1]...)
	slices.Sort(first)
	slices.Sort(second)
	if !slices.Equal(first, []uint64{1, 3}) || !slices.Equal(second, []uint64{1, 3}) {
		t.Fatal("补偿只可删除事实源确认的孤立ID")
	}
}

func TestKbIndexReconcileCancellationStopsBeforeDelete(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := &reconcileVectorStore{ids: map[uint64]struct{}{1: {}}}
	facts := &reconcileFactSource{store: store, onRead: cancel, ignoreCancel: true}
	if _, err := reconcileKbIndex(ctx, store, facts); !errors.Is(err, context.Canceled) || len(store.deletes) != 0 {
		t.Fatal("取消后的事实源结果不得触发删除")
	}
}

func TestKbIndexReconcileCancellationAfterEmptySnapshotIsNotSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := &reconcileVectorStore{ids: map[uint64]struct{}{}, onSnapshot: cancel}
	facts := &reconcileFactSource{store: store}
	if _, err := reconcileKbIndex(ctx, store, facts); !errors.Is(err, context.Canceled) || facts.queries != 0 || len(store.deletes) != 0 {
		t.Fatal("快照返回时取消，不能报告本轮成功或继续操作")
	}
}

func TestKbIndexReconcileCancellationAfterLastDeleteIsNotSuccess(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := &reconcileVectorStore{ids: map[uint64]struct{}{1: {}}, onDelete: cancel}
	facts := &reconcileFactSource{store: store}
	stats, err := reconcileKbIndex(ctx, store, facts)
	if !errors.Is(err, context.Canceled) || stats.Deleted != 1 || len(store.deletes) != 1 {
		t.Fatal("最后删除响应后取消也须报未完成，保留已经确认的删除数")
	}
}
