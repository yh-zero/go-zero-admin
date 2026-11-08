package aiservicelogic

import (
	"context"
	"go-zero-admin/application/ai/rpc/internal/model"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type AdminSetUsagePolicyLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAdminSetUsagePolicyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AdminSetUsagePolicyLogic {
	return &AdminSetUsagePolicyLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AdminSetUsagePolicyLogic) AdminSetUsagePolicy(in *aipb.SetUsagePolicyRequest) (*aipb.UsagePolicyResponse, error) {
	if err := usageAccess(l.ctx, l.svcCtx, in.GetActor(), true); err != nil {
		return nil, err
	}
	if in.ExpectedVersion == nil {
		return nil, usageModelError(model.ErrUsageInvalid)
	}
	row, err := l.svcCtx.UsageStore.SetPolicy(l.ctx, in.UserId, *in.ExpectedVersion, model.UsagePolicyPatch{DailyRequestLimit: in.DailyRequestLimit, DailyTokenWarning: in.DailyTokenWarning, GlobalConcurrentLimit: in.GlobalConcurrentLimit, InheritDefault: in.InheritDefault, UpdatedBy: in.Actor.UserId})
	if err != nil {
		return nil, usageModelError(err)
	}
	return &aipb.UsagePolicyResponse{Policy: usagePolicyMessage(l.ctx, l.svcCtx, row, true)}, nil
}
