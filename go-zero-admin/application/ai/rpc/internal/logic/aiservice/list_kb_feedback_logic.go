package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKbFeedbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListKbFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbFeedbackLogic {
	return &ListKbFeedbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListKbFeedbackLogic) ListKbFeedback(in *aipb.ListKbFeedbackRequest) (*aipb.ListKbFeedbackResponse, error) {
	if err := requireAiActor(in.GetActor()); err != nil {
		return nil, err
	}
	if in.KbId <= 0 {
		return nil, xerr.NewErrCode(400)
	}
	page, size, err := aiPagination(in.Page)
	if err != nil {
		return nil, err
	}
	status := in.Status
	if status == "" {
		status = "pending"
	}
	if status != "pending" && status != "accepted" && status != "dismissed" && status != "all" {
		return nil, xerr.NewErrCode(400)
	}
	if _, err = l.svcCtx.AiKbModel.FindOwned(l.ctx, in.KbId, in.Actor.UserId); err != nil {
		return nil, feedbackError(err)
	}
	rows, total, err := l.svcCtx.AiKbFeedbackModel.ListOwned(l.ctx, in.Actor.UserId, in.KbId, page, size, status)
	if err != nil {
		return nil, feedbackError(err)
	}
	list, err := kbFeedbackProtos(rows)
	return &aipb.ListKbFeedbackResponse{List: list, Page: &aipb.PageResponse{Total: total}}, err
}
