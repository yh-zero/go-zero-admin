package tenant

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/tenant/rpc/tenantpb"
)

type StartTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartTenantLogic(ctx context.Context, s *svc.ServiceContext) *StartTenantLogic {
	return &StartTenantLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *StartTenantLogic) StartTenant(req *types.TenantIdReq) (*types.EmptyResp, error) {
	actor, err := authorizeTenant(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	_, err = l.svcCtx.TenantRpc.StartTenant(l.ctx, &tenantpb.StartTenantRequest{Actor: actor, TenantID: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
