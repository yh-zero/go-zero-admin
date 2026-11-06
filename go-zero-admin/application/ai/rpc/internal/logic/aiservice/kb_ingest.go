package aiservicelogic

import (
	"context"

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

// ingestChunks 对已完成入库（pending）的分块执行 embed→索引→置终态。
// 返回 nil 表示成功；失败时文档已被标记 failed，错误仅用于日志，不回显外部细节。
func ingestChunks(ctx context.Context, svcCtx *svc.ServiceContext, docId, kbId, ownerId int64, chunks []model.KbChunkRow, chunkIds []uint64) error {
	texts := make([]string, 0, len(chunks))
	for _, c := range chunks {
		texts = append(texts, c.Content)
	}
	vecs, err := svcCtx.Embedder.Embed(ctx, texts)
	if err != nil {
		if markErr := svcCtx.AiKbDocModel.MarkFailed(ctx, docId, "向量化失败，请稍后重试"); markErr != nil {
			logx.WithContext(ctx).Errorf("文档 %d 摄取失败且终态标记失败: %v", docId, markErr)
		}
		return err
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
		if markErr := svcCtx.AiKbDocModel.MarkFailed(ctx, docId, "向量索引写入失败，请稍后重试"); markErr != nil {
			logx.WithContext(ctx).Errorf("文档 %d 索引写入失败且终态标记失败: %v", docId, markErr)
		}
		return err
	}
	if err := svcCtx.AiKbDocModel.MarkIngested(ctx, docId, chunkIds, svcCtx.Config.AI.EmbeddingModel); err != nil {
		return err
	}
	return nil
}

// requireKbReady 校验知识库检索依赖就绪（Embedding + Qdrant），未配置按契约返回 503 降级
func requireKbReady(svcCtx *svc.ServiceContext) error {
	if svcCtx.Embedder == nil || svcCtx.VectorStore == nil {
		return xerr.NewErrCodeMsg(503, "知识库检索未配置（缺少 Embedding 密钥或向量索引），管理功能不受影响")
	}
	return nil
}
