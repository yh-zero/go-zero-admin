package svc

import (
	"context"
	"slices"
	"sync/atomic"
	"testing"
	"time"

	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/vectordb"
)

func TestKbIngestPollBacksOffAndWakeupClaimsPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wake := make(chan struct{}, 2)
	polled := make(chan time.Time, 16)
	done := make(chan struct{})
	go func() {
		defer close(done)
		pollKbIngest(ctx, wake, func(context.Context) bool { polled <- time.Now(); return false })
	}()
	first := <-polled
	second := <-polled
	third := <-polled
	if second.Sub(first) < 200*time.Millisecond || third.Sub(second) < 400*time.Millisecond {
		t.Fatal("空队列必须逐步退避，不能持续250ms轮询")
	}
	wake <- struct{}{}
	select {
	case <-polled:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("新提交作业的本实例唤醒未及时领取")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("停机必须取消退避等待")
	}
}

func TestKbIngestReaperIsSingleAndBounded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var active, maxActive atomic.Int32
	seen := make(chan struct{}, 4)
	done := make(chan struct{})
	go func() {
		defer close(done)
		reapKbIngestPeriodically(ctx, 20*time.Millisecond, func(call context.Context) {
			n := active.Add(1)
			for old := maxActive.Load(); n > old && !maxActive.CompareAndSwap(old, n); old = maxActive.Load() {
			}
			if _, ok := call.Deadline(); !ok {
				t.Error("单次过期收敛必须有界")
			}
			seen <- struct{}{}
			select {
			case <-call.Done():
			case <-time.After(30 * time.Millisecond):
			}
			active.Add(-1)
		})
	}()
	<-seen
	<-seen
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("reaper停止未取消在途扫描")
	}
	if maxActive.Load() != 1 {
		t.Fatal("定时扫描不可重叠或由两个worker重复发起")
	}
}

func TestNotifyKbIngestIsNonblockingAndWakesBothSlots(t *testing.T) {
	s := &ServiceContext{}
	s.NotifyKbIngest()
	for range 10 {
		s.NotifyKbIngest()
	}
	for range 2 {
		select {
		case <-s.kbIngestWakeup():
		default:
			t.Fatal("应为两个worker保留唤醒信号")
		}
	}
}

type expiredIngestVectorStore struct {
	vectordb.Store
	deleted [][]uint64
	started chan struct{}
}

func (s *expiredIngestVectorStore) Delete(ctx context.Context, ids []uint64) error {
	s.deleted = append(s.deleted, slices.Clone(ids))
	if len(s.deleted) == 1 {
		if s.started != nil {
			close(s.started)
		}
		<-ctx.Done()
		return ctx.Err()
	}
	return nil
}

func expiredIngestAttempts(context.Context) ([]*model.KbIngestAttempt, error) {
	return []*model.KbIngestAttempt{
		{Job: &model.KbIngestJobRow{Id: 1}, ChunkIds: []uint64{11}, OldChunkIds: []uint64{12}},
		{Job: &model.KbIngestJobRow{Id: 2}, ChunkIds: []uint64{21}, OldChunkIds: []uint64{22}},
		{Job: &model.KbIngestJobRow{Id: 3}, ChunkIds: []uint64{31}, OldChunkIds: []uint64{32}},
	}, nil
}

func TestKbIngestReaperContinuesOtherFailedAttemptsAfterOneVectorDeleteTimesOut(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	store := &expiredIngestVectorStore{}
	reapKbIngestPeriodically(ctx, time.Hour, func(scan context.Context) {
		reapKbIngestAttempts(scan, func(expiry context.Context) ([]*model.KbIngestAttempt, error) {
			deadline, ok := expiry.Deadline()
			if !ok || time.Until(deadline) > 5100*time.Millisecond {
				t.Error("Expired事实源阶段必须单独有界5秒")
			}
			return expiredIngestAttempts(expiry)
		}, store)
		cancel()
	})
	want := [][]uint64{{11, 12}, {21, 22}, {31, 32}}
	if !slices.EqualFunc(store.deleted, want, slices.Equal[[]uint64]) {
		t.Fatalf("首个向量删除超时不能跳过已置failed的其他代次: deleted=%v want=%v", store.deleted, want)
	}
}

func TestKbIngestReaperParentCancellationStopsVectorCleanupPromptly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := &expiredIngestVectorStore{started: make(chan struct{})}
	done := make(chan struct{})
	go func() {
		defer close(done)
		reapKbIngestPeriodically(ctx, time.Hour, func(scan context.Context) { reapKbIngestAttempts(scan, expiredIngestAttempts, store) })
	}()
	select {
	case <-store.started:
	case <-time.After(time.Second):
		t.Fatal("未开始过期索引清理")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("停机取消后向量删除未及时停止")
	}
	if len(store.deleted) != 1 {
		t.Fatal("停机取消应停止后续向量请求")
	}
}
