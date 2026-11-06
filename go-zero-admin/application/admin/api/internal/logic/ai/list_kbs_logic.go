package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"
)

type ListKbsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListKbsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbsLogic {
	return &ListKbsLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ListKbsLogic) ListKbs(req *types.AiPageReq) (*types.AiListKbsResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ListKbs(l.ctx, &aipb.ListKbsRequest{Actor: actor, Page: &aipb.PageRequest{Page: req.Page, PageSize: req.PageSize}})
	if err != nil {
		return nil, err
	}
	list := make([]*types.AiKbInfo, 0, len(out.List))
	for _, item := range out.List {
		list = append(list, mapAiKb(item))
	}
	return &types.AiListKbsResp{List: list, Total: out.GetPage().GetTotal()}, nil
}
