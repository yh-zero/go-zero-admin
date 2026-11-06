package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKbDocsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListKbDocsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbDocsLogic {
	return &ListKbDocsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListKbDocsLogic) ListKbDocs(in *aipb.ListKbDocsRequest) (*aipb.ListKbDocsResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	page, size, err := aiPagination(in.Page)
	if err != nil {
		return nil, err
	}
	rows, total, err := l.svcCtx.AiKbDocModel.ListInKb(l.ctx, in.KbId, in.Actor.UserId, page, size)
	if err != nil {
		return nil, aiModelError(err)
	}
	list := make([]*aipb.KbDocInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, kbDocRowToPb(r))
	}
	return &aipb.ListKbDocsResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
