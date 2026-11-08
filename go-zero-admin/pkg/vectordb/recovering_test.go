package vectordb

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type recoveryFixture struct {
	Store
	checks, searches int
	fail             bool
	check            func(context.Context) error
}

func (f *recoveryFixture) EnsureCollection(ctx context.Context) error {
	f.checks++
	if f.check != nil {
		return f.check(ctx)
	}
	if f.fail {
		return errors.New("temporary outage")
	}
	return nil
}

func TestRecoveringStoreWaitingCallerCanCancel(t *testing.T) {
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	f := &recoveryFixture{check: func(context.Context) error { close(started); <-release; return nil }}
	s := NewRecoveringStore(f)
	go func() { finished <- s.EnsureCollection(context.Background()) }()
	<-started
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	waiting := make(chan error, 1)
	go func() { waiting <- s.EnsureCollection(ctx) }()
	select {
	case err := <-waiting:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("等待期间应返回取消: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("取消请求不能等待其他初始化完成")
	}
	close(release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
}

func (f *recoveryFixture) Search(context.Context, []float32, int64, int64, int) ([]Hit, error) {
	f.searches++
	return []Hit{{ID: 7, Score: .8}}, nil
}

func TestRecoveringStoreBackoffAndRecovery(t *testing.T) {
	f := &recoveryFixture{fail: true}
	s := NewRecoveringStore(f)
	now := time.Unix(100, 0)
	s.now = func() time.Time { return now }
	if _, err := s.Search(context.Background(), nil, 1, 2, 5); err == nil {
		t.Fatal("故障应返回错误")
	}
	for range 20 {
		_ = s.EnsureCollection(context.Background())
	}
	if f.checks != 1 || f.searches != 0 {
		t.Fatal("退避期间不得反复探测或执行检索")
	}
	f.fail = false
	now = now.Add(time.Second)
	hits, err := s.Search(context.Background(), nil, 1, 2, 5)
	if err != nil || len(hits) != 1 || hits[0].ID != 7 || f.checks != 2 {
		t.Fatalf("恢复后应复用原调用: %v %v", hits, err)
	}
	_ = s.EnsureCollection(context.Background())
	if f.checks != 2 {
		t.Fatal("已初始化不应重复建集合")
	}
}

func TestRecoveringStoreConcurrentInitialization(t *testing.T) {
	f := &recoveryFixture{}
	s := NewRecoveringStore(f)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() {
			if err := s.EnsureCollection(context.Background()); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if f.checks != 1 {
		t.Fatalf("并发初始化次数=%d", f.checks)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(s.EnsureCollection(ctx), context.Canceled) {
		t.Fatal("必须保留调用取消")
	}
}
