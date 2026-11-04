package main

import (
	"flag"
	"fmt"
	"go-zero-admin/pkg/logsafe"

	"go-zero-admin/application/tenant/rpc/internal/config"
	tenantserviceServer "go-zero-admin/application/tenant/rpc/internal/server/tenantservice"
	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/tenant.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	if err := logsafe.Setup(c.Log, c.Name); err != nil {
		panic(err)
	}
	sqlx.DisableLog()
	serviceDesc := tenantpb.File_application_tenant_rpc_desc_tenant_proto.Services().Get(0)
	for i := 0; i < serviceDesc.Methods().Len(); i++ {
		c.Middlewares.StatConf.IgnoreContentMethods = append(c.Middlewares.StatConf.IgnoreContentMethods, "/"+string(serviceDesc.FullName())+"/"+string(serviceDesc.Methods().Get(i).Name()))
	}
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
