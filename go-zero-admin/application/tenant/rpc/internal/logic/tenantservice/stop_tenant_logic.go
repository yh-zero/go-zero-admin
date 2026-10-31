package tenantservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/xerr"
	"time"
)

type StopTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopTenantLogic(ctx context.Context, s *svc.ServiceContext) *StopTenantLogic {
	return &StopTenantLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *StopTenantLogic) StopTenant(in *tenantpb.StopTenantRequest) (*tenantpb.NoDataResponse, error) {
	tenant, release, err := managedTenant(l.ctx, l.svcCtx, in.Actor, in.TenantID)
	if err != nil {
		return nil, err
	}
	defer release()
	ctx, cancel := context.WithTimeout(l.ctx, 90*time.Second)
	defer cancel()
	if err = routeTenant(ctx, l.svcCtx, tenant, false); err != nil {
		return nil, tenantFailure(l.svcCtx, tenant, "停用租户路由失败")
	}
	if _, err = compose(ctx, l.svcCtx, tenant, nil, "stop", "--timeout", "10"); err != nil {
		return nil, tenantFailure(l.svcCtx, tenant, "停止租户服务失败")
	}
	if err = l.svcCtx.TenantRegistryModel.SetState(ctx, tenant, "stopped", "", tenant.AdminAccountReady == 1); err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &tenantpb.NoDataResponse{}, nil
}
