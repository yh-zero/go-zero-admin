package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUsageSummaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUsageSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUsageSummaryLogic {
	return &GetUsageSummaryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUsageSummaryLogic) GetUsageSummary(in *aipb.UsageQueryRequest) (*aipb.UsageSummaryResponse, error) {
	if err := usageAccess(l.ctx, l.svcCtx, in.GetActor(), false); err != nil {
		return nil, err
	}
	row, err := l.svcCtx.UsageStore.Summary(l.ctx, usageFilter(in, false))
	if err != nil {
		return nil, usageModelError(err)
	}
	return &aipb.UsageSummaryResponse{Summary: usageSummaryMessage(row)}, nil
}
