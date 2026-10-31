package appadmin

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/admin/api/internal/logic/appauth"
)

type ListMembershipsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListMembershipsLogic(ctx context.Context, s *svc.ServiceContext) *ListMembershipsLogic {
	return &ListMembershipsLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *ListMembershipsLogic) ListMemberships(req *types.AppListMembershipReq) (*types.AppListMembershipResp, error) {
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	r, err := l.svcCtx.AppAdminRpc.ListMemberships(l.ctx, &apppb.ListMembershipRequest{Actor: appAdminActor(l.ctx), Page: &apppb.PageRequest{Page: req.Page, PageSize: req.PageSize}, UserID: req.UserId, PlanCode: req.PlanCode, ActiveOnly: req.ActiveOnly})
	if err != nil {
		return nil, err
	}
	list := make([]types.MembershipGrantInfo, 0, len(r.List))
	for _, u := range r.List {
		list = append(list, appauth.MapMembership(u))
	}
	return &types.AppListMembershipResp{List: list, Total: r.GetPage().GetTotal()}, nil
}
