package main

import (
	"flag"
	"fmt"

	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/config"
	fileserviceServer "go-zero-admin/application/file/rpc/internal/server/fileservice"
	"go-zero-admin/application/file/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var configFile = flag.String("f", "etc/file.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	// 文件字节、签名链接及数据库个人字段不进入默认内容日志。
	sqlx.DisableLog()
	services := filepb.File_application_file_rpc_desc_file_proto.Services()
	for i := 0; i < services.Len(); i++ {
		service := services.Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			method := "/" + string(service.FullName()) + "/" + string(service.Methods().Get(j).Name())
			c.Middlewares.StatConf.IgnoreContentMethods = append(c.Middlewares.StatConf.IgnoreContentMethods, method)
			zrpc.DontLogClientContentForMethod(method)
		}
	}
	ctx := svc.NewServiceContext(c)

	s := zrpc.MustNewServer(c.RpcServerConf, func(grpcServer *grpc.Server) {
		filepb.RegisterFileServiceServer(grpcServer, fileserviceServer.NewFileServiceServer(ctx))

		if c.Mode == service.DevMode || c.Mode == service.TestMode {
			reflection.Register(grpcServer)
		}
	})
	// protobuf 字段开销留有余量；实际文件大小仍由业务校验为 10 MiB。
	s.AddOptions(grpc.MaxRecvMsgSize(12<<20), grpc.MaxSendMsgSize(12<<20))
	defer s.Stop()

	fmt.Printf("Starting rpc server at %s...\n", c.ListenOn)
	s.Start()
}
