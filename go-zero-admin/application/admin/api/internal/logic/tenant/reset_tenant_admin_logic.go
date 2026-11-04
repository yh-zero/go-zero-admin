package tenant

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/xerr"
)

type ResetTenantAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResetTenantAdminLogic(ctx context.Context, s *svc.ServiceContext) *ResetTenantAdminLogic {
	return &ResetTenantAdminLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *ResetTenantAdminLogic) ResetTenantAdmin(req *types.TenantIdReq) (*types.ResetTenantAdminResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := authorizeTenant(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	result, err := l.svcCtx.TenantRpc.ResetTenantAdmin(l.ctx, &tenantpb.ResetTenantAdminRequest{Actor: actor, TenantID: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.ResetTenantAdminResp{NewPassword: result.NewPassword}, nil
}
