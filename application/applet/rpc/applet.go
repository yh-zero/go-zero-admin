package main

import (
	"flag"
	"fmt"

	"go-zero-admin/application/applet/rpc/internal/config"
	apiServer "go-zero-admin/application/applet/rpc/internal/server/api"
	authorityServer "go-zero-admin/application/applet/rpc/internal/server/authority"
	casbinServer "go-zero-admin/application/applet/rpc/internal/server/casbin"
	menuServer "go-zero-admin/application/applet/rpc/internal/server/menu"
	userServer "go-zero-admin/application/applet/rpc/internal/server/user"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/applet.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterUserServer(grpcServer, userServer.NewUserServer(ctx))
		pb.RegisterMenuServer(grpcServer, menuServer.NewMenuServer(ctx))
		pb.RegisterAuthorityServer(grpcServer, authorityServer.NewAuthorityServer(ctx))
		pb.RegisterAPIServer(grpcServer, apiServer.NewAPIServer(ctx))
		pb.RegisterCasbinServer(grpcServer, casbinServer.NewCasbinServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
