package tenant

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/xerr"
)

type GetTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTenantLogic(ctx context.Context, s *svc.ServiceContext) *GetTenantLogic {
	return &GetTenantLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *GetTenantLogic) GetTenant(req *types.TenantIdReq) (*types.GetTenantResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := authorizeTenant(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	result, err := l.svcCtx.TenantRpc.GetTenant(l.ctx, &tenantpb.GetTenantRequest{Actor: actor, TenantID: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.GetTenantResp{Tenant: mapTenantInfo(result.Tenant)}, nil
}
