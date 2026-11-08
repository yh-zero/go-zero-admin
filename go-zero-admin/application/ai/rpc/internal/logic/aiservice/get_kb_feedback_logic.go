package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetKbFeedbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetKbFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKbFeedbackLogic {
	return &GetKbFeedbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetKbFeedbackLogic) GetKbFeedback(in *aipb.GetKbFeedbackRequest) (*aipb.KbFeedbackResponse, error) {
	if err := requireAiActor(in.GetActor()); err != nil {
		return nil, err
	}
	if in.KbId <= 0 || in.FeedbackId <= 0 {
		return nil, xerr.NewErrCode(400)
	}
	row, err := l.svcCtx.AiKbFeedbackModel.FindOwned(l.ctx, in.Actor.UserId, in.KbId, in.FeedbackId)
	if err != nil {
		return nil, feedbackError(err)
	}
	item, err := kbFeedbackProto(row)
	return &aipb.KbFeedbackResponse{Feedback: item}, err
}
