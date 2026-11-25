package svc

import (
	"context"
	"crypto/tls"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
	"go-zero-admin/application/admin/rpc/internal/config"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/file/rpc/client/fileservice"
	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/pkg/casbinauth"
	"go-zero-admin/pkg/jwtx"
)

type ServiceContext struct {
	Config   config.Config
	DB       sqlx.SqlConn
	Redis    *redis.Redis
	Sessions *jwtx.SessionStore
	FileRpc  fileservice.FileService

	SysUserModel        model.SysUserModel
	SysDeptModel        model.SysDeptModel
	SysRoleModel        model.SysRoleModel
	SysMenuModel        model.SysMenuModel
	SysRoleMenuModel    model.SysRoleMenuModel
	SysApiModel         model.SysApiModel
	SysPermissionModel  model.SysPermissionModel
	CasbinRuleModel     model.CasbinRuleModel
	SysDictModel        model.SysDictModel
	SysAuditLogModel    model.SysAuditLogModel
	SysPolicyStateModel model.SysPolicyStateModel
	Enforcer            *casbinauth.Holder
	PolicyRevision      int64
	PolicyGeneratedAt   int64
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.Mysql.DataSource)
	rdb := redis.MustNewRedis(c.CacheRedis[0].RedisConf)
	redisConfig := c.CacheRedis[0].RedisConf
	options := &goredis.Options{Addr: redisConfig.Host, Username: redisConfig.User, Password: redisConfig.Pass}
	if redisConfig.Tls {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	sessionRedis := goredis.NewClient(options)
	fileService := filepb.File_application_file_rpc_desc_file_proto.Services().Get(0)
	for i := 0; i < fileService.Methods().Len(); i++ {
		zrpc.DontLogClientContentForMethod("/" + string(fileService.FullName()) + "/" + string(fileService.Methods().Get(i).Name()))
	}
	ctx := &ServiceContext{
		Config:              c,
		DB:                  conn,
		Redis:               rdb,
		Sessions:            jwtx.NewSessionStore(sessionRedis, "admin"),
		FileRpc:             fileservice.NewFileService(zrpc.MustNewClient(c.FileRpc)),
		SysUserModel:        model.NewSysUserModel(conn, c.CacheRedis),
		SysDeptModel:        model.NewSysDeptModel(conn, c.CacheRedis),
		SysRoleModel:        model.NewSysRoleModel(conn, c.CacheRedis),
		SysMenuModel:        model.NewSysMenuModel(conn, c.CacheRedis),
		SysRoleMenuModel:    model.NewSysRoleMenuModel(conn, c.CacheRedis),
		SysApiModel:         model.NewSysApiModel(conn, c.CacheRedis),
		SysPermissionModel:  model.NewSysPermissionModel(conn, c.CacheRedis),
		CasbinRuleModel:     model.NewCasbinRuleModel(conn, c.CacheRedis),
		SysDictModel:        model.NewSysDictModel(conn, c.CacheRedis),
		SysAuditLogModel:    model.NewSysAuditLogModel(conn, c.CacheRedis),
		SysPolicyStateModel: model.NewSysPolicyStateModel(conn, c.CacheRedis),
	}
	if err := ctx.loadPolicies(); err != nil {
		panic("load casbin policies failed: " + err.Error())
	}
	return ctx
}

func (s *ServiceContext) loadPolicies() error {
	rows, err := s.CasbinRuleModel.ListActivePolicies(context.Background())
	if err != nil {
		return err
	}
	rules := make([]casbinauth.PolicyRule, 0, len(rows))
	for _, r := range rows {
		rules = append(rules, casbinauth.PolicyRule{Ptype: r.Ptype, V0: r.V0, V1: r.V1, V2: r.V2, V3: r.V3, V4: r.V4, V5: r.V5})
	}
	enforcer, err := casbinauth.NewHolder(rules)
	if err != nil {
		return err
	}
	revision, err := s.SysPolicyStateModel.CurrentRevision(context.Background())
	if err != nil {
		return err
	}
	s.Enforcer = enforcer
	s.PolicyRevision = revision
	s.PolicyGeneratedAt = time.Now().Unix()
	return nil
}

func (s *ServiceContext) PolicySnapshot() (revision, generatedAt int64) {
	return s.PolicyRevision, s.PolicyGeneratedAt
}

// PublishPolicyChanged 先递增版本并重建快照，再通知 API 进程刷新 enforcer。
func (s *ServiceContext) PublishPolicyChanged(ctx context.Context) {
	if _, err := s.SysPolicyStateModel.BumpRevision(ctx); err != nil {
		logx.WithContext(ctx).Errorf("bump policy revision failed: %v", err)
		return
	}
	if err := s.loadPolicies(); err != nil {
		logx.WithContext(ctx).Errorf("reload casbin policies failed: %v", err)
		return
	}
	notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	_, _ = s.Redis.PublishCtx(notifyCtx, "casbin:policy:changed", s.PolicyRevision)
}
