// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package aiusage

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetUsageSummaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminGetUsageSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetUsageSummaryLogic {
	return &AdminGetUsageSummaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminGetUsageSummaryLogic) AdminGetUsageSummary(req *types.AiUsageAdminQueryReq) (resp *types.AiUsageSummaryResp, err error) {
	actor, err := usageActor(l.ctx, l.svcCtx, req, true)
	if err != nil {
		return nil, err
	}
	query := adminUsageQuery(req)
	query.Actor = actor
	result, err := l.svcCtx.AiRpc.AdminGetUsageSummary(l.ctx, query)
	if err != nil {
		return nil, err
	}
	return &types.AiUsageSummaryResp{Summary: mapUsageSummary(result.Summary)}, nil
}
