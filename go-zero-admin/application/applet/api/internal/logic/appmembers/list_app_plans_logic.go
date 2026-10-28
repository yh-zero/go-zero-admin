// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appmembers

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppPlansLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 读取会员方案
func NewListAppPlansLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppPlansLogic {
	return &ListAppPlansLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAppPlansLogic) ListAppPlans() (resp *types.AppPlanList, err error) {
	reply, err := l.svcCtx.AppMembershipRPC.ListAppPlans(l.ctx, &pb.AppAdminRequest{Actor: appbridge.Actor(l.ctx)})
	if err != nil {
		return nil, err
	}
	response, err := appbridge.Copy[types.AppPlanList](reply)
	if err != nil {
		return nil, err
	}
	if response.List == nil {
		response.List = []types.AppPlan{}
	}
	return response, nil
}
