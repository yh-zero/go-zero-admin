package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKbsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListKbsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbsLogic {
	return &ListKbsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListKbsLogic) ListKbs(in *aipb.ListKbsRequest) (*aipb.ListKbsResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	page, size, err := aiPagination(in.Page)
	if err != nil {
		return nil, err
	}
	rows, total, err := l.svcCtx.AiKbModel.ListOwned(l.ctx, in.Actor.UserId, page, size)
	if err != nil {
		return nil, aiModelError(err)
	}
	list := make([]*aipb.KbInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, kbRowToPb(r))
	}
	return &aipb.ListKbsResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
