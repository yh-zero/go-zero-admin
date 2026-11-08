package svc

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/pkg/vectordb"
)

const kbIndexReconcileBatchSize = 500
const kbIndexReconcileTimeout = 120 * time.Second

// StartKbIndexReconciler 只清理事实源已无效的索引，不补向量或重放Embedding。
// 同集合实例通过短租约减少重复扫描，失败残留由下一周期重新读取实际索引处理。
func (s *ServiceContext) StartKbIndexReconciler(parent context.Context) func() {
	return s.startKbIndexReconciler(parent, 5*time.Second, 5*time.Minute)
}

func (s *ServiceContext) startKbIndexReconciler(parent context.Context, initial, interval time.Duration) func() {
	if s.Redis == nil || s.VectorStore == nil || s.AiKbChunkModel == nil {
		return func() {}
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		timer := time.NewTimer(initial)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				s.runKbIndexReconcile(ctx)
				timer.Reset(interval)
			}
		}
	}()
	return func() { cancel(); <-done }
}

func (s *ServiceContext) runKbIndexReconcile(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, kbIndexReconcileTimeout)
	defer cancel()
	lock := redis.NewRedisLock(s.Redis, s.kbIndexReconcileLeaseKey())
	// 比单轮上下文长30秒；进程中断后仍有界释放，不依赖永久锁或自动重放。
	lock.SetExpire(150)
	locked, err := lock.AcquireCtx(ctx)
	if err != nil {
		logx.WithContext(ctx).Error("知识库索引对账租约不可用，本轮未扫描")
		return
	}
	if !locked {
		return
	}
	defer func() {
		cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer stop()
		if _, err := lock.ReleaseCtx(cleanup); err != nil {
			logx.WithContext(cleanup).Error("知识库索引对账租约未确认释放，将等待过期")
		}
	}()
	stats, err := reconcileKbIndex(ctx, s.VectorStore, s.AiKbChunkModel)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			message, reason := "知识库索引对账未完成，等待下一周期", "index_reconcile_failed"
			if errors.Is(err, vectordb.ErrReconcileLimit) {
				message, reason = "知识库索引对账超过扫描上限，请人工核对或重建集合", "index_limit_manual"
			}
			logx.WithContext(ctx).Errorw(message, logx.Field("reason", reason), logx.Field("scanned", stats.Scanned), logx.Field("deleted", stats.Deleted))
		}
		return
	}
	logx.WithContext(ctx).Infow("知识库索引对账完成", logx.Field("scanned", stats.Scanned), logx.Field("deleted", stats.Deleted))
}

func (s *ServiceContext) kbIndexReconcileLeaseKey() string {
	collection := vectordb.CollectionName(s.Config.AI.EmbeddingProvider, s.Config.AI.EmbeddingModel, s.Config.AI.EmbeddingDimensions)
	key := sha256.Sum256([]byte(strings.TrimSpace(s.Config.Qdrant.Address) + "\x00" + collection))
	return fmt.Sprintf("ai:kb:index:reconcile:%x", key)
}

type retainedIndexSource interface {
	FindRetainedIndexIDs(context.Context, []uint64) (map[uint64]struct{}, error)
}

type kbIndexReconcileStats struct {
	Scanned int
	Deleted int // 只累计成功确认的删除；失败请求可能已部分生效，由下轮实际快照核对。
}

func reconcileKbIndex(ctx context.Context, store vectordb.Store, facts retainedIndexSource) (kbIndexReconcileStats, error) {
	var stats kbIndexReconcileStats
	if err := ctx.Err(); err != nil {
		return stats, err
	}
	// 必须先完整读取索引；截断或中途失败的快照不能驱动任何删除。
	snapshot, err := store.AllIDs(ctx)
	if err != nil {
		return stats, fmt.Errorf("索引快照未完成: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return stats, err
	}
	stats.Scanned = len(snapshot)
	if stats.Scanned > 1_000_000 {
		return stats, vectordb.ErrReconcileLimit
	}
	verify := func(ids []uint64) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		retained, err := facts.FindRetainedIndexIDs(ctx, ids)
		if err != nil {
			return fmt.Errorf("索引事实源核对失败: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		orphaned := make([]uint64, 0, len(ids))
		for _, id := range ids {
			if _, keep := retained[id]; !keep {
				orphaned = append(orphaned, id)
			}
		}
		if len(orphaned) == 0 {
			return nil
		}
		if err := store.Delete(ctx, orphaned); err != nil {
			return fmt.Errorf("孤立索引清理未完成: %w", err)
		}
		stats.Deleted += len(orphaned)
		return nil
	}
	batch := make([]uint64, 0, kbIndexReconcileBatchSize)
	for id := range snapshot {
		batch = append(batch, id)
		if len(batch) == kbIndexReconcileBatchSize {
			if err := verify(batch); err != nil {
				return stats, err
			}
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		if err := verify(batch); err != nil {
			return stats, err
		}
	}
	if err := ctx.Err(); err != nil {
		return stats, err
	}
	return stats, nil
}
