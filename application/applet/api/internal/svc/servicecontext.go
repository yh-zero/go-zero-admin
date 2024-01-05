package svc

import (
	"github.com/casbin/casbin/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"go-zero-admin/application/applet/api/internal/config"
	"go-zero-admin/application/applet/api/internal/middleware"
	"go-zero-admin/application/applet/rpc/client/api"
	"go-zero-admin/application/applet/rpc/client/authority"
	casbinRPC "go-zero-admin/application/applet/rpc/client/casbin"
	"go-zero-admin/application/applet/rpc/client/menu"
	"go-zero-admin/application/applet/rpc/client/user"
)

type ServiceContext struct {
	Config             config.Config
	Authority          rest.Middleware
	BizRedis           *redis.Redis
	AppletUserRPC      user.User
	AppletMenuRPC      menu.Menu
	AppletAuthorityRPC authority.Authority
	AppletAPIRPC       api.API
	AppletCasbinRPC    casbinRPC.Casbin
	Casbin             *casbin.SyncedCachedEnforcer
	BanRoleData        map[string]bool // ban role means the role status is not normal
}

func NewServiceContext(c config.Config) *ServiceContext {
	rds := redis.New(c.BizRedis.Host, redis.WithPass(c.BizRedis.Pass))

	casB := c.CasbinConf.MustNewCasbinWithRedisWatcher(c.DB.DataSource, c.BizRedis)

	svc := &ServiceContext{
		Config:             c,
		BizRedis:           redis.New(c.BizRedis.Host, redis.WithPass(c.BizRedis.Pass)),
		AppletUserRPC:      user.NewUser(zrpc.MustNewClient(c.AppletRPC)),
		AppletMenuRPC:      menu.NewMenu(zrpc.MustNewClient(c.AppletRPC)),
		AppletAuthorityRPC: authority.NewAuthority(zrpc.MustNewClient(c.AppletRPC)),
		AppletAPIRPC:       api.NewAPI(zrpc.MustNewClient(c.AppletRPC)),
		AppletCasbinRPC:    casbinRPC.NewCasbin(zrpc.MustNewClient(c.AppletRPC)),
		Casbin:             casB,
	}

	svc.Authority = middleware.NewAuthorityMiddleware(casB, rds, svc.BanRoleData).Handle

	return svc
}
