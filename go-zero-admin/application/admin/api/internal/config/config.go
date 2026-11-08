package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	AdminRpc       zrpc.RpcClientConf
	FileRpc        zrpc.RpcClientConf
	TenantRpc      *zrpc.RpcClientConf `json:",optional"`
	AppRpc         *zrpc.RpcClientConf `json:",optional"`
	AiRpc          *zrpc.RpcClientConf `json:",optional"`
	Redis          redis.RedisConf
	TrustedProxies []string `json:",optional"`
	Auth           struct {
		AccessSecret string
		AccessExpire int64
	}
}
