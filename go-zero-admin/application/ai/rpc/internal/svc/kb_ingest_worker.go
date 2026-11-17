package svc

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/vectordb"
)

const KbIngestTimeout = 90 * time.Second
const kbCleanupTimeout = 5 * time.Second

func (s *ServiceContext) kbIngestWakeup() chan struct{} {
	s.kbIngestNotifyOnce.Do(func() { s.kbIngestNotify = make(chan struct{}, 2) })
	return s.kbIngestNotify
}

// NotifyKbIngest 只提示本实例重新核对MySQL队列；跨实例或漏通知仍由有界轮询兜底。
func (s *ServiceContext) NotifyKbIngest() {
	wakeup := s.kbIngestWakeup()
	for range 2 {
		select {
		case wakeup <- struct{}{}:
		default:
		}
	}
}

// StartKbIngestWorkers 的两个执行槽从持久队列领取；请求取消不会终止已提交的任务。
// stop 取消本进程正在执行的任务并等待其安全终态，queued 保留供下次启动继续。
func (s *ServiceContext) StartKbIngestWorkers(parent context.Context) func() {
	ctx, cancel := context.WithCancel(parent)
	var wg sync.WaitGroup
	if s.KbIngestQueue != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			reapKbIngestPeriodically(ctx, 5*time.Second, s.reapKbIngest)
		}()
		wakeup := s.kbIngestWakeup()
		for range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				pollKbIngest(ctx, wakeup, func(ctx context.Context) bool {
					var attempt *model.KbIngestAttempt
					err := model.ErrNotFound
					if s.Embedder != nil && s.VectorStore != nil && checkKbVectorReady(ctx, s.VectorStore) == nil {
						attempt, err = s.KbIngestQueue.Claim(ctx, uuid.NewString())
					}
					if err == nil {
						s.runKbIngest(ctx, attempt)
						return true
					}
					if !errors.Is(err, model.ErrNotFound) && !errors.Is(err, context.Canceled) {
						logx.WithContext(ctx).Error("知识库后台任务领取失败")
					}
					return false
				})
			}()
		}
	}
	return func() { cancel(); wg.Wait() }
}

func pollKbIngest(ctx context.Context, wakeup <-chan struct{}, poll func(context.Context) bool) {
	delay := 250 * time.Millisecond
	for ctx.Err() == nil {
		if poll(ctx) {
			delay = 250 * time.Millisecond
			continue
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-wakeup:
			timer.Stop()
			delay = 250 * time.Millisecond
		case <-timer.C:
			delay = min(delay*2, 2*time.Second)
		}
	}
}

