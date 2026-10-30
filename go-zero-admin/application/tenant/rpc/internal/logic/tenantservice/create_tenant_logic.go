package tenantservicelogic

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/xerr"
)

type CreateTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateTenantLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateTenantLogic {
	return &CreateTenantLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateTenantLogic) CreateTenant(in *tenantpb.CreateTenantRequest) (*tenantpb.CreateTenantResponse, error) {
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if !regexp.MustCompile("^[a-z0-9-]{2,32}$").MatchString(slug) {
		return nil, xerr.NewErrCodeMsg(400, "租户标识只能含小写字母/数字/连字符，2-32 位")
	}
	_, err := l.svcCtx.TenantRegistryModel.FindOneBySlug(l.ctx, slug)
	if err == nil {
		return nil, xerr.NewErrCodeMsg(400, "租户标识已存在")
	}
	domain := slug + "." + l.svcCtx.Config.Tenant.DomainSuffix
	composeProject := "gozero-t-" + slug
	adminPassword := in.AdminPassword
	if adminPassword == "" {
		adminPassword = generatePassword(16)
	}
	jwtSecret := generatePassword(32)
	appJwtSecret := generatePassword(32)
	mysqlRoot := generatePassword(24)
	now := time.Now()
	result, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"INSERT INTO tenant_registry (slug, domain, compose_project, status, mysql_root_password, jwt_access_secret, app_jwt_access_secret, created_by, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?)",
		slug, domain, composeProject, "creating", mysqlRoot, jwtSecret, appJwtSecret, in.Actor.UserID, now, now)
	if err != nil {
		l.Errorf("insert tenant_registry error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	id, _ := result.LastInsertId()
	go l.provisionTenant(id, slug, composeProject, mysqlRoot, jwtSecret, appJwtSecret)
	return &tenantpb.CreateTenantResponse{
		Tenant: &tenantpb.TenantInfo{
			Id: id, Slug: slug, Domain: domain, ComposeProject: composeProject,
			Status: "creating", CreatedBy: in.Actor.UserID, CreatedAt: now.Unix(), UpdatedAt: now.Unix(),
		},
		AdminPassword: adminPassword,
	}, nil
}

// provisionTenant 渲染模板 + 拉起租户栈 + 生成 Nginx 配置（后台执行）
func (l *CreateTenantLogic) provisionTenant(tenantID int64, slug, composeProject, mysqlRoot, jwtSecret, appJwtSecret string) {
	ctx := context.Background()
	cfg := l.svcCtx.Config.Tenant
	stackDir := cfg.StackTemplateDir

	// 1. 渲染 env 模板
	envContent, err := os.ReadFile(filepath.Join(stackDir, "env.template"))
	if err != nil {
		l.failProvision(tenantID, "读取 env 模板失败: "+err.Error())
		return
	}
	replacer := strings.NewReplacer(
		"${SLUG}", slug,
		"${MYSQL_ROOT_PASSWORD}", mysqlRoot,
		"${JWT_ACCESS_SECRET}", jwtSecret,
		"${APP_JWT_ACCESS_SECRET}", appJwtSecret,
	)
	env := replacer.Replace(string(envContent))
	envPath := filepath.Join(stackDir, ".env."+slug)
	if err := os.WriteFile(envPath, []byte(env), 0600); err != nil {
		l.failProvision(tenantID, "写 env 失败: "+err.Error())
		return
	}

	// 2. 拉起租户栈
	cmd := exec.Command("docker", "compose", "-p", composeProject,
		"-f", filepath.Join(stackDir, "docker-compose.tenant.yml"),
		"--env-file", envPath, "up", "-d", "--build", "--wait")
	if out, err := cmd.CombinedOutput(); err != nil {
		l.failProvision(tenantID, fmt.Sprintf("compose up 失败: %v %s", err, out))
		return
	}

	// 3. 生成 Nginx server 块
	confContent, err := os.ReadFile(filepath.Join(stackDir, "tenant.conf.template"))
	if err != nil {
		l.failProvision(tenantID, "读取 nginx 模板失败: "+err.Error())
		return
	}
	conf := strings.NewReplacer("${SLUG}", slug).Replace(string(confContent))
	confPath := filepath.Join(cfg.NginxConfDir, slug+".conf")
	if err := os.WriteFile(confPath, []byte(conf), 0644); err != nil {
		l.failProvision(tenantID, "写 nginx 配置失败: "+err.Error())
		return
	}

	// 4. 通知 nginx reload
	reload := exec.Command("docker", "exec", "gozero-nginx", "nginx", "-s", "reload")
	_ = reload.Run()

	// 5. 标记就绪
	_, _ = l.svcCtx.DB.ExecCtx(ctx,
		"UPDATE tenant_registry SET status='active', admin_account_ready=1, updated_at=NOW() WHERE id=?", tenantID)
}

// failProvision 记录失败原因
func (l *CreateTenantLogic) failProvision(tenantID int64, msg string) {
	l.Errorf("tenant %d provisioning failed: %s", tenantID, msg)
	_, _ = l.svcCtx.DB.ExecCtx(context.Background(),
		"UPDATE tenant_registry SET status='error', error_msg=?, updated_at=NOW() WHERE id=?", msg, tenantID)
}

// generatePassword 生成随机密码（大小写字母+数字）
func generatePassword(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	_, _ = rand.Read(b)
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}
