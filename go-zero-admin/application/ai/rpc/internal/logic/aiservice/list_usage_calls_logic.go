package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListUsageCallsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListUsageCallsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUsageCallsLogic {
	return &ListUsageCallsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListUsageCallsLogic) ListUsageCalls(in *aipb.UsageQueryRequest) (*aipb.ListUsageCallsResponse, error) {
	if err := usageAccess(l.ctx, l.svcCtx, in.GetActor(), false); err != nil {
		return nil, err
	}
	rows, total, err := l.svcCtx.UsageStore.List(l.ctx, usageFilter(in, false))
	if err != nil {
		return nil, usageModelError(err)
	}
	return &aipb.ListUsageCallsResponse{List: usageCallMessages(rows), Page: &aipb.PageResponse{Total: total}}, nil
}
