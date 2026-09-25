package svc

import (
	"go-zero-admin/application/applet/api/internal/config"
	"go-zero-admin/application/applet/api/internal/middleware"
	"go-zero-admin/application/applet/rpc/client/api"
	"go-zero-admin/application/applet/rpc/client/authority"
	casbinRPC "go-zero-admin/application/applet/rpc/client/casbin"
	dictionaryRPC "go-zero-admin/application/applet/rpc/client/dictionary"
	"go-zero-admin/application/applet/rpc/client/menu"
	"go-zero-admin/application/applet/rpc/client/user"
	"go-zero-admin/pkg/rpcprivacy"
	"net/http"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

const (
	defaultOssConnectTimeout   = 1
	defaultOssReadWriteTimeout = 3
)

type ServiceContext struct {
	Session             rest.Middleware
	Config              config.Config
	Authority           rest.Middleware
	BizRedis            *redis.Redis
	AppletUserRPC       user.User
	AppletMenuRPC       menu.Menu
	AppletAuthorityRPC  authority.Authority
	AppletAPIRPC        api.Api
	AppletCasbinRPC     casbinRPC.Casbin
	AppletDictionaryRPC dictionaryRPC.Dictionary
	OssClient           *oss.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.Oss.ConnectTimeout == 0 {
		c.Oss.ConnectTimeout = defaultOssConnectTimeout
	}
	if c.Oss.ReadWriteTimeout == 0 {
		c.Oss.ReadWriteTimeout = defaultOssReadWriteTimeout
	}
	var oc *oss.Client
	if c.Oss.Endpoint != "" && c.Oss.AccessKeyId != "" && c.Oss.AccessKeySecret != "" && c.Oss.BucketName != "" {
		var err error
		oc, err = oss.New(c.Oss.Endpoint, c.Oss.AccessKeyId, c.Oss.AccessKeySecret, oss.Timeout(c.Oss.ConnectTimeout, c.Oss.ReadWriteTimeout))
		if err != nil {
			panic(err)
		}
	}

	rds := redis.MustNewRedis(c.BizRedis, redis.WithPass(c.BizRedis.Pass)) // jsonMark:骑着毛驴背单词

	rpcprivacy.ConfigureClient()
	appletRPC := zrpc.MustNewClient(c.AppletRPC)
	casbinCli := casbinRPC.NewCasbin(appletRPC)

	svc := &ServiceContext{
		Config:              c,
		OssClient:           oc,
		BizRedis:            rds,
		AppletUserRPC:       user.NewUser(appletRPC),
		AppletMenuRPC:       menu.NewMenu(appletRPC),
		AppletAuthorityRPC:  authority.NewAuthority(appletRPC),
		AppletAPIRPC:        api.NewApi(appletRPC),
		AppletCasbinRPC:     casbinCli,
		AppletDictionaryRPC: dictionaryRPC.NewDictionary(appletRPC),
	}

	// 权限鉴权走RPC casbin api层不直连数据库
	svc.Session = middleware.NewSessionMiddleware(svc.AppletUserRPC).Handle
	authority := middleware.NewAuthorityMiddleware(casbinCli).Handle
	svc.Authority = func(next http.HandlerFunc) http.HandlerFunc { return svc.Session(authority(next)) }

	return svc
}
