package svc

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/application/ai/rpc/internal/config"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/vectordb"
)

type reconcileWorkerFacts struct{ model.AiKbChunkModel }

func (reconcileWorkerFacts) FindRetainedIndexIDs(context.Context, []uint64) (map[uint64]struct{}, error) {
	panic("空索引快照不得读取事实源")
}

type reconcileWorkerStore struct {
	vectordb.Store
	entered chan context.Context
	block   bool
	calls   atomic.Int32
}

func (s *reconcileWorkerStore) AllIDs(ctx context.Context) (map[uint64]struct{}, error) {
	s.calls.Add(1)
	select {
	case s.entered <- ctx:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	if s.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return map[uint64]struct{}{}, nil
}

func reconcileWorkerContext(t *testing.T, store *reconcileWorkerStore) *ServiceContext {
	t.Helper()
	address := os.Getenv("GOZERO_AI_TEST_REDIS")
	if address == "" {
		t.Skip("未设置本轮隔离Redis，未执行后台对账租约验证")
	}
	rdb, err := redis.NewRedis(redis.RedisConf{Host: address, Type: "node"})
	if err != nil || !rdb.PingCtx(context.Background()) {
		t.Fatal("本轮隔离Redis不可达")
	}
	return &ServiceContext{Config: config.Config{AI: config.ConfigAI{EmbeddingProvider: "test", EmbeddingModel: uuid.NewString(), EmbeddingDimensions: 2}}, Redis: rdb, VectorStore: store, AiKbChunkModel: reconcileWorkerFacts{}}
}

func awaitReconcileEntry(t *testing.T, entered <-chan context.Context) context.Context {
	t.Helper()
	select {
	case ctx := <-entered:
		return ctx
	case <-time.After(time.Second):
		t.Fatal("后台对账未开始")
		return nil
	}
}

func TestKbIndexReconcilerUsesSharedLeaseAndCancelsOnStop(t *testing.T) {
	store := &reconcileWorkerStore{entered: make(chan context.Context, 2), block: true}
	first := reconcileWorkerContext(t, store)
	second := *first
	first.Config.Qdrant.Address = " local-reconcile:6334 "
	second.Config.Qdrant.Address = "local-reconcile:6334"
	stop := first.startKbIndexReconciler(context.Background(), 0, time.Hour)
	t.Cleanup(stop)
	round := awaitReconcileEntry(t, store.entered)
	deadline, ok := round.Deadline()
	if !ok || time.Until(deadline) > 120*time.Second {
		t.Fatal("单轮必须有不超过120秒的超时")
	}
	probe, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	second.runKbIndexReconcile(probe)
	if store.calls.Load() != 1 {
		t.Fatal("同集合第二实例不得重复扫描活动租约")
	}
	stopped := make(chan struct{})
	go func() { stop(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("停止必须取消并等待本轮退出")
	}
	if round.Err() != context.Canceled {
		t.Fatal("停止后扫描上下文未取消")
	}
	key := first.kbIndexReconcileLeaseKey()
	if exists, err := first.Redis.ExistsCtx(context.Background(), key); err != nil || exists {
		t.Fatal("本轮租约未精确释放")
	}
}

func TestKbIndexReconcilerRunsLaterCyclesAndReleasesLease(t *testing.T) {
	store := &reconcileWorkerStore{entered: make(chan context.Context, 8)}
	service := reconcileWorkerContext(t, store)
	stop := service.startKbIndexReconciler(context.Background(), 0, 10*time.Millisecond)
	t.Cleanup(stop)
	awaitReconcileEntry(t, store.entered)
	awaitReconcileEntry(t, store.entered)
	stop()
	if store.calls.Load() < 2 {
		t.Fatal("后台任务必须在下一周期重新读取实际索引")
	}
	key := service.kbIndexReconcileLeaseKey()
	if exists, err := service.Redis.ExistsCtx(context.Background(), key); err != nil || exists {
		t.Fatal("周期结束后租约未释放")
	}
}

func TestKbIndexReconcilerLeaseFailureDoesNotScan(t *testing.T) {
	store := &reconcileWorkerStore{entered: make(chan context.Context, 1)}
	rdb, err := redis.NewRedis(redis.RedisConf{Host: "127.0.0.1:1", Type: "node", NonBlock: true})
	if err != nil {
		t.Fatal(err)
	}
	service := &ServiceContext{Redis: rdb, VectorStore: store, AiKbChunkModel: reconcileWorkerFacts{}}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	service.runKbIndexReconcile(ctx)
	if store.calls.Load() != 0 {
		t.Fatal("Redis租约不可用时不能扫描或删除索引")
	}
}

func TestKbIndexReconcilerLogsManualLimitWithoutVendorDetails(t *testing.T) {
	service := reconcileWorkerContext(t, &reconcileWorkerStore{entered: make(chan context.Context, 1)})
	store := &reconcileVectorStore{ids: map[uint64]struct{}{1: {}}, snapshotErr: fmt.Errorf("%w: PRIVATE_VENDOR_DETAIL", vectordb.ErrReconcileLimit)}
	service.VectorStore = store
	var captured bytes.Buffer
	previous := logx.Reset()
	logx.SetWriter(logx.NewWriter(&captured))
	t.Cleanup(func() { logx.SetWriter(previous) })
	service.runKbIndexReconcile(context.Background())
	output := captured.String()
	if !strings.Contains(output, "index_limit_manual") || !strings.Contains(output, "人工") || strings.Contains(output, "PRIVATE_") || len(store.deletes) != 0 {
		t.Fatalf("扫描超限必须明确人工处理且不泄露外部错误: %s", output)
	}
}
