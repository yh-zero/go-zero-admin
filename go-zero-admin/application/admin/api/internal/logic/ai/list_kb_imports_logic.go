// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package ai

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKbImportsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListKbImportsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbImportsLogic {
	return &ListKbImportsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListKbImportsLogic) ListKbImports(req *types.AiKbImportListReq) (resp *types.AiListKbImportsResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ListKbImports(l.ctx, &aipb.ListKbImportsRequest{Actor: actor, KbId: req.Id, Page: &aipb.PageRequest{Page: req.Page, PageSize: req.PageSize}})
	if err != nil {
		return nil, err
	}
	list := make([]*types.AiKbIngestBatchInfo, 0, len(out.List))
	for _, row := range out.List {
		list = append(list, mapAiKbBatch(row))
	}
	return &types.AiListKbImportsResp{List: list, Total: out.GetPage().GetTotal()}, nil
}
