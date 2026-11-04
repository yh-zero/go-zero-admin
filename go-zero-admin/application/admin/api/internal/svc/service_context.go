package svc

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/go-playground/validator/v10"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"

	"go-zero-admin/application/admin/api/internal/config"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	apiserviceclient "go-zero-admin/application/admin/rpc/client/apiservice"
	auditserviceclient "go-zero-admin/application/admin/rpc/client/auditservice"
	authserviceclient "go-zero-admin/application/admin/rpc/client/authservice"
	casbinserviceclient "go-zero-admin/application/admin/rpc/client/casbinservice"
	deptserviceclient "go-zero-admin/application/admin/rpc/client/deptservice"
	dictserviceclient "go-zero-admin/application/admin/rpc/client/dictservice"
	healthserviceclient "go-zero-admin/application/admin/rpc/client/healthservice"
	menuserviceclient "go-zero-admin/application/admin/rpc/client/menuservice"
	profileserviceclient "go-zero-admin/application/admin/rpc/client/profileservice"
	roleserviceclient "go-zero-admin/application/admin/rpc/client/roleservice"
	userserviceclient "go-zero-admin/application/admin/rpc/client/userservice"
	"go-zero-admin/application/ai/rpc/aipb"
	aiserviceclient "go-zero-admin/application/ai/rpc/client/aiservice"
	"go-zero-admin/application/app/rpc/apppb"
	appadminserviceclient "go-zero-admin/application/app/rpc/client/appadminservice"
	appauthserviceclient "go-zero-admin/application/app/rpc/client/appauthservice"
	fileserviceclient "go-zero-admin/application/file/rpc/client/fileservice"
	"go-zero-admin/application/file/rpc/filepb"
	tenantserviceclient "go-zero-admin/application/tenant/rpc/client/tenantservice"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/casbinauth"
	"go-zero-admin/pkg/jwtx"
)

// PolicySyncChannel Redis 通知频道（admin-rpc 变更策略时发布）
const PolicySyncChannel = "casbin:policy:changed"

type ServiceContext struct {
	Config      config.Config
	AdminRpc    zrpc.Client
	FileRpcConn zrpc.Client
	Enforcer    *casbinauth.Holder
	AuthRpc     authserviceclient.AuthService
	UserRpc     userserviceclient.UserService
	DeptRpc     deptserviceclient.DeptService
	RoleRpc     roleserviceclient.RoleService
	FileRpc     fileserviceclient.FileService
	CasbinRpc   casbinserviceclient.CasbinService
	MenuRpc     menuserviceclient.MenuService
	DictRpc     dictserviceclient.DictService
	ApiRpc      apiserviceclient.ApiService
	AuditRpc    auditserviceclient.AuditService
	ProfileRpc  profileserviceclient.ProfileService
	Redis       *goredis.Client
	Sessions    *jwtx.SessionStore
	HealthRpc   healthserviceclient.HealthService
	ApiRoutes   []*adminpb.ApiRoute
	TenantRpc   tenantserviceclient.TenantService
	AppAuthRpc  appauthserviceclient.AppAuthService
	AppAdminRpc appadminserviceclient.AppAdminService
	AiRpc       aiserviceclient.AiService
	// Validator 请求参数校验器：logic 入口调用 StructCtx，校验规则见 .api 类型的 validate 标签
	Validator *validator.Validate
}

