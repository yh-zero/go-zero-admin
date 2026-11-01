// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appadmin

import (
	"context"

	"go-zero-admin/application/admin/api/internal/logic/appauth"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppFeedbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAppFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppFeedbackLogic {
	return &ListAppFeedbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAppFeedbackLogic) ListAppFeedback(req *types.AppFeedbackListReq) (resp *types.AppFeedbackListResp, err error) {
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	r, err := l.svcCtx.AppAdminRpc.ListAppFeedback(l.ctx, &apppb.AppFeedbackListRequest{
		Actor: appAdminActor(l.ctx), Page: &apppb.PageRequest{Page: req.Page, PageSize: req.PageSize},
		UserId: req.UserId, Status: req.Status,
	})
	if err != nil {
		return nil, err
	}
	list := make([]types.AppFeedbackInfo, 0, len(r.List))
	for _, row := range r.List {
		list = append(list, appauth.MapAppFeedback(row))
	}
	return &types.AppFeedbackListResp{List: list, Total: r.GetPage().GetTotal()}, nil
}
