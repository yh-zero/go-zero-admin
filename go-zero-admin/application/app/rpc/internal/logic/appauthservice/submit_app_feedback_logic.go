package appauthservicelogic

import (
	"context"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitAppFeedbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSubmitAppFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitAppFeedbackLogic {
	return &SubmitAppFeedbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SubmitAppFeedbackLogic) SubmitAppFeedback(in *apppb.AppSubmitFeedbackRequest) (*apppb.NoDataResponse, error) {
	user, _, err := CurrentAppUser(l.ctx, l.svcCtx, in.AccessToken)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Content) == "" || utf8.RuneCountInString(in.Content) > 2000 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	_, err = l.svcCtx.AppFeedbackModel.Insert(l.ctx, &model.AppFeedback{UserId: user.Id, Content: in.Content, Reply: "", Status: "pending"})
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &apppb.NoDataResponse{}, nil
}
