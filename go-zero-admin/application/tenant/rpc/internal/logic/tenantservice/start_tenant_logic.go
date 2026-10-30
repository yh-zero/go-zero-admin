package tenantservicelogic

import (
	"context"
	"os/exec"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"

	"go-zero-admin/pkg/xerr"
)

type StartTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartTenantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartTenantLogic {
	return &StartTenantLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *StartTenantLogic) StartTenant(in *tenantpb.StartTenantRequest) (*tenantpb.NoDataResponse, error) {
	tenant, err := l.svcCtx.TenantRegistryModel.FindOne(l.ctx, uint64(in.TenantID))
	if err != nil {
		return nil, xerr.NewErrCodeMsg(404, "租户不存在")
	}
	cmd := exec.Command("docker", "compose", "-p", tenant.ComposeProject, "start")
	if out, err := cmd.CombinedOutput(); err != nil {
		l.Errorf("start tenant %s: %v %s", tenant.Slug, err, out)
		return nil, xerr.NewErrCodeMsg(500, "启用失败")
	}
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE tenant_registry SET status='active', updated_at=NOW() WHERE id=?", in.TenantID)
	return &tenantpb.NoDataResponse{}, nil
}
