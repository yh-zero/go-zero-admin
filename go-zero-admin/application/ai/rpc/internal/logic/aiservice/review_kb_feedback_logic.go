package aiservicelogic

import (
	"context"
	"encoding/json"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReviewKbFeedbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReviewKbFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReviewKbFeedbackLogic {
	return &ReviewKbFeedbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ReviewKbFeedbackLogic) ReviewKbFeedback(in *aipb.ReviewKbFeedbackRequest) (*aipb.KbFeedbackResponse, error) {
	if err := requireAiActor(in.GetActor()); err != nil {
		return nil, err
	}
	if in.KbId <= 0 || in.FeedbackId <= 0 || in.ExpectedRevision < 1 {
		return nil, xerr.NewErrCode(400)
	}
	if err := validateReviewShape(in.Review, in.Status); err != nil {
		return nil, err
	}
	row, err := l.svcCtx.AiKbFeedbackModel.FindOwned(l.ctx, in.Actor.UserId, in.KbId, in.FeedbackId)
	if err != nil {
		return nil, feedbackError(err)
	}
	if err = validateReviewEvidence(row.Snapshot, in.Review); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(in.Review)
	if err != nil {
		return nil, xerr.NewErrCode(400)
	}
	row, err = l.svcCtx.AiKbFeedbackModel.ReviewOwned(l.ctx, in.Actor.UserId, in.KbId, in.FeedbackId, in.ExpectedRevision, in.Status, string(raw))
	if err != nil {
		return nil, feedbackError(err)
	}
	item, err := kbFeedbackProto(row)
	return &aipb.KbFeedbackResponse{Feedback: item}, err
}
