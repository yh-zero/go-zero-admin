package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"go-zero-admin/pkg/middlecasbin"
)

type Config struct {
	rest.RestConf
	JwtAuth struct {
		AccessSecret string
		AccessExpire int
	}
	BizRedis redis.RedisConf
	Page     struct {
		PageNo   int64
		PageSize int64
	}
	AppletRPC  zrpc.RpcClientConf
	CasbinConf middlecasbin.CasbinConf
	DB         struct {
		DataSource   string
		MaxOpenConns int `json:",default=10"`
		MaxIdleConns int `json:",default=100"`
		MaxLifetime  int `json:",default=3600"`
	}
}
