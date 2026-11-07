package main

import (
	"context"
	"flag"
	"fmt"
	"go-zero-admin/pkg/logsafe"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/config"
	aiserviceServer "go-zero-admin/application/ai/rpc/internal/server/aiservice"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/ai.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())
	if err := logsafe.Setup(c.Log, c.Name); err != nil {
		panic(err)
	}
	// 对话内容、工具参数和个人字段不进入默认 RPC/SQL 内容日志。
	sqlx.DisableLog()
	services := aipb.File_application_ai_rpc_desc_ai_proto.Services()
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			method := "/" + string(service.FullName()) + "/" + string(service.Methods().Get(j).Name())
			c.Middlewares.StatConf.IgnoreContentMethods = append(c.Middlewares.StatConf.IgnoreContentMethods, method)
			zrpc.DontLogClientContentForMethod(method)
		}
	}
	ctx := svc.NewServiceContext(c)
	stopKbWorkers := ctx.StartKbIngestWorkers(context.Background())
	defer stopKbWorkers()

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		aipb.RegisterAiServiceServer(grpcServer, aiserviceServer.NewAiServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
