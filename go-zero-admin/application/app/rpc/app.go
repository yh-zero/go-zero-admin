package main

import (
	"flag"
	"fmt"

	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/config"
	appadminserviceServer "go-zero-admin/application/app/rpc/internal/server/appadminservice"
	appauthserviceServer "go-zero-admin/application/app/rpc/internal/server/appauthservice"
	"go-zero-admin/application/app/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/app.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		apppb.RegisterAppAuthServiceServer(grpcServer, appauthserviceServer.NewAppAuthServiceServer(ctx))
		apppb.RegisterAppAdminServiceServer(grpcServer, appadminserviceServer.NewAppAdminServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
