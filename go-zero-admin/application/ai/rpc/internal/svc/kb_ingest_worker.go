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

// StartKbIngestWorkers 的两个执行槽从持久队列领取；请求取消不会终止已提交的任务。
// stop 取消本进程正在执行的任务并等待其安全终态，queued 保留供下次启动继续。
func (s *ServiceContext) StartKbIngestWorkers(parent context.Context) func() {
	ctx, cancel := context.WithCancel(parent)
	var wg sync.WaitGroup
	if s.KbIngestQueue != nil {
		for range 2 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for ctx.Err() == nil {
					s.reapKbIngest(ctx)
					var attempt *model.KbIngestAttempt
					err := model.ErrNotFound
					if s.Embedder != nil && s.VectorStore != nil {
						attempt, err = s.KbIngestQueue.Claim(ctx, uuid.NewString())
					}
					if err == nil {
						s.runKbIngest(ctx, attempt)
						continue
					}
					if !errors.Is(err, model.ErrNotFound) && !errors.Is(err, context.Canceled) {
						logx.WithContext(ctx).Error("知识库后台任务领取失败")
					}
					timer := time.NewTimer(250 * time.Millisecond)
					select {
					case <-ctx.Done():
						timer.Stop()
					case <-timer.C:
					}
				}
			}()
		}
	}
	return func() { cancel(); wg.Wait() }
}

func (s *ServiceContext) reapKbIngest(ctx context.Context) {
	attempts, err := s.KbIngestQueue.Expired(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		logx.WithContext(ctx).Error("知识库过期任务收敛失败")
	}
	for _, attempt := range attempts {
		if s.VectorStore == nil {
			continue
		}
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), kbCleanupTimeout)
		// 进程可能在清理旧代次之前中断，恢复时一并补偿该作业接管的旧索引。
		ids := append([]uint64(nil), attempt.ChunkIds...)
		ids = append(ids, attempt.OldChunkIds...)
		if err := s.VectorStore.Delete(cleanup, ids); err != nil {
			logx.WithContext(cleanup).Errorf("知识库任务 %d 过期向量清理失败", attempt.Job.Id)
		}
		cancel()
	}
}

func (s *ServiceContext) runKbIngest(parent context.Context, a *model.KbIngestAttempt) {
	ctx, cancel := context.WithTimeout(parent, KbIngestTimeout)
	defer cancel()
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
