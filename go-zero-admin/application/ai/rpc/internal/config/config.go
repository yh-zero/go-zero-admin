package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"go-zero-admin/pkg/aiagent"
	"go-zero-admin/pkg/rpcsecurity"
)

type Config struct {
	zrpc.RpcServerConf
	RPCAuth   rpcsecurity.Credential `json:",optional"`
	AppletRPC zrpc.RpcClientConf
	BizRedis  redis.RedisConf  `json:",optional"`
	AI        aiagent.Settings `json:",optional"`
	DB        struct {
		DataSource   string
		MaxOpenConns int `json:",default=100"`
		MaxIdleConns int `json:",default=10"`
		MaxLifetime  int `json:",default=3600"`
	}
}
