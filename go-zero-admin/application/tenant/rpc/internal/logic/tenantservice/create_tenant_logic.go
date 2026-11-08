package tenantservicelogic

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/tenant/rpc/internal/model"
	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/jwtx"
	"go-zero-admin/pkg/objectstorage"
	"go-zero-admin/pkg/xerr"
	"golang.org/x/crypto/bcrypt"
)

type CreateTenantLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateTenantLogic(ctx context.Context, s *svc.ServiceContext) *CreateTenantLogic {
	return &CreateTenantLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}

func (l *CreateTenantLogic) CreateTenant(in *tenantpb.CreateTenantRequest) (*tenantpb.CreateTenantResponse, error) {
	if in.Actor == nil || in.Actor.UserID <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	slug := strings.ToLower(strings.TrimSpace(in.Slug))
	if !slugPattern.MatchString(slug) {
		return nil, xerr.NewErrCodeMsg(400, "租户标识须为 2-32 位小写字母、数字、连字符，首尾不能为连字符")
	}
	if slug == "platform" {
		return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "租户标识与平台保留标识冲突")
	}
	username := strings.TrimSpace(in.AdminUsername)
	if username == "" {
		username = "admin"
	}
	if !regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9_]{2,31}$").MatchString(username) {
		return nil, xerr.NewErrCodeMsg(400, "管理员用户名须为 3-32 位字母、数字或下划线，以字母开头")
	}
	password := in.AdminPassword
	if password == "" {
		random, err := jwtx.RandomToken()
		if err != nil {
			return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
		password = random[:24]
	}
	if len(password) < 8 || len(password) > 72 {
		return nil, xerr.NewErrCodeMsg(400, "管理员密码须为 8-72 字节")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	suffix := l.svcCtx.Config.Tenant.DomainSuffix
	if !domainPattern.MatchString(suffix) || strings.Contains(suffix, "..") {
		return nil, xerr.NewErrCodeMsg(500, "租户域名配置不正确")
	}
	release, err := lifecycleLock(l.ctx, l.svcCtx, slug)
	if err != nil {
		return nil, err
	}
	handedOff := false
	defer func() {
		if !handedOff {
			release()
		}
	}()
	existing, err := l.svcCtx.TenantRegistryModel.FindOneBySlug(l.ctx, slug)
	if err == nil && !existing.DeletedAt.Valid {
		return nil, xerr.NewErrCodeMsg(400, "租户标识已存在")
	}
	if err != nil && !errors.Is(err, sqlx.ErrNotFound) {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	tenant := &model.TenantRegistry{Slug: slug, Domain: slug + "." + suffix, ComposeProject: "gozero-t-" + slug, Status: "creating", CreatedBy: sql.NullInt64{Int64: in.Actor.UserID, Valid: true}, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if existing != nil { // 复活软删记录：复用 id 保持审计连续，栈资源已全清
		tenant.Id = existing.Id
		tenant.CreatedAt = existing.CreatedAt
	}
	if err = checkTenantResources(l.ctx, l.svcCtx, tenant); err != nil {
		return nil, xerr.NewErrCodeMsg(400, "租户资源已存在或 Docker 运行环境不可用")
	}
	tenant.MysqlRootPassword, err = jwtx.RandomToken()
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	tenant.JwtAccessSecret, err = jwtx.RandomToken()
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	tenant.AppJwtAccessSecret, err = jwtx.RandomToken()
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if existing != nil {
		// 软删复活：全字段重置（密钥/状态/域名），deleted_at 置空
		if err = l.svcCtx.TenantRegistryModel.Update(l.ctx, tenant); err != nil {
			return nil, xerr.NewErrCode(xerr.DB_ERROR)
		}
	} else {
		result, err := l.svcCtx.TenantRegistryModel.Insert(l.ctx, tenant)
		if err != nil {
			return nil, xerr.NewErrCode(xerr.DB_ERROR)
		}
		id, err := result.LastInsertId()
		if err != nil {
			return nil, xerr.NewErrCode(xerr.DB_ERROR)
		}
		tenant.Id = uint64(id)
	}
	response := mapTenant(tenant)
	handedOff = true
	go func() {
		defer release()
		defer func() {
			if recover() != nil {
				_ = tenantFailure(l.svcCtx, tenant, "创建租户异常中断", nil)
			}
		}()
		l.provisionTenant(tenant, username, string(hash))
	}()
	return &tenantpb.CreateTenantResponse{Tenant: response, AdminPassword: password}, nil
}

func checkTenantResources(ctx context.Context, s *svc.ServiceContext, tenant *model.TenantRegistry) error {
	if _, err := os.Stat(tenantEnvPath(s, tenant)); err == nil {
		return errors.New("tenant state already exists")
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, args := range [][]string{
		{"ps", "-aq", "--filter", "label=com.docker.compose.project=" + tenant.ComposeProject},
		{"volume", "ls", "-q", "--filter", "label=com.docker.compose.project=" + tenant.ComposeProject},
		{"network", "ls", "-q", "--filter", "label=com.docker.compose.project=" + tenant.ComposeProject},
	} {
		output, err := dockerCommand(ctx, nil, args...)
		if err != nil || strings.TrimSpace(string(output)) != "" {
			return errors.New("tenant resources unavailable")
		}
	}
	_, err := dockerCommand(ctx, nil, "image", "inspect", "gozero-platform-api", "gozero-platform-admin-rpc", "gozero-platform-app-rpc", "gozero-platform-file-rpc", "gozero-platform-ai-rpc", "mysql:9.7", "redis:8", "--format", "{{.Id}}")
	return err
}

func (l *CreateTenantLogic) provisionTenant(tenant *model.TenantRegistry, username, passwordHash string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	storage, err := tenantStorage(l.svcCtx, tenant)
	if err != nil {
		_ = tenantFailure(l.svcCtx, tenant, "租户对象存储配置不可用", err)
		return
	}
	if err = renderTenantEnv(l.svcCtx, tenant, storage); err != nil {
		_ = tenantFailure(l.svcCtx, tenant, "写入租户运行配置失败", err)
		reportStep(l.svcCtx, tenant, "render_env")
		return
	}
	reportStep(l.svcCtx, tenant, "render_env")
	if _, err = compose(ctx, l.svcCtx, tenant, nil, "up", "-d", "--no-build", "--pull", "never", "--wait", "--wait-timeout", "90", "mysql", "redis"); err != nil {
		_ = tenantFailure(l.svcCtx, tenant, "启动租户数据库失败", err)
		reportStep(l.svcCtx, tenant, "start_db")
		return
	}
	reportStep(l.svcCtx, tenant, "start_db")
	if err = initializeTenant(ctx, l.svcCtx, tenant, username, passwordHash); err != nil {
		_ = tenantFailure(l.svcCtx, tenant, "初始化租户数据库失败", err)
		reportStep(l.svcCtx, tenant, "init_db")
		return
	}
	reportStep(l.svcCtx, tenant, "init_db")
	if err = createTenantBuckets(ctx, storage, tenant); err != nil {
		_ = tenantFailure(l.svcCtx, tenant, "创建租户存储空间失败", err)
		reportStep(l.svcCtx, tenant, "create_buckets")
		return
	}
	reportStep(l.svcCtx, tenant, "create_buckets")
	// 初始化成功后保留 ready 标记；服务启动失败可以修复环境后再次 start。
	if err = l.svcCtx.TenantRegistryModel.SetState(ctx, tenant, "creating", "", true); err != nil {
		_ = tenantFailure(l.svcCtx, tenant, "保存租户初始化状态失败", err)
		return
	}
	tenant.AdminAccountReady = 1
	if _, err = compose(ctx, l.svcCtx, tenant, nil, "up", "-d", "--no-build", "--pull", "never", "--wait", "--wait-timeout", "60"); err != nil {
		_ = tenantFailure(l.svcCtx, tenant, "启动租户应用失败", err)
		reportStep(l.svcCtx, tenant, "start_apps")
		return
	}
	reportStep(l.svcCtx, tenant, "start_apps")
	if err = waitTenantReady(ctx, l.svcCtx, tenant); err != nil {
		_ = tenantFailure(l.svcCtx, tenant, "租户应用依赖未就绪", err)
		reportStep(l.svcCtx, tenant, "wait_ready")
		return
	}
	reportStep(l.svcCtx, tenant, "wait_ready")
	if err = routeTenant(ctx, l.svcCtx, tenant, true); err != nil {
		_ = tenantFailure(l.svcCtx, tenant, "生成租户路由失败", err)
		reportStep(l.svcCtx, tenant, "route")
		return
	}
	reportStep(l.svcCtx, tenant, "route")
	if err = l.svcCtx.TenantRegistryModel.SetState(ctx, tenant, "active", "", true); err != nil {
		_ = tenantFailure(l.svcCtx, tenant, "保存租户就绪状态失败", err)
	}
}

func renderTenantEnv(s *svc.ServiceContext, t *model.TenantRegistry, storage objectstorage.Config) error {
	cfg := s.Config.Tenant
	if strings.ContainsAny(cfg.EdgeNetwork, "\r\n=") {
		return errors.New("invalid edge network")
	}
	dsn := "root:" + t.MysqlRootPassword + "@tcp(mysql:3306)/gozero_admin?parseTime=true&charset=utf8mb4&loc=Asia%2FShanghai"
	cache := []map[string]any{{"Host": "redis:6379", "Type": "node"}}
	rpc := func(name string, port int) map[string]any {
		return map[string]any{"Name": name, "ListenOn": fmt.Sprintf("0.0.0.0:%d", port), "Mysql": map[string]any{"DataSource": dsn}, "CacheRedis": cache}
	}
	admin := rpc("admin-rpc", 6001)
	admin["JwtAuth"] = map[string]any{"AccessSecret": t.JwtAccessSecret, "AccessExpire": 7200, "RefreshExpire": 2592000}
	admin["Captcha"] = map[string]any{"Enabled": true, "ExpireSeconds": 120}
	admin["FileRpc"] = map[string]any{"Endpoints": []string{"file-rpc:6003"}, "NonBlock": true, "Timeout": 1500}
	app := rpc("app-rpc", 6002)
	app["Timeout"] = 12000
	app["JwtAuth"] = map[string]any{"AccessSecret": t.AppJwtAccessSecret, "AccessExpire": 900}
	app["FileRpc"] = map[string]any{"Endpoints": []string{"file-rpc:6003"}, "NonBlock": true, "Timeout": 1500}
	file := rpc("file-rpc", 6003)
	file["Timeout"] = 35000
	file["Storage"] = storage
	ai := rpc("ai-rpc", 6004)
	ai["Timeout"] = 120000
	ai["AI"] = map[string]any{"Enabled": false}
	api := map[string]any{"Name": "admin-api", "Host": "0.0.0.0", "Port": 7001,
		"AdminRpc":       map[string]any{"Endpoints": []string{t.Slug + "-admin-rpc:6001"}},
		"FileRpc":        map[string]any{"Endpoints": []string{t.Slug + "-file-rpc:6003"}, "Timeout": 40000},
		"AppRpc":         map[string]any{"Endpoints": []string{"app-rpc:6002"}, "NonBlock": true, "Timeout": 15000},
		"AiRpc":          map[string]any{"Endpoints": []string{"ai-rpc:6004"}, "NonBlock": true, "Timeout": 120000},
		"Redis":          map[string]any{"Host": "redis:6379", "Type": "node"},
		"TrustedProxies": cfg.TrustedProxies,
		"Auth":           map[string]any{"AccessSecret": t.JwtAccessSecret, "AccessExpire": 7200}}
	configs := []struct {
		name  string
		value any
	}{{"API_CONFIG_B64", api}, {"ADMIN_CONFIG_B64", admin}, {"APP_CONFIG_B64", app}, {"FILE_CONFIG_B64", file}, {"AI_CONFIG_B64", ai}}
	var output strings.Builder
	fmt.Fprintf(&output, "TENANT_SLUG=%s\nMYSQL_ROOT_PASSWORD=%s\nEDGE_NETWORK=%s\n", t.Slug, t.MysqlRootPassword, cfg.EdgeNetwork)
	for _, entry := range configs {
		data, err := json.Marshal(entry.value)
		if err != nil {
			return err
		}
		fmt.Fprintf(&output, "%s=%s\n", entry.name, base64.StdEncoding.EncodeToString(data))
	}
	if err := os.MkdirAll(cfg.StateDir, 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(tenantEnvPath(s, t), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.WriteString(f, output.String())
	return err
}

func initializeTenant(ctx context.Context, s *svc.ServiceContext, t *model.TenantRegistry, username, passwordHash string) error {
	f, err := os.Open(s.Config.Tenant.InitSQLPath)
	if err != nil {
		return err
	}
	defer f.Close()
	return model.InitializeTenantDatabase(ctx, f, username, passwordHash, func(ctx context.Context, input io.Reader) ([]byte, error) {
		return tenantSQL(ctx, s, t, input)
	})
}

func tenantStorage(s *svc.ServiceContext, t *model.TenantRegistry) (objectstorage.Config, error) {
	var config struct{ Storage objectstorage.Config }
	if err := conf.Load(s.Config.Tenant.StorageConfigPath, &config, conf.UseEnv()); err != nil {
		return objectstorage.Config{}, errors.New("storage config unavailable")
	}
	config.Storage.PublicBucket = "gozero-t-" + t.Slug + "-public"
	config.Storage.PrivateBucket = "gozero-t-" + t.Slug + "-private"
	if _, err := objectstorage.New(config.Storage); err != nil {
		return objectstorage.Config{}, errors.New("storage config invalid")
	}
	return config.Storage, nil
}
func tenantS3(config objectstorage.Config) *s3.Client {
	cfg := aws.Config{Region: config.Region, Credentials: credentials.NewStaticCredentialsProvider(config.AccessKeyID, config.SecretAccessKey, ""), HTTPClient: &http.Client{Timeout: 15 * time.Second}, RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired, ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired}
	return s3.NewFromConfig(cfg, func(o *s3.Options) { o.BaseEndpoint = aws.String(config.Endpoint); o.UsePathStyle = true })
}
func missingBucket(err error) bool {
	var apiError interface{ ErrorCode() string }
	return errors.As(err, &apiError) && (apiError.ErrorCode() == "NotFound" || apiError.ErrorCode() == "NoSuchBucket" || apiError.ErrorCode() == "404")
}
func createTenantBuckets(ctx context.Context, config objectstorage.Config, t *model.TenantRegistry) error {
	client := tenantS3(config)
	for _, bucket := range []string{config.PublicBucket, config.PrivateBucket} {
		_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(bucket)})
		if err == nil || !missingBucket(err) {
			return errors.New("tenant bucket already exists or unavailable")
		}
	}
	for _, bucket := range []string{config.PublicBucket, config.PrivateBucket} {
		if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
			return errors.New("cannot create tenant bucket")
		}
		tags := []s3types.Tag{{Key: aws.String("TenantProject"), Value: aws.String(t.ComposeProject)}, {Key: aws.String("TenantRegistryId"), Value: aws.String(fmt.Sprint(t.Id))}}
		if _, err := client.PutBucketTagging(ctx, &s3.PutBucketTaggingInput{Bucket: aws.String(bucket), Tagging: &s3types.Tagging{TagSet: tags}}); err != nil {
			// 该桶刚由当前调用创建且尚未写入对象，标记失败时立即回收，避免留下无法证明归属的桶。
			_, _ = client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)})
			return errors.New("cannot mark tenant bucket")
		}
	}
	policy := map[string]any{"Version": "2012-10-17", "Statement": []any{map[string]any{"Effect": "Allow", "Principal": "*", "Action": []string{"s3:GetObject"}, "Resource": []string{"arn:aws:s3:::" + config.PublicBucket + "/*"}}}}
	data, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	_, err = client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{Bucket: aws.String(config.PublicBucket), Policy: aws.String(string(data))})
	if err != nil {
		return errors.New("cannot set tenant bucket policy")
	}
	return nil
}
