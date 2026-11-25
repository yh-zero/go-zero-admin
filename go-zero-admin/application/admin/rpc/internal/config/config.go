package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	Mysql struct {
		DataSource string
	}

	CacheRedis cache.CacheConf
	FileRpc    zrpc.RpcClientConf

	JwtAuth struct {
		AccessSecret  string
		AccessExpire  int64
		RefreshExpire int64 `json:",default=2592000"`
	}

	Captcha struct {
		Enabled       bool `json:",default=true"`
		ExpireSeconds int
	}

	DefaultUserPassword string `json:",optional"`

	// Audit 审计保留策略；RetentionDays=0 默认关闭自动删除。
	Audit struct {
		RetentionDays          int `json:",default=0"`
		CleanupIntervalSeconds int `json:",default=3600"`
		CleanupBatchSize       int `json:",default=1000"`
	}
}
