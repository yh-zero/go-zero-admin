package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type RetryKbDocLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRetryKbDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RetryKbDocLogic {
	return &RetryKbDocLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// RetryKbDoc 重试摄取（仅 failed/pending）：清旧分块与旧向量 → 重新分块入库 → 摄取
func (l *RetryKbDocLogic) RetryKbDoc(in *aipb.RetryKbDocRequest) (*aipb.RetryKbDocResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	// 归属/存在性校验先于就绪校验：404 语义不受降级影响
	doc, err := l.svcCtx.AiKbDocModel.FindOwned(l.ctx, in.KbId, in.DocId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	if err := requireKbReady(l.svcCtx); err != nil {
		return nil, err
	}
	if doc.Status != "failed" && doc.Status != "pending" {
		return nil, aiModelError(model.ErrKbDocState)
	}
	oldChunkIds, err := l.svcCtx.AiKbDocModel.ResetForRetry(l.ctx, in.KbId, in.DocId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	if l.svcCtx.VectorStore != nil && len(oldChunkIds) > 0 {
		if err := l.svcCtx.VectorStore.Delete(l.ctx, oldChunkIds); err != nil {
			l.Errorf("文档 %d 旧向量清除失败（等待对账收敛）: %v", in.DocId, err)
		}
	}
	chunks := chunkText(doc.Content)
	chunkIds, err := l.svcCtx.AiKbDocModel.ReinsertChunks(l.ctx, in.DocId, in.KbId, chunks)
	if err != nil {
		return nil, aiModelError(err)
	}
	if err := ingestChunks(l.ctx, l.svcCtx, in.DocId, in.KbId, in.Actor.UserId, chunks, chunkIds); err != nil {
		l.Errorf("文档 %d 重试摄取失败: %v", in.DocId, err)
	}
	row, err := l.svcCtx.AiKbDocModel.FindOwned(l.ctx, in.KbId, in.DocId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.RetryKbDocResponse{Doc: kbDocRowToPb(row)}, nil
}
