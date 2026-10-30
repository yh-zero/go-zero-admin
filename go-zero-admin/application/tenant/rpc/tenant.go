package main

import (
	"flag"
	"fmt"

	"go-zero-admin/application/tenant/rpc/internal/config"
	tenantserviceServer "go-zero-admin/application/tenant/rpc/internal/server/tenantservice"
	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/tenant.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		tenantpb.RegisterTenantServiceServer(grpcServer, tenantserviceServer.NewTenantServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
