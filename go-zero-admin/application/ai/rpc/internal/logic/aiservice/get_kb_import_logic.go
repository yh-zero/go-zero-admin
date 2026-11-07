package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetKbImportLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetKbImportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKbImportLogic {
	return &GetKbImportLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetKbImportLogic) GetKbImport(in *aipb.GetKbImportRequest) (*aipb.GetKbImportResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	if err := validateKbRequestIds(in.BatchId, in.BatchId); err != nil {
		return nil, err
	}
	batch, jobs, err := l.svcCtx.KbIngestQueue.GetOwned(l.ctx, in.KbId, in.Actor.UserId, in.BatchId)
	if err != nil {
		return nil, kbQueueError(err)
	}
	return kbImportToPb(batch, jobs), nil
}
