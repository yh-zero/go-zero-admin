package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"go-zero-admin/pkg/middlecasbin"
)

type Config struct {
	zrpc.RpcServerConf
	BizRedis redis.RedisConf
	DB       struct {
		DataSource   string
		MaxOpenConns int `json:",default=10"`
		MaxIdleConns int `json:",default=100"`
		MaxLifetime  int `json:",default=3600"`
	}
	JwtAuth struct {
		AccessSecret string
		AccessExpire int64
	}
	CasbinConf middlecasbin.CasbinConf
	Default    struct {
		UserPassword string
	}
}
