package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	Mysql      struct{ DataSource string }
	CacheRedis cache.CacheConf
	JwtAuth    struct {
		AccessSecret string
		AccessExpire int64
	}
	DefaultUserPassword string `json:",optional"`
}
