package main

import (
	"flag"
	"fmt"

	"go-zero-admin/application/applet/rpc/internal/config"
	agentToolsServer "go-zero-admin/application/applet/rpc/internal/server/agenttools"
	apiServer "go-zero-admin/application/applet/rpc/internal/server/api"
	auditServer "go-zero-admin/application/applet/rpc/internal/server/audit"
	authorityServer "go-zero-admin/application/applet/rpc/internal/server/authority"
	casbinServer "go-zero-admin/application/applet/rpc/internal/server/casbin"
	dictionaryServer "go-zero-admin/application/applet/rpc/internal/server/dictionary"
	fileServer "go-zero-admin/application/applet/rpc/internal/server/fileresourceservice"
	menuServer "go-zero-admin/application/applet/rpc/internal/server/menu"
	organizationServer "go-zero-admin/application/applet/rpc/internal/server/organization"
	permissionServer "go-zero-admin/application/applet/rpc/internal/server/permission"
	sessionServer "go-zero-admin/application/applet/rpc/internal/server/sessionmanage"
	userServer "go-zero-admin/application/applet/rpc/internal/server/user"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/rpcprivacy"

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
	rpcprivacy.ConfigureServer(&c.RpcServerConf)
	ctx := svc.NewServiceContext(c)
	defer ctx.Close()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		pb.RegisterAgentToolsServer(grpcServer, agentToolsServer.NewAgentToolsServer(ctx))
		pb.RegisterUserServer(grpcServer, userServer.NewUserServer(ctx))
		pb.RegisterPermissionServer(grpcServer, permissionServer.NewPermissionServer(ctx))
		pb.RegisterMenuServer(grpcServer, menuServer.NewMenuServer(ctx))
		pb.RegisterAuthorityServer(grpcServer, authorityServer.NewAuthorityServer(ctx))
		pb.RegisterApiServer(grpcServer, apiServer.NewApiServer(ctx))
		pb.RegisterCasbinServer(grpcServer, casbinServer.NewCasbinServer(ctx))
		pb.RegisterDictionaryServer(grpcServer, dictionaryServer.NewDictionaryServer(ctx))
		pb.RegisterAuditServer(grpcServer, auditServer.NewAuditServer(ctx))
		pb.RegisterOrganizationServer(grpcServer, organizationServer.NewOrganizationServer(ctx))
		pb.RegisterFileResourceServiceServer(grpcServer, fileServer.NewFileResourceServiceServer(ctx))
		pb.RegisterSessionManageServer(grpcServer, sessionServer.NewSessionManageServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
