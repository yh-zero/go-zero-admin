// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package aiusage

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUsageSummaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUsageSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUsageSummaryLogic {
	return &GetUsageSummaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUsageSummaryLogic) GetUsageSummary(req *types.AiUsageQueryReq) (resp *types.AiUsageSummaryResp, err error) {
	actor, err := usageActor(l.ctx, l.svcCtx, req, false)
	if err != nil {
		return nil, err
	}
	query := selfUsageQuery(req)
	query.Actor = actor
	result, err := l.svcCtx.AiRpc.GetUsageSummary(l.ctx, query)
	if err != nil {
		return nil, err
	}
	return &types.AiUsageSummaryResp{Summary: mapUsageSummary(result.Summary)}, nil
}
