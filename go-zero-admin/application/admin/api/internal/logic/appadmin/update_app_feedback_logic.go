// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appadmin

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAppFeedbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateAppFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAppFeedbackLogic {
	return &UpdateAppFeedbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateAppFeedbackLogic) UpdateAppFeedback(req *types.AppUpdateFeedbackReq) (resp *types.EmptyResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	_, err = l.svcCtx.AppAdminRpc.UpdateAppFeedback(l.ctx, &apppb.AppUpdateFeedbackRequest{
		Actor: appAdminActor(l.ctx), FeedbackId: req.Id, Reply: req.Reply, Status: req.Status,
	})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