func reapKbIngestPeriodically(ctx context.Context, interval time.Duration, reap func(context.Context)) {
	for ctx.Err() == nil {
		// 最多20个过期作业：事实源5秒、每个索引删除5秒；整轮仍有界且不重叠。
		scan, cancel := context.WithTimeout(ctx, 2*time.Minute)
		reap(scan)
		cancel()
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (s *ServiceContext) reapKbIngest(ctx context.Context) {
	reapKbIngestAttempts(ctx, s.KbIngestQueue.Expired, s.VectorStore)
}

func reapKbIngestAttempts(ctx context.Context, expired func(context.Context) ([]*model.KbIngestAttempt, error), store vectordb.Store) {
	expiry, cancel := context.WithTimeout(ctx, kbCleanupTimeout)
	attempts, err := expired(expiry)
	cancel()
	if err != nil && !errors.Is(err, context.Canceled) {
		logx.WithContext(ctx).Error("知识库过期任务收敛失败")
	}
	for _, attempt := range attempts {
		if ctx.Err() != nil {
			return
		}
		if store == nil {
			continue
		}
		cleanup, cancel := context.WithTimeout(ctx, kbCleanupTimeout)
		// 进程可能在清理旧代次之前中断，恢复时一并补偿该作业接管的旧索引。
		ids := append([]uint64(nil), attempt.ChunkIds...)
		ids = append(ids, attempt.OldChunkIds...)
		if err := store.Delete(cleanup, ids); err != nil {
			logx.WithContext(cleanup).Errorf("知识库任务 %d 过期向量清理失败", attempt.Job.Id)
		}
		cancel()
	}
}

func (s *ServiceContext) runKbIngest(parent context.Context, a *model.KbIngestAttempt) {
	ctx, cancel := context.WithTimeout(parent, KbIngestTimeout)
	defer cancel()
	ctx = WithUsageSource(ctx, a.Job.OwnerId, "kb_ingest", a.Job.RequestId, a.Job.Id, -1)
	if len(a.OldChunkIds) > 0 {
		if err := s.VectorStore.Delete(ctx, a.OldChunkIds); err != nil {
			logx.WithContext(ctx).Errorf("知识库任务 %d 旧向量清理失败，等待对账", a.Job.Id)
		}
	}
	err := s.ingestKbChunks(ctx, a.Job.DocId, a.Job.KbId, a.Job.OwnerId, a.Chunks, a.ChunkIds,
		func(ctx context.Context) error { return s.KbIngestQueue.Transition(ctx, a, "indexing", "", "", false) },
		func(ctx context.Context) error {
			return s.KbIngestQueue.Transition(ctx, a, "succeeded", "", s.Config.AI.EmbeddingModel, false)
		},
		func(ctx context.Context, safeErr string) error {
			return s.KbIngestQueue.Transition(ctx, a, "failed", safeErr, "", false)
		},
	)
	if err != nil {
		logx.WithContext(ctx).Errorf("知识库后台任务 %d 未完成", a.Job.Id)
	}
}

// IngestKbChunks 保留同代次摄取能力，供专项回归及已存在的内部调用使用。
func (s *ServiceContext) IngestKbChunks(ctx context.Context, docId, kbId, ownerId int64, chunks []model.KbChunkRow, ids []uint64) error {
	return s.ingestKbChunks(ctx, docId, kbId, ownerId, chunks, ids,
		func(ctx context.Context) error { return s.AiKbDocModel.MarkIndexing(ctx, docId, ids) },
		func(ctx context.Context) error {
			return s.AiKbDocModel.MarkIngested(ctx, docId, ids, s.Config.AI.EmbeddingModel)
		},
		func(ctx context.Context, safeErr string) error {
			return s.AiKbDocModel.MarkFailed(ctx, docId, ids, safeErr)
		},
	)
}

func (s *ServiceContext) ingestKbChunks(ctx context.Context, docId, kbId, ownerId int64, chunks []model.KbChunkRow, ids []uint64, indexing, ready func(context.Context) error, failed func(context.Context, string) error) error {
	ctx = WithUsageSource(ctx, ownerId, "kb_ingest", "", docId, -1)
	fail := func(safeErr string) {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), kbCleanupTimeout)
		defer cancel()
		if err := failed(cleanup, safeErr); err != nil {
			logx.WithContext(cleanup).Errorf("文档 %d 失败终态或缓存未完成", docId)
		}
		if err := s.VectorStore.Delete(cleanup, ids); err != nil {
			logx.WithContext(cleanup).Errorf("文档 %d 本次向量清除失败，等待对账", docId)
		}
	}
	texts := make([]string, 0, len(chunks))
	for _, c := range chunks {
		texts = append(texts, c.Content)
	}
	vecs, err := s.Embedder.Embed(ctx, texts)
	if err != nil {
		fail("向量化失败，请核对后手动重试")
		return err
	}
	if err := ctx.Err(); err != nil {
		fail("文档摄取超时或取消，请核对后手动重试")
		return err
	}
	if len(vecs) != len(chunks) || len(ids) != len(chunks) {
		fail("向量化结果不完整，请手动重试")
		return fmt.Errorf("文档 %d 摄取分块数量不一致", docId)
	}
	if err := indexing(ctx); err != nil {
		if !model.KbDocCommitted(err) {
			fail("文档状态已变化，请刷新状态")
			return err
		}
		logx.WithContext(ctx).Errorf("文档 %d 索引阶段已提交但缓存未刷新", docId)
	}
	points := make([]vectordb.Point, 0, len(chunks))
	for i, c := range chunks {
		points = append(points, vectordb.Point{ID: ids[i], Vector: vecs[i], KbID: kbId, OwnerID: ownerId, DocID: docId, TokenCount: c.TokenCount})
	}
	if err := s.VectorStore.Upsert(ctx, points); err != nil {
		fail("向量索引写入失败，请手动重试")
		return err
	}
	if err := ready(ctx); err != nil {
		if !model.KbDocCommitted(err) {
			fail("文档摄取未完成，请核对后手动重试")
		}
		return err
	}
	return nil
}
