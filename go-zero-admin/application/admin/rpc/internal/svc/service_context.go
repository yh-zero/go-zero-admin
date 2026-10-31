package svc

import (
	"context"
	"crypto/tls"
	goredis "github.com/redis/go-redis/v9"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
	"go-zero-admin/application/admin/rpc/internal/config"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/file/rpc/client/fileservice"
	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/pkg/jwtx"
)

type ServiceContext struct {
	Config   config.Config
	DB       sqlx.SqlConn
	Redis    *redis.Redis
	Sessions *jwtx.SessionStore
	FileRpc  fileservice.FileService

	SysUserModel     model.SysUserModel
	SysDeptModel     model.SysDeptModel
	SysRoleModel     model.SysRoleModel
	SysMenuModel     model.SysMenuModel
	SysRoleMenuModel model.SysRoleMenuModel
	SysApiModel      model.SysApiModel
	CasbinRuleModel  model.CasbinRuleModel
	SysDictModel     model.SysDictModel
	SysAuditLogModel model.SysAuditLogModel
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
	return &ServiceContext{
		Config:           c,
		DB:               conn,
		Redis:            rdb,
		Sessions:         jwtx.NewSessionStore(sessionRedis, "admin"),
		FileRpc:          fileservice.NewFileService(zrpc.MustNewClient(c.FileRpc)),
		SysUserModel:     model.NewSysUserModel(conn, c.CacheRedis),
		SysDeptModel:     model.NewSysDeptModel(conn, c.CacheRedis),
		SysRoleModel:     model.NewSysRoleModel(conn, c.CacheRedis),
		SysMenuModel:     model.NewSysMenuModel(conn, c.CacheRedis),
		SysRoleMenuModel: model.NewSysRoleMenuModel(conn, c.CacheRedis),
		SysApiModel:      model.NewSysApiModel(conn, c.CacheRedis),
		CasbinRuleModel:  model.NewCasbinRuleModel(conn, c.CacheRedis),
		SysDictModel:     model.NewSysDictModel(conn, c.CacheRedis),
		SysAuditLogModel: model.NewSysAuditLogModel(conn, c.CacheRedis),
	}
}

// PublishPolicyChanged 策略变更后通知 API 进程刷新 enforcer（Casbin 热更新）
func (s *ServiceContext) PublishPolicyChanged(ctx context.Context) {
	notifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	_, _ = s.Redis.PublishCtx(notifyCtx, "casbin:policy:changed", time.Now().Unix())
}
