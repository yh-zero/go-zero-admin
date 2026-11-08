package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListTaskFeedbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListTaskFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListTaskFeedbackLogic {
	return &ListTaskFeedbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListTaskFeedbackLogic) ListTaskFeedback(in *aipb.ListTaskFeedbackRequest) (*aipb.ListTaskFeedbackResponse, error) {
	if err := requireAiActor(in.GetActor()); err != nil {
		return nil, err
	}
	if in.TaskId <= 0 {
		return nil, xerr.NewErrCode(400)
	}
	if _, err := l.svcCtx.AiTaskModel.FindSummaryOwned(l.ctx, in.TaskId, in.Actor.UserId); err != nil {
		return nil, feedbackError(err)
	}
	rows, err := l.svcCtx.AiKbFeedbackModel.ListTaskOwned(l.ctx, in.Actor.UserId, in.TaskId)
	if err != nil {
		return nil, feedbackError(err)
	}
	list, err := kbFeedbackProtos(rows)
	return &aipb.ListTaskFeedbackResponse{List: list}, err
}
