package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminListUsagePoliciesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAdminListUsagePoliciesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListUsagePoliciesLogic {
	return &AdminListUsagePoliciesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AdminListUsagePoliciesLogic) AdminListUsagePolicies(in *aipb.UsageQueryRequest) (*aipb.ListUsagePoliciesResponse, error) {
	if err := usageAccess(l.ctx, l.svcCtx, in.GetActor(), true); err != nil {
		return nil, err
	}
	rows, total, err := l.svcCtx.UsageStore.ListPolicies(l.ctx, in.GetPage().GetPage(), in.GetPage().GetPageSize(), in.UserId)
	if err != nil {
		return nil, usageModelError(err)
	}
	list := make([]*aipb.UsagePolicyInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, usagePolicyMessage(l.ctx, l.svcCtx, row, true))
	}
	return &aipb.ListUsagePoliciesResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
