package casbinservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type ListPoliciesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPoliciesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPoliciesLogic {
	return &ListPoliciesLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// ListPolicies 全量返回 casbin_rule（API 层初始化/刷新 enforcer 用）
func (l *ListPoliciesLogic) ListPolicies(in *adminpb.ListPoliciesRequest) (*adminpb.ListPoliciesResponse, error) {
	rows, err := l.svcCtx.CasbinRuleModel.ListActivePolicies(l.ctx)
	if err != nil {
		l.Errorf("query casbin_rule error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	rules := make([]*adminpb.PolicyRule, 0, len(rows))
	for _, r := range rows {
		rules = append(rules, &adminpb.PolicyRule{
			Ptype: r.Ptype, V0: r.V0, V1: r.V1, V2: r.V2, V3: r.V3, V4: r.V4, V5: r.V5,
		})
	}
	return &adminpb.ListPoliciesResponse{Rules: rules}, nil
}
