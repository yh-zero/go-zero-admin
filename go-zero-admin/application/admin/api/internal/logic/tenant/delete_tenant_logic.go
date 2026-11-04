package tenant

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/xerr"
)

type DeleteTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteTenantLogic(ctx context.Context, s *svc.ServiceContext) *DeleteTenantLogic {
	return &DeleteTenantLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *DeleteTenantLogic) DeleteTenant(req *types.DeleteTenantReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := authorizeTenant(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	_, err = l.svcCtx.TenantRpc.DeleteTenant(l.ctx, &tenantpb.DeleteTenantRequest{Actor: actor, TenantID: req.Id, ConfirmSlug: req.ConfirmSlug})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
