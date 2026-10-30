package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"go-zero-admin/application/app/rpc/internal/config"
	"go-zero-admin/application/app/rpc/internal/model"
)

type ServiceContext struct {
	Config config.Config
	DB     sqlx.SqlConn
	Redis  *redis.Redis

	AppUserModel            model.AppUserModel
	AppMembershipGrantModel model.AppMembershipGrantModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.Mysql.DataSource)
	rdb := redis.MustNewRedis(c.CacheRedis[0].RedisConf)
	return &ServiceContext{
		Config:                  c,
		DB:                      conn,
		Redis:                   rdb,
		AppUserModel:            model.NewAppUserModel(conn, c.CacheRedis),
		AppMembershipGrantModel: model.NewAppMembershipGrantModel(conn, c.CacheRedis),
	}
}
