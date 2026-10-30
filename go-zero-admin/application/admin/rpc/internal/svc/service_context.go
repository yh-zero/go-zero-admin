package svc

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/admin/rpc/internal/config"
	"go-zero-admin/application/admin/rpc/internal/model"
)

type ServiceContext struct {
	Config config.Config
	DB     sqlx.SqlConn
	Redis  *redis.Redis

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
	return &ServiceContext{
		Config:           c,
		DB:               conn,
		Redis:            rdb,
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
	_, _ = s.Redis.PublishCtx(ctx, "casbin:policy:changed", time.Now().Unix())
}
