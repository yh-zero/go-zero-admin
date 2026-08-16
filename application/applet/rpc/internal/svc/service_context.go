package svc

import (
	"go-zero-admin/application/applet/rpc/internal/config"
	"go-zero-admin/pkg/orm"

	"github.com/casbin/casbin/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config   config.Config
	DB       *orm.DB
	BizRedis *redis.Redis
	Casbin   *casbin.SyncedCachedEnforcer // 常驻enforcer 供casbin鉴权/策略管理
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

	// 只创建一次 供所有logic共用
	casb := c.CasbinConf.MustNewCasbinWithRedisWatcher(c.DB.DataSource, c.BizRedis)

	return &ServiceContext{
		Config:   c,
		DB:       db,
		BizRedis: rds,
		Casbin:   casb,
	}
}
