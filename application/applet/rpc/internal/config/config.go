package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"go-zero-admin/pkg/appauth"
	"go-zero-admin/pkg/middlecasbin"
	"go-zero-admin/pkg/objectstorage"
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
	AppJwtAuth appauth.Config `json:",optional"`
	IPLocation struct {
		IPv4Path string `json:",default=data/ipregion/ip2region_v4.xdb"`
		IPv6Path string `json:",default=data/ipregion/ip2region_v6.xdb"`
	} `json:",optional"`
	Default struct {
		UserPassword string
	}
	// Storage credentials belong only to this RPC process.
	Storage objectstorage.Config `json:",optional"`
	// Mail owns the SMTP credentials and rate-limit settings for outbound
	// verification codes. The API tier talks to it over gRPC.
	Mail MailConfig `json:",optional"`
}

// MailConfig uses implicit TLS (typically port 465) and verifies the SMTP cert.
type MailConfig struct {
	Host     string `json:",optional"`
	Port     int    `json:",optional"`
	From     string `json:",optional"`
	Username string `json:",optional"`
	Password string `json:",optional"`
}
