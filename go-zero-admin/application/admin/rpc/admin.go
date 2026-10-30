package main

import (
	"flag"
	"fmt"

	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/config"
	apiserviceServer "go-zero-admin/application/admin/rpc/internal/server/apiservice"
	auditserviceServer "go-zero-admin/application/admin/rpc/internal/server/auditservice"
	authserviceServer "go-zero-admin/application/admin/rpc/internal/server/authservice"
	casbinserviceServer "go-zero-admin/application/admin/rpc/internal/server/casbinservice"
	deptserviceServer "go-zero-admin/application/admin/rpc/internal/server/deptservice"
	dictserviceServer "go-zero-admin/application/admin/rpc/internal/server/dictservice"
	menuserviceServer "go-zero-admin/application/admin/rpc/internal/server/menuservice"
	profileserviceServer "go-zero-admin/application/admin/rpc/internal/server/profileservice"
	roleserviceServer "go-zero-admin/application/admin/rpc/internal/server/roleservice"
	userserviceServer "go-zero-admin/application/admin/rpc/internal/server/userservice"
	"go-zero-admin/application/admin/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/admin.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		adminpb.RegisterAuthServiceServer(grpcServer, authserviceServer.NewAuthServiceServer(ctx))
		adminpb.RegisterUserServiceServer(grpcServer, userserviceServer.NewUserServiceServer(ctx))
		adminpb.RegisterProfileServiceServer(grpcServer, profileserviceServer.NewProfileServiceServer(ctx))
		adminpb.RegisterRoleServiceServer(grpcServer, roleserviceServer.NewRoleServiceServer(ctx))
		adminpb.RegisterMenuServiceServer(grpcServer, menuserviceServer.NewMenuServiceServer(ctx))
		adminpb.RegisterDeptServiceServer(grpcServer, deptserviceServer.NewDeptServiceServer(ctx))
		adminpb.RegisterDictServiceServer(grpcServer, dictserviceServer.NewDictServiceServer(ctx))
		adminpb.RegisterApiServiceServer(grpcServer, apiserviceServer.NewApiServiceServer(ctx))
		adminpb.RegisterAuditServiceServer(grpcServer, auditserviceServer.NewAuditServiceServer(ctx))
		adminpb.RegisterCasbinServiceServer(grpcServer, casbinserviceServer.NewCasbinServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
