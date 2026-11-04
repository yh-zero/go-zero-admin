// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appauth

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type SubmitAppFeedbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSubmitAppFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SubmitAppFeedbackLogic {
	return &SubmitAppFeedbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SubmitAppFeedbackLogic) SubmitAppFeedback(req *types.AppSubmitFeedbackReq) (resp *types.EmptyResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if l.svcCtx.AppAuthRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	_, err = l.svcCtx.AppAuthRpc.SubmitAppFeedback(l.ctx, &apppb.AppSubmitFeedbackRequest{
		AccessToken: appToken(l.ctx), Content: req.Content,
	})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
