package appmembershipservicelogic

import (
	"context"
	appaccount "go-zero-admin/application/applet/rpc/internal/logic/appaccountservice"
	"go-zero-admin/application/applet/rpc/internal/model"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppPlansLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAppPlansLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppPlansLogic {
	return &ListAppPlansLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListAppPlansLogic) ListAppPlans(in *pb.AppAdminRequest) (*pb.AppPlanListResponse, error) {
	if in == nil {
		return nil, membershipError("请求参数不能为空")
	}
	if err := appaccount.AuthorizeAdmin(l.ctx, l.svcCtx, in.Actor, plansPath, "GET"); err != nil {
		return nil, err
	}
	var plans []model.AppMembershipPlan
	if err := l.svcCtx.DB.WithContext(l.ctx).Where("code = ? AND version = ? AND enabled = ?", "vip", 1, true).Order("id").Find(&plans).Error; err != nil {
		return nil, err
	}
	result := &pb.AppPlanListResponse{List: []*pb.AppPlanInfo{}}
	for _, plan := range plans {
		rights, err := entitlements(plan.Entitlements)
		if err != nil {
			return nil, err
		}
		result.List = append(result.List, &pb.AppPlanInfo{ID: plan.ID, Code: plan.Code, Name: plan.Name, Version: plan.Version, Entitlements: rights})
	}
	return result, nil
}
