package rpcprivacy

import (
	"slices"
	"testing"

	"github.com/zeromicro/go-zero/zrpc"
	_ "go-zero-admin/application/applet/rpc/pb"
)

func TestEveryApplicationRPCOmitsBodiesAndPreservesConfig(t *testing.T) {
	config := zrpc.RpcServerConf{}
	config.Middlewares.StatConf.IgnoreContentMethods = []string{"/existing/Method"}
	ConfigureServer(&config)
	count := len(config.Middlewares.StatConf.IgnoreContentMethods)
	ConfigureServer(&config)
	if len(config.Middlewares.StatConf.IgnoreContentMethods) != count {
		t.Fatal("configuration is not idempotent")
	}
	for _, method := range []string{"/existing/Method", "/pb.User/GetUserInfo", "/pb.User/GetUserList", "/pb.Casbin/Enforce", "/pb.Menu/GetMenuTree"} {
		if !slices.Contains(config.Middlewares.StatConf.IgnoreContentMethods, method) {
			t.Fatalf("RPC body logging still enabled for %s", method)
		}
	}
}
