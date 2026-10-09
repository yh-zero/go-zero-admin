package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"go-zero-admin/pkg/middlecasbin"
	"go-zero-admin/pkg/rpcsecurity"
)

type Config struct {
	zrpc.RpcServerConf
	BizRedis      redis.RedisConf
	RPCAuth       rpcsecurity.Credential `json:",optional"`
	UserResources struct {
		// Must describe the AI service's actual database; unset disables AI lifecycle writes.
		AIDataSource string `json:",optional"`
	} `json:",optional"`
	DB struct {
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
