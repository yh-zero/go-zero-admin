package tenantservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/tenant/rpc/internal/svc"
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
func (l *GetTenantLogic) GetTenant(in *tenantpb.GetTenantRequest) (*tenantpb.GetTenantResponse, error) {
	if in.Actor == nil || in.Actor.UserID <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if in.TenantID <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	r, err := l.svcCtx.TenantRegistryModel.FindOne(l.ctx, uint64(in.TenantID))
	if errors.Is(err, sqlx.ErrNotFound) {
		return nil, xerr.NewErrCode(xerr.NOT_FOUND)
	}
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if r.DeletedAt.Valid {
		return nil, xerr.NewErrCode(xerr.NOT_FOUND)
	}
	return &tenantpb.GetTenantResponse{Tenant: mapTenant(r)}, nil
}
