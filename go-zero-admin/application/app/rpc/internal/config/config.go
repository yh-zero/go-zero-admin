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
		AccessSecret  string
		AccessExpire  int64
		RefreshExpire int64 `json:",default=2592000"`
	}
	FileRpc                   *zrpc.RpcClientConf `json:",optional"`
	EmailVerificationRequired bool                `json:",default=true"`
	SMTP                      struct {
		Host         string `json:",optional"`
		Port         int    `json:",default=587"`
		Username     string `json:",optional"`
		Password     string `json:",optional"`
		From         string `json:",optional"`
		FromName     string `json:",optional"`
		ValidSeconds int    `json:",default=600,range=[1:]"`
		ImplicitTLS  bool   `json:",default=false"`
	}
	DefaultUserPassword string `json:",optional"`
}
