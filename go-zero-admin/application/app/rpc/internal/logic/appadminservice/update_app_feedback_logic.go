package appadminservicelogic

import (
	"context"
	"unicode/utf8"

	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAppFeedbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateAppFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAppFeedbackLogic {
	return &UpdateAppFeedbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateAppFeedbackLogic) UpdateAppFeedback(in *apppb.AppUpdateFeedbackRequest) (*apppb.NoDataResponse, error) {
	if err := requireActor(in.Actor); err != nil {
		return nil, err
	}
	if in.FeedbackId <= 0 || in.Reply == nil && in.Status == nil || in.Reply != nil && utf8.RuneCountInString(*in.Reply) > 2000 || in.Status != nil && *in.Status != "pending" && *in.Status != "resolved" {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if err := l.svcCtx.AppFeedbackModel.Patch(l.ctx, uint64(in.FeedbackId), in.Reply, in.Status); err != nil {
		return nil, appDataError(err)
	}
	return &apppb.NoDataResponse{}, nil
}
