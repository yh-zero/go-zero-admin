package tenantservicelogic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/tenant/rpc/internal/model"
	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/xerr"
)

var slugPattern = regexp.MustCompile("^[a-z0-9][a-z0-9-]{0,30}[a-z0-9]$")
var domainPattern = regexp.MustCompile("^[a-z0-9][a-z0-9.-]*[a-z0-9]$")

type StartTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartTenantLogic(ctx context.Context, s *svc.ServiceContext) *StartTenantLogic {
	return &StartTenantLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *StartTenantLogic) StartTenant(in *tenantpb.StartTenantRequest) (*tenantpb.NoDataResponse, error) {
	tenant, release, err := managedTenant(l.ctx, l.svcCtx, in.Actor, in.TenantID)
	if err != nil {
		return nil, err
	}
	defer release()
	if tenant.AdminAccountReady != 1 {
		return nil, xerr.NewErrCodeMsg(400, "租户初始化尚未完成，不能启动")
	}
	ctx, cancel := context.WithTimeout(l.ctx, 90*time.Second)
	defer cancel()
	if _, err = compose(ctx, l.svcCtx, tenant, nil, "up", "-d", "--no-build", "--pull", "never", "--wait", "--wait-timeout", "60"); err != nil {
		return nil, tenantFailure(l.svcCtx, tenant, "启动租户服务失败", err)
	}
	if err = waitTenantReady(ctx, l.svcCtx, tenant); err != nil {
		return nil, tenantFailure(l.svcCtx, tenant, "租户依赖未就绪", err)
	}
	if err = routeTenant(ctx, l.svcCtx, tenant, true); err != nil {
		return nil, tenantFailure(l.svcCtx, tenant, "启用租户路由失败", err)
	}
	if err = l.svcCtx.TenantRegistryModel.SetState(ctx, tenant, "active", "", true); err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &tenantpb.NoDataResponse{}, nil
}

// 锁覆盖完整外部操作；TTL 大于最长创建任务，禁止同租户启停/删除相互交错。
func lifecycleLock(ctx context.Context, s *svc.ServiceContext, key string) (func(), error) {
	lock := redis.NewRedisLock(s.Redis, "tenant:lifecycle:"+key)
	lock.SetExpire(600)
	ok, err := lock.AcquireCtx(ctx)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if !ok {
		return nil, xerr.NewErrCodeMsg(400, "租户操作正在进行，请稍后重试")
	}
	return func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, _ = lock.ReleaseCtx(releaseCtx)
	}, nil
}

