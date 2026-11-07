package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKbImportsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListKbImportsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbImportsLogic {
	return &ListKbImportsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListKbImportsLogic) ListKbImports(in *aipb.ListKbImportsRequest) (*aipb.ListKbImportsResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	page, size, err := aiPagination(in.Page)
	if err != nil {
		return nil, err
	}
	rows, total, err := l.svcCtx.KbIngestQueue.ListOwned(l.ctx, in.KbId, in.Actor.UserId, page, size)
	if err != nil {
		return nil, kbQueueError(err)
	}
	list := make([]*aipb.KbIngestBatchInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, kbBatchToPb(row))
	}
	return &aipb.ListKbImportsResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
