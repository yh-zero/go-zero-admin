// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package aiusage

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListUsageCallsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListUsageCallsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUsageCallsLogic {
	return &ListUsageCallsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListUsageCallsLogic) ListUsageCalls(req *types.AiUsageQueryReq) (resp *types.AiUsageCallsResp, err error) {
	actor, err := usageActor(l.ctx, l.svcCtx, req, false)
	if err != nil {
		return nil, err
	}
	query := selfUsageQuery(req)
	query.Actor = actor
	result, err := l.svcCtx.AiRpc.ListUsageCalls(l.ctx, query)
	if err != nil {
		return nil, err
	}
	return &types.AiUsageCallsResp{List: mapUsageCalls(result.List), Total: result.GetPage().GetTotal()}, nil
}