func NewServiceContext(c config.Config) *ServiceContext {
	ignoreRPCContent(adminpb.File_application_admin_rpc_desc_admin_proto)
	ignoreRPCContent(filepb.File_application_file_rpc_desc_file_proto)
	adminRpc := zrpc.MustNewClient(c.AdminRpc)
	fileRpcConn := zrpc.MustNewClient(c.FileRpc, zrpc.WithDialOption(grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(12<<20), grpc.MaxCallRecvMsgSize(12<<20))))

	redisOptions := &goredis.Options{Addr: c.Redis.Host, Username: c.Redis.User, Password: c.Redis.Pass}
	if c.Redis.Tls {
		redisOptions.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	rdb := goredis.NewClient(redisOptions)

	casbinSvc := casbinserviceclient.NewCasbinService(adminRpc)

	// 启动时拉全量策略
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := casbinSvc.ListPolicies(ctx, &adminpb.ListPoliciesRequest{})
	if err != nil {
		panic("load casbin policies failed: " + err.Error())
	}
	rules := make([]casbinauth.PolicyRule, 0, len(resp.Rules))
	for _, r := range resp.Rules {
		rules = append(rules, casbinauth.PolicyRule{Ptype: r.Ptype, V0: r.V0, V1: r.V1, V2: r.V2, V3: r.V3, V4: r.V4, V5: r.V5})
	}
	holder, err := casbinauth.NewHolder(rules)
	if err != nil {
		panic("create enforcer failed: " + err.Error())
	}

	svcCtx := &ServiceContext{
		Config:      c,
		AdminRpc:    adminRpc,
		FileRpcConn: fileRpcConn,
		HealthRpc:   healthserviceclient.NewHealthService(adminRpc),
		Sessions:    jwtx.NewSessionStore(rdb, "admin"),
		Enforcer:    holder,
		AuthRpc:     authserviceclient.NewAuthService(adminRpc),
		UserRpc:     userserviceclient.NewUserService(adminRpc),
		DeptRpc:     deptserviceclient.NewDeptService(adminRpc),
		RoleRpc:     roleserviceclient.NewRoleService(adminRpc),
		FileRpc:     fileserviceclient.NewFileService(fileRpcConn),
		CasbinRpc:   casbinSvc,
		MenuRpc:     menuserviceclient.NewMenuService(adminRpc),
		DictRpc:     dictserviceclient.NewDictService(adminRpc),
		ApiRpc:      apiserviceclient.NewApiService(adminRpc),
		AuditRpc:    auditserviceclient.NewAuditService(adminRpc),
		ProfileRpc:  profileserviceclient.NewProfileService(adminRpc),
		Redis:       rdb,
		Validator:   NewValidator(),
	}

	if c.TenantRpc != nil {
		ignoreRPCContent(tenantpb.File_application_tenant_rpc_desc_tenant_proto)
		svcCtx.TenantRpc = tenantserviceclient.NewTenantService(zrpc.MustNewClient(*c.TenantRpc))
	}
	if c.AppRpc != nil {
		ignoreRPCContent(apppb.File_application_app_rpc_desc_app_proto)
		client := zrpc.MustNewClient(*c.AppRpc)
		svcCtx.AppAuthRpc = appauthserviceclient.NewAppAuthService(client)
		svcCtx.AppAdminRpc = appadminserviceclient.NewAppAdminService(client)
	}
	if c.AiRpc != nil {
		ignoreRPCContent(aipb.File_application_ai_rpc_desc_ai_proto)
		svcCtx.AiRpc = aiserviceclient.NewAiService(zrpc.MustNewClient(*c.AiRpc))
	}
	go svcCtx.watchPolicyChanges()

	return svcCtx
}

func ignoreRPCContent(descriptor protoreflect.FileDescriptor) {
	services := descriptor.Services()
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			zrpc.DontLogClientContentForMethod("/" + string(service.FullName()) + "/" + string(service.Methods().Get(j).Name()))
		}
	}
}

// watchPolicyChanges 订阅 Redis 频道刷新策略；pub/sub 丢失消息不可恢复，另有定时全量兜底
func (s *ServiceContext) watchPolicyChanges() {
	for {
		pubSub := s.Redis.Subscribe(context.Background(), PolicySyncChannel)
		ch := pubSub.Channel()
		ticker := time.NewTicker(60 * time.Second)
	Loop:
		for {
			select {
			case _, ok := <-ch:
				if !ok {
					// 订阅通道断开（Redis 重启等），退避后重订阅
					logx.Error("casbin policy channel closed, resubscribing")
					break Loop
				}
				s.refreshPolicies()
			case <-ticker.C:
				s.refreshPolicies()
			}
		}
		ticker.Stop()
		_ = pubSub.Close()
		time.Sleep(5 * time.Second)
	}
}

// refreshPolicies 全量拉取并原子替换 enforcer；失败保留旧实例
func (s *ServiceContext) refreshPolicies() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := s.CasbinRpc.ListPolicies(ctx, &adminpb.ListPoliciesRequest{})
	if err != nil {
		logx.Errorf("refresh policies error: %v", err)
		return
	}
	rules := make([]casbinauth.PolicyRule, 0, len(resp.Rules))
	for _, r := range resp.Rules {
		rules = append(rules, casbinauth.PolicyRule{Ptype: r.Ptype, V0: r.V0, V1: r.V1, V2: r.V2, V3: r.V3, V4: r.V4, V5: r.V5})
	}
	if err := s.Enforcer.Swap(rules); err != nil {
		logx.Errorf("rebuild enforcer error: %v", err)
		return
	}
	logx.Info("casbin policies refreshed")
}
