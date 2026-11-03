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
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/app.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	sqlx.DisableLog()
	services := apppb.File_application_app_rpc_desc_app_proto.Services()
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			c.Middlewares.StatConf.IgnoreContentMethods = append(c.Middlewares.StatConf.IgnoreContentMethods, "/"+string(service.FullName())+"/"+string(service.Methods().Get(j).Name()))
		}
	}
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
