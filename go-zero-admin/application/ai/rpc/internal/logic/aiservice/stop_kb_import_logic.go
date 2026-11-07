package aiservicelogic

import (
	"context"
	"go-zero-admin/application/ai/rpc/internal/model"
	"time"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type StopKbImportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopKbImportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopKbImportLogic {
	return &StopKbImportLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StopKbImportLogic) StopKbImport(in *aipb.StopKbImportRequest) (*aipb.GetKbImportResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	if err := validateKbRequestIds(in.BatchId, in.BatchId); err != nil {
		return nil, err
	}
	ids, err := l.svcCtx.KbIngestQueue.StopOwned(l.ctx, in.KbId, in.Actor.UserId, in.BatchId)
	if err != nil && !model.KbDocCommitted(err) {
		return nil, kbQueueError(err)
	}
	if len(ids) > 0 && l.svcCtx.VectorStore != nil {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(l.ctx), 5*time.Second)
		defer cancel()
		if err := l.svcCtx.VectorStore.Delete(ctx, ids); err != nil {
			l.Error("批次停止后的旧向量清理失败，等待对账")
		}
	}
	return NewGetKbImportLogic(l.ctx, l.svcCtx).GetKbImport(&aipb.GetKbImportRequest{Actor: in.Actor, KbId: in.KbId, BatchId: in.BatchId})
}
