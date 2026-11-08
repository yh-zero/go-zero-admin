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

type ListKbEvaluationsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListKbEvaluationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbEvaluationsLogic {
	return &ListKbEvaluationsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListKbEvaluationsLogic) ListKbEvaluations(req *types.AiListKbEvaluationsReq) (resp *types.AiListKbEvaluationsResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ListKbEvaluations(l.ctx, &aipb.ListKbEvaluationsRequest{Actor: actor, KbId: req.Id,
		Page: &aipb.PageRequest{Page: req.Page, PageSize: req.PageSize}, Status: req.Status, RequestId: req.RequestId})
	if err != nil {
		return nil, err
	}
	return &types.AiListKbEvaluationsResp{List: mapKbEvaluationList(out.GetList()), Total: out.GetPage().GetTotal()}, nil
}
