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

type ListKbFeedbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListKbFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbFeedbackLogic {
	return &ListKbFeedbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListKbFeedbackLogic) ListKbFeedback(req *types.AiKbFeedbackListReq) (resp *types.AiKbFeedbackListResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ListKbFeedback(l.ctx, &aipb.ListKbFeedbackRequest{Actor: actor, KbId: req.Id, Page: &aipb.PageRequest{Page: req.Page, PageSize: req.PageSize}, Status: req.Status})
	if err != nil {
		return nil, err
	}
	return &types.AiKbFeedbackListResp{List: mapKbFeedbackList(out.List), Total: out.GetPage().GetTotal()}, nil
}
