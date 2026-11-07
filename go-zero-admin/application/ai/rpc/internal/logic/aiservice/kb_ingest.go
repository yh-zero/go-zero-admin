package aiservicelogic

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/vectordb"
	"go-zero-admin/pkg/xerr"
)

// 知识库摄取纪律（契约：AI 知识库与向量检索）：
// MySQL 事务先提交（pending）→ embed → Qdrant upsert → 置 ready；
// 任一步失败置 failed + 安全错误，可整体重试；禁止先写向量索引。

// kbContentMaxBytes 契约上限：内联文档 content ≤ 64 KiB
const kbContentMaxBytes = 64 * 1024

const kbIngestTimeout = 90 * time.Second
const kbCleanupTimeout = 5 * time.Second

// ingestChunks 对已完成入库（pending）的分块执行 embed→索引→置终态。
// 失败时条件写入 failed 并补偿本次向量；代次已变化或资源已删除时保留当前状态。
// 错误仅用于日志，不回显外部细节。
func ingestChunks(ctx context.Context, svcCtx *svc.ServiceContext, docId, kbId, ownerId int64, chunks []model.KbChunkRow, chunkIds []uint64) error {
	texts := make([]string, 0, len(chunks))
	for _, c := range chunks {
		texts = append(texts, c.Content)
	}
	vecs, err := svcCtx.Embedder.Embed(ctx, texts)
	if err != nil {
		failKbAttempt(ctx, svcCtx, docId, chunkIds, "向量化失败，请稍后重试")
		return err
	}
	if err := ctx.Err(); err != nil {
		failKbAttempt(ctx, svcCtx, docId, chunkIds, "文档摄取超时或取消，请重试")
		return err
	}
	if len(vecs) != len(chunks) || len(chunkIds) != len(chunks) {
		failKbAttempt(ctx, svcCtx, docId, chunkIds, "向量化结果不完整，请重试")
		return fmt.Errorf("文档 %d 摄取分块数量不一致", docId)
	}
	if err := svcCtx.AiKbDocModel.MarkIndexing(ctx, docId, chunkIds); err != nil {
		if !model.KbDocCommitted(err) {
			failKbAttempt(ctx, svcCtx, docId, chunkIds, "文档状态已变化，请刷新状态")
			return err
		}
		logx.WithContext(ctx).Errorf("文档 %d 索引阶段已提交但缓存失效失败: %v", docId, err)
	}
	points := make([]vectordb.Point, 0, len(chunks))
	for i, c := range chunks {
		points = append(points, vectordb.Point{
			ID:         chunkIds[i],
			Vector:     vecs[i],
			KbID:       kbId,
			OwnerID:    ownerId,
			DocID:      docId,
			TokenCount: c.TokenCount,
		})
	}
	if err := svcCtx.VectorStore.Upsert(ctx, points); err != nil {
		failKbAttempt(ctx, svcCtx, docId, chunkIds, "向量索引写入失败，请稍后重试")
		return err
	}
	if err := svcCtx.AiKbDocModel.MarkIngested(ctx, docId, chunkIds, svcCtx.Config.AI.EmbeddingModel); err != nil {
		// ready 已提交但缓存删除失败时，保留本次有效向量，向调用方明确报告缓存错误。
		if !model.KbDocCommitted(err) {
			failKbAttempt(ctx, svcCtx, docId, chunkIds, "文档摄取未完成，请重试")
		}
		return err
	}
	return nil
}

func failKbAttempt(ctx context.Context, svcCtx *svc.ServiceContext, docId int64, chunkIds []uint64, safeErr string) {
	// 请求取消后仍需写入失败终态；独立短截止保证清理不会无限延长同步请求。
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), kbCleanupTimeout)
	defer cancel()
	if err := svcCtx.AiKbDocModel.MarkFailed(cleanupCtx, docId, chunkIds, safeErr); err != nil {
		logx.WithContext(cleanupCtx).Errorf("文档 %d 摄取失败终态未写入或缓存失效失败: %v", docId, err)
	}
	// ID 是本次尝试的唯一代次，即使新尝试已经启动，也仅补偿本次向量。
	if err := svcCtx.VectorStore.Delete(cleanupCtx, chunkIds); err != nil {
		logx.WithContext(cleanupCtx).Errorf("文档 %d 本次失败向量清除失败（等待对账收敛）: %v", docId, err)
	}
}

// requireKbReady 校验知识库检索依赖就绪（Embedding + Qdrant），未配置按契约返回 503 降级
func requireKbReady(svcCtx *svc.ServiceContext) error {
	if svcCtx.Embedder == nil || svcCtx.VectorStore == nil {
		return xerr.NewErrCodeMsg(503, "知识库检索未配置（缺少 Embedding 密钥或向量索引），管理功能不受影响")
	}
	return nil
}
