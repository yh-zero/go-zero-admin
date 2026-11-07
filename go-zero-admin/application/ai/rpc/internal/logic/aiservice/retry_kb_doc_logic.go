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

// RetryKbDoc 原子领取失败或过期尝试并替换分块，再摄取本次分块。
func (l *RetryKbDocLogic) RetryKbDoc(in *aipb.RetryKbDocRequest) (*aipb.RetryKbDocResponse, error) {
	ctx, cancel := context.WithTimeout(l.ctx, kbIngestTimeout)
	defer cancel()
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	// 归属/存在性校验先于就绪校验：404 语义不受降级影响
	doc, err := l.svcCtx.AiKbDocModel.FindOwned(ctx, in.KbId, in.DocId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	if err := requireKbReady(l.svcCtx); err != nil {
		return nil, err
	}
	if doc.Status != "failed" && doc.Status != "pending" {
		return nil, aiModelError(model.ErrKbDocState)
	}
	chunks := chunkText(doc.Content)
	oldChunkIds, chunkIds, err := l.svcCtx.AiKbDocModel.ReplaceChunksForRetry(ctx, in.KbId, in.DocId, in.Actor.UserId, chunks, doc.Version)
	if err != nil && !model.KbDocCommitted(err) {
		return nil, aiModelError(err)
	}
	if err != nil {
		l.Errorf("文档 %d 重试已提交但缓存失效失败: %v", in.DocId, err)
	}
	if l.svcCtx.VectorStore != nil && len(oldChunkIds) > 0 {
		if err := l.svcCtx.VectorStore.Delete(ctx, oldChunkIds); err != nil {
			l.Errorf("文档 %d 旧向量清除失败（等待对账收敛）: %v", in.DocId, err)
		}
	}
	if err := ingestChunks(ctx, l.svcCtx, in.DocId, in.KbId, in.Actor.UserId, chunks, chunkIds); err != nil {
		l.Errorf("文档 %d 重试摄取失败: %v", in.DocId, err)
	}
	row, err := l.svcCtx.AiKbDocModel.FindOwned(l.ctx, in.KbId, in.DocId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.RetryKbDocResponse{Doc: kbDocRowToPb(row)}, nil
}
