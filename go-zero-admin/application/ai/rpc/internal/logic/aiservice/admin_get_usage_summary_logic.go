package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminGetUsageSummaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAdminGetUsageSummaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminGetUsageSummaryLogic {
	return &AdminGetUsageSummaryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AdminGetUsageSummaryLogic) AdminGetUsageSummary(in *aipb.UsageQueryRequest) (*aipb.UsageSummaryResponse, error) {
	if err := usageAccess(l.ctx, l.svcCtx, in.GetActor(), true); err != nil {
		return nil, err
	}
	row, err := l.svcCtx.UsageStore.Summary(l.ctx, usageFilter(in, true))
	if err != nil {
		return nil, usageModelError(err)
	}
	return &aipb.UsageSummaryResponse{Summary: usageSummaryMessage(row)}, nil
}
