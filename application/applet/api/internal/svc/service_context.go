package svc

import (
	"go-zero-admin/application/applet/api/internal/config"
	"go-zero-admin/application/applet/api/internal/middleware"
	"go-zero-admin/application/applet/rpc/client/api"
	"go-zero-admin/application/applet/rpc/client/authority"
	casbinRPC "go-zero-admin/application/applet/rpc/client/casbin"
	"go-zero-admin/application/applet/rpc/client/menu"
	"go-zero-admin/application/applet/rpc/client/user"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/casbin/casbin/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

const (
	defaultOssConnectTimeout   = 1
	defaultOssReadWriteTimeout = 3
)

type ServiceContext struct {
	Config             config.Config
	Authority          rest.Middleware
	BizRedis           *redis.Redis
	AppletUserRPC      user.User
	AppletMenuRPC      menu.Menu
	AppletAuthorityRPC authority.Authority
	AppletAPIRPC       api.Api
	AppletCasbinRPC    casbinRPC.Casbin
	Casbin             *casbin.SyncedCachedEnforcer
	BanRoleData        map[string]bool // ban role means the role status is not normal
	OssClient          *oss.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.Oss.ConnectTimeout == 0 {
		c.Oss.ConnectTimeout = defaultOssConnectTimeout
	}
	if c.Oss.ReadWriteTimeout == 0 {
		c.Oss.ReadWriteTimeout = defaultOssReadWriteTimeout
	}
	oc, err := oss.New(c.Oss.Endpoint, c.Oss.AccessKeyId, c.Oss.AccessKeySecret, oss.Timeout(c.Oss.ConnectTimeout, c.Oss.ReadWriteTimeout))
	if err != nil {
		panic(err)
	}

	//rds := redis.New(c.BizRedis.Host, redis.WithPass(c.BizRedis.Pass))
	rds := redis.MustNewRedis(c.BizRedis, redis.WithPass(c.BizRedis.Pass)) // jsonMark:骑着毛驴背单词

	casB := c.CasbinConf.MustNewCasbinWithRedisWatcher(c.DB.DataSource, c.BizRedis)

	svc := &ServiceContext{
		Config:    c,
		OssClient: oc,
		//BizRedis:           redis.New(c.BizRedis.Host, redis.WithPass(c.BizRedis.Pass)),
		BizRedis:           rds,
		AppletUserRPC:      user.NewUser(zrpc.MustNewClient(c.AppletRPC)),
		AppletMenuRPC:      menu.NewMenu(zrpc.MustNewClient(c.AppletRPC)),
		AppletAuthorityRPC: authority.NewAuthority(zrpc.MustNewClient(c.AppletRPC)),
		AppletAPIRPC:       api.NewApi(zrpc.MustNewClient(c.AppletRPC)),
		AppletCasbinRPC:    casbinRPC.NewCasbin(zrpc.MustNewClient(c.AppletRPC)),
		Casbin:             casB,
	}

	svc.Authority = middleware.NewAuthorityMiddleware(casB, rds, svc.BanRoleData).Handle

	return svc
}
