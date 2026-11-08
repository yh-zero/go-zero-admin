package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUsagePolicyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUsagePolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUsagePolicyLogic {
	return &GetUsagePolicyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUsagePolicyLogic) GetUsagePolicy(in *aipb.UsagePolicyRequest) (*aipb.UsagePolicyResponse, error) {
	if err := usageAccess(l.ctx, l.svcCtx, in.GetActor(), false); err != nil {
		return nil, err
	}
	row, err := l.svcCtx.UsageStore.GetPolicy(l.ctx, in.Actor.UserId)
	if err != nil {
		return nil, usageModelError(err)
	}
	return &aipb.UsagePolicyResponse{Policy: usagePolicyMessage(l.ctx, l.svcCtx, row, false)}, nil
}
