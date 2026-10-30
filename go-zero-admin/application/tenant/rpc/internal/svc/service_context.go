package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"go-zero-admin/application/tenant/rpc/internal/config"
	"go-zero-admin/application/tenant/rpc/internal/model"
)

type ServiceContext struct {
	Config config.Config
	DB     sqlx.SqlConn
	Redis  *redis.Redis

	TenantRegistryModel model.TenantRegistryModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.Mysql.DataSource)
	rdb := redis.MustNewRedis(c.CacheRedis[0].RedisConf)
	return &ServiceContext{
		Config:              c,
		DB:                  conn,
		Redis:               rdb,
		TenantRegistryModel: model.NewTenantRegistryModel(conn, c.CacheRedis),
	}
}
