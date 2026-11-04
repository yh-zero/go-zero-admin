package tenant

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/xerr"
)

type StopTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopTenantLogic(ctx context.Context, s *svc.ServiceContext) *StopTenantLogic {
	return &StopTenantLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *StopTenantLogic) StopTenant(req *types.TenantIdReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := authorizeTenant(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	_, err = l.svcCtx.TenantRpc.StopTenant(l.ctx, &tenantpb.StopTenantRequest{Actor: actor, TenantID: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
