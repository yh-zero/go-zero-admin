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

type ReviewKbFeedbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReviewKbFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReviewKbFeedbackLogic {
	return &ReviewKbFeedbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReviewKbFeedbackLogic) ReviewKbFeedback(req *types.AiReviewKbFeedbackReq) (resp *types.AiKbFeedbackResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ReviewKbFeedback(l.ctx, &aipb.ReviewKbFeedbackRequest{Actor: actor, KbId: req.Id, FeedbackId: req.FeedbackId, ExpectedRevision: req.ExpectedRevision, Status: req.Status, Review: kbFeedbackReviewRPC(req.Review)})
	if err != nil {
		return nil, err
	}
	return &types.AiKbFeedbackResp{Feedback: mapKbFeedback(out.Feedback)}, nil
}
