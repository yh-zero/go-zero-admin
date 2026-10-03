package main

import (
	"flag"
	"fmt"

	"go-zero-admin/application/ai/rpc/internal/config"
	agentlogic "go-zero-admin/application/ai/rpc/internal/logic/agent"
	agentserver "go-zero-admin/application/ai/rpc/internal/server/agent"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/pkg/rpcprivacy"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/ai.yaml", "the config file")

func main() {
	flag.Parse()
	var c config.Config
	conf.MustLoad(*configFile, &c)
	rpcprivacy.ConfigureServer(&c.RpcServerConf)
	ctx := svc.NewServiceContext(c)
	defer ctx.Close()
	if err := ctx.StartAgentRuntime(agentlogic.NewAgentExecutor(ctx)); err != nil {
		panic(err)
	}
	s := zrpc.MustNewServer(c.RpcServerConf, func(server *grpc.Server) {
		pb.RegisterAgentServer(server, agentserver.NewAgentServer(ctx))
		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(server)
		}
	})
	defer s.Stop()
	fmt.Printf("Starting AI rpc server at %s...\n", c.ListenOn)
	s.Start()
}
