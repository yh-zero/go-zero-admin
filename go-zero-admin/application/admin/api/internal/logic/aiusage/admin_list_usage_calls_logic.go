// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package aiusage

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminListUsageCallsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminListUsageCallsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminListUsageCallsLogic {
	return &AdminListUsageCallsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminListUsageCallsLogic) AdminListUsageCalls(req *types.AiUsageAdminQueryReq) (resp *types.AiUsageCallsResp, err error) {
	actor, err := usageActor(l.ctx, l.svcCtx, req, true)
	if err != nil {
		return nil, err
	}
	query := adminUsageQuery(req)
	query.Actor = actor
	result, err := l.svcCtx.AiRpc.AdminListUsageCalls(l.ctx, query)
	if err != nil {
		return nil, err
	}
	return &types.AiUsageCallsResp{List: mapUsageCalls(result.List), Total: result.GetPage().GetTotal()}, nil
}
