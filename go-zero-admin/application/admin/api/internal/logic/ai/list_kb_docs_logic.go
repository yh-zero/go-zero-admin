package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"
)

type ListKbDocsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListKbDocsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbDocsLogic {
	return &ListKbDocsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ListKbDocsLogic) ListKbDocs(req *types.AiKbDocListReq) (*types.AiListKbDocsResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ListKbDocs(l.ctx, &aipb.ListKbDocsRequest{Actor: actor, KbId: req.Id, Page: &aipb.PageRequest{Page: req.Page, PageSize: req.PageSize}})
	if err != nil {
		return nil, err
	}
	list := make([]*types.AiKbDocInfo, 0, len(out.List))
	for _, item := range out.List {
		list = append(list, mapAiKbDoc(item))
	}
	return &types.AiListKbDocsResp{List: list, Total: out.GetPage().GetTotal()}, nil
}