func managedTenant(ctx context.Context, s *svc.ServiceContext, actor *tenantpb.Actor, id int64) (*model.TenantRegistry, func(), error) {
	if actor == nil || actor.UserID <= 0 {
		return nil, nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if id <= 0 {
		return nil, nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	tenant, err := s.TenantRegistryModel.FindOne(ctx, uint64(id))
	if errors.Is(err, sqlx.ErrNotFound) {
		return nil, nil, xerr.NewErrCode(xerr.NOT_FOUND)
	}
	if err != nil {
		return nil, nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if tenant.DeletedAt.Valid {
		return nil, nil, xerr.NewErrCode(xerr.NOT_FOUND)
	}
	if !slugPattern.MatchString(tenant.Slug) || tenant.ComposeProject != "gozero-t-"+tenant.Slug || tenant.Domain != tenant.Slug+"."+s.Config.Tenant.DomainSuffix {
		return nil, nil, xerr.NewErrCodeMsg(400, "此记录不是本系统创建的租户栈，禁止生命周期操作")
	}
	release, err := lifecycleLock(ctx, s, tenant.Slug)
	if err != nil {
		return nil, nil, err
	}
	tenant, err = s.TenantRegistryModel.FindOne(ctx, uint64(id))
	if err != nil || tenant.DeletedAt.Valid {
		release()
		return nil, nil, xerr.NewErrCode(xerr.NOT_FOUND)
	}
	return tenant, release, nil
}

func tenantEnvPath(s *svc.ServiceContext, tenant *model.TenantRegistry) string {
	return filepath.Join(s.Config.Tenant.StateDir, tenant.Slug+".env")
}
func compose(ctx context.Context, s *svc.ServiceContext, tenant *model.TenantRegistry, input io.Reader, args ...string) ([]byte, error) {
	if !slugPattern.MatchString(tenant.Slug) || tenant.ComposeProject != "gozero-t-"+tenant.Slug {
		return nil, errors.New("invalid tenant project")
	}
	envPath := tenantEnvPath(s, tenant)
	data, err := os.ReadFile(envPath)
	if err != nil || !strings.HasPrefix(string(data), "TENANT_SLUG="+tenant.Slug+"\n") {
		return nil, errors.New("tenant state unavailable")
	}
	base := []string{"compose", "-p", tenant.ComposeProject, "-f", filepath.Join(s.Config.Tenant.StackTemplateDir, "docker-compose.tenant.yml"), "--env-file", envPath}
	return dockerCommand(ctx, input, append(base, args...)...)
}

// 不把命令输出或底层错误返回用户/日志，其中可能包含配置和凭据。
func dockerCommand(ctx context.Context, input io.Reader, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = input
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// 失败时只取 stderr：docker 成功时也可能往 stderr 打进度/警告，
		// 不能混入返回值影响调用方的输出判断（残留预检、初始化结果校验）。
		detail := strings.TrimSpace(stderr.String())
		if detail == "" {
			detail = err.Error()
		}
		return nil, fmt.Errorf("docker %s: %s", args[0], detail)
	}
	return stdout.Bytes(), nil
}
func tenantSQL(ctx context.Context, s *svc.ServiceContext, tenant *model.TenantRegistry, input io.Reader) ([]byte, error) {
	return compose(ctx, s, tenant, input, "exec", "-T", "mysql", "sh", "-c", "MYSQL_PWD=\"$MYSQL_ROOT_PASSWORD\" exec mysql -uroot --batch --skip-column-names --default-character-set=utf8mb4")
}
func waitTenantReady(ctx context.Context, s *svc.ServiceContext, tenant *model.TenantRegistry) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		output, err := compose(ctx, s, tenant, nil, "exec", "-T", "api", "wget", "-q", "-O", "-", "http://127.0.0.1:7001/ready")
		var response struct{ Code int }
		if err == nil && json.Unmarshal(output, &response) == nil && response.Code == 200 {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("tenant is not ready")
		case <-ticker.C:
		}
	}
}
func routeTenant(ctx context.Context, s *svc.ServiceContext, tenant *model.TenantRegistry, enabled bool) error {
	if enabled && (tenant.Domain == "localhost" || tenant.Domain == "platform.localhost") {
		return errors.New("tenant domain conflicts with platform host")
	}
	// 下划线不在租户 slug 白名单中，避免全局锁与合法租户重名。
	release, err := lifecycleLock(ctx, s, "_nginx_routes")
	if err != nil {
		return err
	}
	defer release()
	path := filepath.Join(s.Config.Tenant.NginxConfDir, tenant.ComposeProject+".conf")
	previous, readErr := os.ReadFile(path)
	if readErr != nil && !os.IsNotExist(readErr) {
		return errors.New("cannot read tenant route")
	}
	if enabled {
		template, err := os.ReadFile(filepath.Join(s.Config.Tenant.StackTemplateDir, "tenant.conf.template"))
		if err != nil {
			return errors.New("route template unavailable")
		}
		text := strings.NewReplacer("$"+"{DOMAIN}", tenant.Domain, "$"+"{SLUG}", tenant.Slug).Replace(string(template))
		if err = os.WriteFile(path, []byte(text), 0644); err != nil {
			return errors.New("cannot write tenant route")
		}
	} else if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
		return errors.New("cannot remove tenant route")
	}
	rollback := func() {
		if readErr == nil {
			_ = os.WriteFile(path, previous, 0644)
		} else {
			_ = os.Remove(path)
		}
	}
	if _, err = dockerCommand(ctx, nil, "exec", s.Config.Tenant.NginxContainer, "nginx", "-t"); err != nil {
		rollback()
		return errors.New("nginx validation failed")
	}
	if _, err = dockerCommand(ctx, nil, "exec", s.Config.Tenant.NginxContainer, "nginx", "-s", "reload"); err != nil {
		rollback()
		return errors.New("nginx reload failed")
	}
	return nil
}
func tenantFailure(s *svc.ServiceContext, tenant *model.TenantRegistry, message string, err error) error {
	if err != nil {
		message = fmt.Sprintf("%s: %v", message, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.TenantRegistryModel.SetState(ctx, tenant, "error", message, tenant.AdminAccountReady == 1); err != nil {
		logx.Error("tenant failure state could not be persisted")
	}
	return xerr.NewErrCodeMsg(xerr.SERVER_COMMON_ERROR, message)
}

// reportStep 更新 registry 的当前步骤。不接收调用方 ctx：provisioning 取消/超时后步骤汇报仍要尽力送达。
func reportStep(s *svc.ServiceContext, tenant *model.TenantRegistry, step string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.TenantRegistryModel.SetStep(ctx, tenant, step); err != nil {
		if errors.Is(err, model.ErrStepCacheInvalidation) {
			logx.Errorf("tenant step %s write succeeded but cache invalidation failed", step)
			return
		}
		logx.Errorf("tenant step %s could not be written", step)
	}
}
