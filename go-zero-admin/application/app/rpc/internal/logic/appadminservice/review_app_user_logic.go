package appadminservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
	"unicode/utf8"
)

type ReviewAppUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewReviewAppUserLogic(ctx context.Context, s *svc.ServiceContext) *ReviewAppUserLogic {
	return &ReviewAppUserLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *ReviewAppUserLogic) ReviewAppUser(in *apppb.AppReviewRequest) (*apppb.NoDataResponse, error) {
	if err := requireActor(in.Actor); err != nil {
		return nil, err
	}
	if !validStatus(in.Status) || utf8.RuneCountInString(in.Reason) > 255 ||
		!validStatus(in.ExpectedStatus) || in.ExpectedReason == nil || utf8.RuneCountInString(in.GetExpectedReason()) > 512 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if in.UserID <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	expected := model.ReviewDecision{Status: in.ExpectedStatus, Reason: in.GetExpectedReason()}
	next := model.ReviewDecision{Status: in.Status, Reason: in.Reason}
	err := l.svcCtx.AppUserModel.SetReview(l.ctx, uint64(in.UserID), expected, next, func(ctx context.Context) error {
		if err := l.svcCtx.Sessions.RevokeAll(ctx, in.UserID); err != nil {
			return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, model.ErrAppReviewChanged) {
			return nil, xerr.NewErrCodeMsg(409, "审核决定已变化，请核对最新状态和说明后重新操作")
		}
		if business, ok := xerr.AsCodeError(err); ok {
			return nil, business
		}
		if errors.Is(err, model.ErrNotFound) {
			return nil, xerr.NewErrCode(xerr.NOT_FOUND)
		}
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &apppb.NoDataResponse{}, nil
}
