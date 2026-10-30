package svc

import (
	"context"
	"time"

	"github.com/casbin/casbin/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"

	"go-zero-admin/application/admin/api/internal/config"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	apiserviceclient "go-zero-admin/application/admin/rpc/client/apiservice"
	auditserviceclient "go-zero-admin/application/admin/rpc/client/auditservice"
	authserviceclient "go-zero-admin/application/admin/rpc/client/authservice"
	casbinserviceclient "go-zero-admin/application/admin/rpc/client/casbinservice"
	deptserviceclient "go-zero-admin/application/admin/rpc/client/deptservice"
	dictserviceclient "go-zero-admin/application/admin/rpc/client/dictservice"
	menuserviceclient "go-zero-admin/application/admin/rpc/client/menuservice"
	profileserviceclient "go-zero-admin/application/admin/rpc/client/profileservice"
	roleserviceclient "go-zero-admin/application/admin/rpc/client/roleservice"
	userserviceclient "go-zero-admin/application/admin/rpc/client/userservice"
	fileserviceclient "go-zero-admin/application/file/rpc/client/fileservice"
	"go-zero-admin/pkg/casbinauth"
)

// PolicySyncChannel Redis 通知频道（admin-rpc 变更策略时发布）
const PolicySyncChannel = "casbin:policy:changed"

type ServiceContext struct {
	Config      config.Config
	AdminRpc    zrpc.Client
	FileRpcConn zrpc.Client
	Enforcer    *casbin.CachedEnforcer
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
}

func NewServiceContext(c config.Config) *ServiceContext {
	adminRpc := zrpc.MustNewClient(c.AdminRpc)
	fileRpcConn := zrpc.MustNewClient(c.FileRpc)

	rdb := goredis.NewClient(&goredis.Options{Addr: c.Redis.Host})

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
	enforcer, err := casbinauth.NewEnforcer(rules)
	if err != nil {
		panic("create enforcer failed: " + err.Error())
	}

	svcCtx := &ServiceContext{
		Config:     c,
		AdminRpc:   adminRpc,
		Enforcer:   enforcer,
		AuthRpc:    authserviceclient.NewAuthService(adminRpc),
		UserRpc:    userserviceclient.NewUserService(adminRpc),
		DeptRpc:    deptserviceclient.NewDeptService(adminRpc),
		RoleRpc:    roleserviceclient.NewRoleService(adminRpc),
		FileRpc:    fileserviceclient.NewFileService(fileRpcConn),
		CasbinRpc:  casbinSvc,
		MenuRpc:    menuserviceclient.NewMenuService(adminRpc),
		DictRpc:    dictserviceclient.NewDictService(adminRpc),
		ApiRpc:     apiserviceclient.NewApiService(adminRpc),
		AuditRpc:   auditserviceclient.NewAuditService(adminRpc),
		ProfileRpc: profileserviceclient.NewProfileService(adminRpc),
		Redis:      rdb,
	}

	go svcCtx.watchPolicyChanges()

	return svcCtx
}

// watchPolicyChanges 订阅 Redis 频道，策略变更时刷新 enforcer
func (s *ServiceContext) watchPolicyChanges() {
	pubSub := s.Redis.Subscribe(context.Background(), PolicySyncChannel)
	ch := pubSub.Channel()
	for range ch {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		resp, err := s.CasbinRpc.ListPolicies(ctx, &adminpb.ListPoliciesRequest{})
		cancel()
		if err != nil {
			logx.Errorf("refresh policies error: %v", err)
			continue
		}
		rules := make([]casbinauth.PolicyRule, 0, len(resp.Rules))
		for _, r := range resp.Rules {
			rules = append(rules, casbinauth.PolicyRule{Ptype: r.Ptype, V0: r.V0, V1: r.V1, V2: r.V2, V3: r.V3, V4: r.V4, V5: r.V5})
		}
		newEnforcer, err := casbinauth.NewEnforcer(rules)
		if err != nil {
			logx.Errorf("rebuild enforcer error: %v", err)
			continue
		}
		s.Enforcer = newEnforcer
		logx.Info("casbin policies refreshed")
	}
}
