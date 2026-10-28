package rpcsecurity

import (
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/zrpc"
)

func TestServerRequiresAuthenticationOutsideLoopback(t *testing.T) {
	for _, address := range []string{"127.0.0.1:6001", "[::1]:6001"} {
		config := zrpc.RpcServerConf{ListenOn: address}
		config.Mode = service.DevMode
		if err := ValidateServer(config, Credential{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, address := range []string{"0.0.0.0:6001", "192.168.0.10:6001", ":6001"} {
		if err := ValidateServer(zrpc.RpcServerConf{ListenOn: address}, Credential{}); err == nil {
			t.Fatalf("unprotected listener accepted: %s", address)
		}
	}
	productionLocal := zrpc.RpcServerConf{ListenOn: "127.0.0.1:6001"}
	productionLocal.Mode = service.ProMode
	if err := ValidateServer(productionLocal, Credential{}); err == nil {
		t.Fatal("production loopback listener omitted authentication")
	}
	server := zrpc.RpcServerConf{ListenOn: "0.0.0.0:6001", Auth: true, StrictControl: true, Redis: redis.RedisKeyConf{RedisConf: redis.RedisConf{Host: "127.0.0.1:6379", Type: "node"}, Key: "tokens"}}
	credential := Credential{App: "applet-api", Token: strings.Repeat("x", 32)}
	if err := ValidateServer(server, credential); err != nil {
		t.Fatal(err)
	}
	server.StrictControl = false
	if err := ValidateServer(server, credential); err == nil {
		t.Fatal("fail-open authentication accepted")
	}
}

func TestCredentialsAndProductionClients(t *testing.T) {
	for _, credential := range []Credential{{}, {App: "app", Token: "short"}, {App: "app", Token: "replace-with-a-random-secret-at-least-32-characters"}, {App: "bad app", Token: strings.Repeat("x", 32)}} {
		if err := ValidateCredential(credential); err == nil {
			t.Fatalf("invalid credential accepted: app=%q", credential.App)
		}
	}
	if err := ValidateClient(zrpc.RpcClientConf{}, service.ProMode); err == nil {
		t.Fatal("production client missing credentials was accepted")
	}
	if err := ValidateClient(zrpc.RpcClientConf{}, service.DevMode); err != nil {
		t.Fatal(err)
	}
	if err := ValidateClient(zrpc.RpcClientConf{App: "app"}, service.DevMode); err == nil {
		t.Fatal("partial credential was accepted")
	}
}
