package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminListUsageCallsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAdminListUsageCallsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListUsageCallsLogic {
	return &AdminListUsageCallsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AdminListUsageCallsLogic) AdminListUsageCalls(in *aipb.UsageQueryRequest) (*aipb.ListUsageCallsResponse, error) {
	if err := usageAccess(l.ctx, l.svcCtx, in.GetActor(), true); err != nil {
		return nil, err
	}
	rows, total, err := l.svcCtx.UsageStore.List(l.ctx, usageFilter(in, true))
	if err != nil {
		return nil, usageModelError(err)
	}
	return &aipb.ListUsageCallsResponse{List: usageCallMessages(rows), Page: &aipb.PageResponse{Total: total}}, nil
}
