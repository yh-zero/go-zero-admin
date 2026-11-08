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

type ListTaskFeedbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListTaskFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTaskFeedbackLogic {
	return &ListTaskFeedbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListTaskFeedbackLogic) ListTaskFeedback(req *types.AiIdReq) (resp *types.AiTaskFeedbackListResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ListTaskFeedback(l.ctx, &aipb.ListTaskFeedbackRequest{Actor: actor, TaskId: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.AiTaskFeedbackListResp{List: mapKbFeedbackList(out.List)}, nil
}
