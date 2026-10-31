package tenant

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/tenant/rpc/tenantpb"
)

type CreateTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateTenantLogic(ctx context.Context, s *svc.ServiceContext) *CreateTenantLogic {
	return &CreateTenantLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *CreateTenantLogic) CreateTenant(req *types.CreateTenantReq) (*types.CreateTenantResp, error) {
	actor, err := authorizeTenant(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	result, err := l.svcCtx.TenantRpc.CreateTenant(l.ctx, &tenantpb.CreateTenantRequest{Actor: actor, Slug: req.Slug, AdminUsername: req.AdminUsername, AdminPassword: req.AdminPassword})
	if err != nil {
		return nil, err
	}
	return &types.CreateTenantResp{Tenant: mapTenantInfo(result.Tenant), AdminPassword: result.AdminPassword}, nil
}
