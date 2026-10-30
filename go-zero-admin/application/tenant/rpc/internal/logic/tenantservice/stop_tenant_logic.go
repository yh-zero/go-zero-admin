package tenantservicelogic

import (
	"context"
	"os/exec"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"

	"go-zero-admin/pkg/xerr"
)

type StopTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopTenantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopTenantLogic {
	return &StopTenantLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *StopTenantLogic) StopTenant(in *tenantpb.StopTenantRequest) (*tenantpb.NoDataResponse, error) {
	tenant, err := l.svcCtx.TenantRegistryModel.FindOne(l.ctx, uint64(in.TenantID))
	if err != nil {
		return nil, xerr.NewErrCodeMsg(404, "租户不存在")
	}
	cmd := exec.Command("docker", "compose", "-p", tenant.ComposeProject, "stop")
	if out, err := cmd.CombinedOutput(); err != nil {
		l.Errorf("stop tenant %s: %v %s", tenant.Slug, err, out)
		return nil, xerr.NewErrCodeMsg(500, "停用失败")
	}
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE tenant_registry SET status='stopped', updated_at=NOW() WHERE id=?", in.TenantID)
	return &tenantpb.NoDataResponse{}, nil
}
