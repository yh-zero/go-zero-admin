package svc

import (
	"go-zero-admin/application/ai/rpc/internal/config"
	"go-zero-admin/application/applet/rpc/client/agenttools"
	"go-zero-admin/application/applet/rpc/client/casbin"
	"go-zero-admin/application/applet/rpc/client/user"
	"go-zero-admin/pkg/agentjobs"
	"go-zero-admin/pkg/aiagent"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/orm"
	"go-zero-admin/pkg/rpcprivacy"
	"go-zero-admin/pkg/rpcsecurity"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
)

type ServiceContext struct {
	Config              config.Config
	DB                  *orm.DB
	BizRedis            *redis.Redis
	AgentConfig         aiagent.Config
	AgentRunner         aiagent.Runner
	AgentJobs           *agentjobs.Manager
	AppletUserRPC       user.User
	AppletCasbinRPC     casbin.Casbin
	AppletAgentToolsRPC agenttools.AgentTools
	appletConn          *grpc.ClientConn
}

func NewServiceContext(c config.Config) *ServiceContext {
	if err := rpcsecurity.ValidateClient(c.AppletRPC, c.Mode); err != nil {
		panic(err)
	}
	if err := rpcsecurity.RegisterCredential(c.RpcServerConf, c.RPCAuth); err != nil {
		panic(err)
	}
	rpcprivacy.ConfigureClient()
	client, err := zrpc.NewClient(c.AppletRPC, zrpc.WithUnaryClientInterceptor(audit.ClientInterceptor()))
	if err != nil {
		panic("unable to initialize business RPC client")
	}
	db, err := orm.NewMysql(&orm.Config{DSN: c.DB.DataSource, MaxOpenConns: c.DB.MaxOpenConns, MaxIdleCnns: c.DB.MaxIdleConns, MaxLifetime: c.DB.MaxLifetime})
	if err != nil {
		_ = client.Conn().Close()
		panic("unable to initialize AI task database")
	}
	s := &ServiceContext{
		Config: c, DB: db, appletConn: client.Conn(),
		AppletUserRPC: user.NewUser(client), AppletCasbinRPC: casbin.NewCasbin(client),
		AppletAgentToolsRPC: agenttools.NewAgentTools(client),
	}
	if c.BizRedis.Host != "" {
		s.BizRedis = redis.MustNewRedis(c.BizRedis, redis.WithPass(c.BizRedis.Pass))
	}
	return s
}

func (s *ServiceContext) Close() {
	if s.AgentJobs != nil {
		s.AgentJobs.Close()
	}
	if s.appletConn != nil {
		_ = s.appletConn.Close()
	}
	if s.DB != nil {
		if sqlDB, err := s.DB.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}
