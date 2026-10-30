package tenantservicelogic

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"

	"go-zero-admin/pkg/xerr"
)

type DeleteTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteTenantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteTenantLogic {
	return &DeleteTenantLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteTenantLogic) DeleteTenant(in *tenantpb.DeleteTenantRequest) (*tenantpb.NoDataResponse, error) {
	tenant, err := l.svcCtx.TenantRegistryModel.FindOne(l.ctx, uint64(in.TenantID))
	if err != nil {
		return nil, xerr.NewErrCodeMsg(404, "租户不存在")
	}
	if in.ConfirmSlug != tenant.Slug {
		return nil, xerr.NewErrCodeMsg(400, "确认标识不匹配，删除已取消")
	}
	cmd := exec.Command("docker", "compose", "-p", tenant.ComposeProject, "down", "-v")
	if out, err := cmd.CombinedOutput(); err != nil {
		l.Errorf("delete tenant %s: %v %s", tenant.Slug, err, out)
		return nil, xerr.NewErrCodeMsg(500, "删除失败")
	}
	// 删 nginx 配置 + env 文件
	cfg := l.svcCtx.Config.Tenant
	_ = os.Remove(filepath.Join(cfg.NginxConfDir, tenant.Slug+".conf"))
	_ = os.Remove(filepath.Join(cfg.StackTemplateDir, ".env."+tenant.Slug))
	// 软删登记
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE tenant_registry SET deleted_at=NOW() WHERE id=?", in.TenantID)
	return &tenantpb.NoDataResponse{}, nil
}
