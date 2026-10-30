package tenantservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"

	"go-zero-admin/pkg/xerr"
)

type GetTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTenantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTenantLogic {
	return &GetTenantLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetTenantLogic) GetTenant(in *tenantpb.GetTenantRequest) (*tenantpb.GetTenantResponse, error) {
	var r tenantRow
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &r,
		"SELECT id, slug, domain, compose_project, status, error_msg, created_by, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at), COALESCE(admin_account_ready,0) FROM tenant_registry WHERE id=? AND deleted_at IS NULL",
		in.TenantID)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(404, "租户不存在")
	}
	return &tenantpb.GetTenantResponse{Tenant: mapTenant(&r)}, nil
}
