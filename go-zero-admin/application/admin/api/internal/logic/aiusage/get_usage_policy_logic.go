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

type GetUsagePolicyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUsagePolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUsagePolicyLogic {
	return &GetUsagePolicyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUsagePolicyLogic) GetUsagePolicy() (resp *types.AiUsagePolicyResp, err error) {
	actor, err := usageActor(l.ctx, l.svcCtx, nil, false)
	if err != nil {
		return nil, err
	}
	result, err := l.svcCtx.AiRpc.GetUsagePolicy(l.ctx, &aipb.UsagePolicyRequest{Actor: actor})
	if err != nil {
		return nil, err
	}
	return &types.AiUsagePolicyResp{Policy: mapUsagePolicy(result.Policy)}, nil
}
