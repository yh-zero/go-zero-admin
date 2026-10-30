package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	AdminRpc zrpc.RpcClientConf
	FileRpc  zrpc.RpcClientConf
	Redis    redis.RedisConf
	Auth     struct {
		AccessSecret string
		AccessExpire int64
	}
}
