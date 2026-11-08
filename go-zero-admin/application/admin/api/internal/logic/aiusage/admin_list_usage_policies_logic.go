// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package aiusage

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminListUsagePoliciesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminListUsagePoliciesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListUsagePoliciesLogic {
	return &AdminListUsagePoliciesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminListUsagePoliciesLogic) AdminListUsagePolicies(req *types.AiUsageAdminQueryReq) (resp *types.AiUsagePoliciesResp, err error) {
	actor, err := usageActor(l.ctx, l.svcCtx, req, true)
	if err != nil {
		return nil, err
	}
	query := adminUsageQuery(req)
	query.Actor = actor
	result, err := l.svcCtx.AiRpc.AdminListUsagePolicies(l.ctx, query)
	if err != nil {
		return nil, err
	}
	list := make([]*types.AiUsagePolicyInfo, 0, len(result.List))
	for _, r := range result.List {
		list = append(list, mapUsagePolicy(r))
	}
	return &types.AiUsagePoliciesResp{List: list, Total: result.GetPage().GetTotal()}, nil
}
