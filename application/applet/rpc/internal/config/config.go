package config

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"go-zero-admin/pkg/appauth"
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
	AppJwtAuth appauth.Config `json:",optional"`
	IPLocation struct {
		IPv4Path string `json:",default=data/ipregion/ip2region_v4.xdb"`
		IPv6Path string `json:",default=data/ipregion/ip2region_v6.xdb"`
	} `json:",optional"`
	Default struct {
		UserPassword string
	}
	// Oss owns the Aliyun OSS credentials. Only this RPC process should ever see
	// AccessKeySecret; the API tier must not be configured with these values.
	Oss OssConfig `json:",optional"`
	// Mail owns the SMTP credentials and rate-limit settings for outbound
	// verification codes. The API tier talks to it over gRPC.
	Mail MailConfig `json:",optional"`
}

// OssConfig uses implicit TLS endpoints (or unencrypted if Insecure=true for
// local dev). AccessKeySecret must be supplied via the YAML/secret manager, not
// through the API tier.
type OssConfig struct {
	Endpoint         string `json:",optional"`
	AccessKeyId      string `json:",optional"`
	AccessKeySecret  string `json:",optional"`
	BucketName       string `json:",optional"`
	ConnectTimeout   int64  `json:",default=1"`
	ReadWriteTimeout int64  `json:",default=3"`
}

// MailConfig uses implicit TLS (typically port 465) and verifies the SMTP cert.
type MailConfig struct {
	Host     string `json:",optional"`
	Port     int    `json:",optional"`
	From     string `json:",optional"`
	Username string `json:",optional"`
	Password string `json:",optional"`
}
