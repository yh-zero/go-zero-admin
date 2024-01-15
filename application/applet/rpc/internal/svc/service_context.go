package svc

import (
	"go-zero-admin/application/applet/rpc/internal/config"
	"go-zero-admin/pkg/orm"

	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config   config.Config
	DB       *orm.DB
	BizRedis *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	db := orm.MustNewMysql(&orm.Config{
		DSN:          c.DB.DataSource,
		MaxOpenConns: c.DB.MaxOpenConns,
		MaxIdleCnns:  c.DB.MaxIdleConns,
		MaxLifetime:  c.DB.MaxLifetime,
	})
	rds := redis.MustNewRedis(redis.RedisConf{
		Host: c.BizRedis.Host,
		Type: c.BizRedis.Type,
		Pass: c.BizRedis.Pass,
	})

	return &ServiceContext{
		Config:   c,
		DB:       db,
		BizRedis: rds,
	}
}
