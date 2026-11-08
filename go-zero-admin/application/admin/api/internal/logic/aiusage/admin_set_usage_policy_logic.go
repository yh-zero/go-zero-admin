// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package aiusage

import (
	"context"
	"go-zero-admin/application/ai/rpc/aipb"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminSetUsagePolicyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAdminSetUsagePolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminSetUsagePolicyLogic {
	return &AdminSetUsagePolicyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AdminSetUsagePolicyLogic) AdminSetUsagePolicy(req *types.AiSetUsagePolicyReq) (resp *types.AiUsagePolicyResp, err error) {
	actor, err := usageActor(l.ctx, l.svcCtx, req, true)
	if err != nil {
		return nil, err
	}
	result, err := l.svcCtx.AiRpc.AdminSetUsagePolicy(l.ctx, &aipb.SetUsagePolicyRequest{Actor: actor, UserId: req.UserId, ExpectedVersion: req.ExpectedVersion, DailyRequestLimit: req.DailyRequestLimit, DailyTokenWarning: req.DailyTokenWarning, GlobalConcurrentLimit: req.GlobalConcurrentLimit, InheritDefault: req.InheritDefault})
	if err != nil {
		return nil, err
	}
	return &types.AiUsagePolicyResp{Policy: mapUsagePolicy(result.Policy)}, nil
}
