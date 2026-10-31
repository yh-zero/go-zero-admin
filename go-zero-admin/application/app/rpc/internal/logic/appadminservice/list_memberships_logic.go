package appadminservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/app/rpc/apppb"
	appauth "go-zero-admin/application/app/rpc/internal/logic/appauthservice"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type ListMembershipsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListMembershipsLogic(ctx context.Context, s *svc.ServiceContext) *ListMembershipsLogic {
	return &ListMembershipsLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *ListMembershipsLogic) ListMemberships(in *apppb.ListMembershipRequest) (*apppb.ListMembershipResponse, error) {
	if err := requireActor(in.Actor); err != nil {
		return nil, err
	}
	page, size, err := pagination(in.GetPage())
	if err != nil {
		return nil, err
	}
	if in.UserID < 0 || len(in.PlanCode) > 32 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	rows, total, err := l.svcCtx.AppMembershipGrantModel.List(l.ctx, in.UserID, in.PlanCode, in.ActiveOnly, page, size)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*apppb.MembershipGrantInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, appauth.MapMembership(r))
	}
	return &apppb.ListMembershipResponse{List: list, Page: &apppb.PageResponse{Total: total}}, nil
}
