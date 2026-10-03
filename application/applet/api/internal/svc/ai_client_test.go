package svc

import (
	"context"
	"net"
	"testing"
	"time"

	"go-zero-admin/application/ai/rpc/pb"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
)

func TestAIClientAbsentOrInvalidConfigurationIsOptional(t *testing.T) {
	for _, c := range []zrpc.RpcClientConf{
		{},
		{Etcd: discov.EtcdConf{Hosts: []string{"127.0.0.1:2379"}}},
		{Endpoints: []string{"127.0.0.1:6002"}, App: "bad app", Token: "invalid"},
	} {
		if client := newAIAgentClient(c, service.DevMode); client != nil {
			t.Fatal("absent/invalid AI configuration must disable only the AI client")
		}
	}
	if client := newAIAgentClient(zrpc.RpcClientConf{Endpoints: []string{"127.0.0.1:6002"}}, service.ProMode); client != nil {
		t.Fatal("production AI must not connect without service credentials")
	}
}

func TestAIClientOfflineDoesNotBlockInitializationEvenWithBlockingConfig(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := listener.Addr().String()
	_ = listener.Close()
	var c zrpc.RpcClientConf
	if err := conf.FillDefault(&c); err != nil {
		t.Fatal(err)
	}
	c.Endpoints, c.NonBlock = []string{endpoint}, false
	started := time.Now()
	client := newAIAgentClient(c, service.DevMode)
	if client == nil || time.Since(started) > time.Second {
		t.Fatal("offline AI must not prevent API initialization")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if _, err := client.GetAgentInfo(ctx, &pb.AgentInfoRequest{}); err == nil {
		t.Fatal("offline AI calls must return an error")
	}
}

func TestAIClientOfflineDiscoveryDoesNotBlockInitialization(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	endpoint := listener.Addr().String()
	_ = listener.Close()
	var c zrpc.RpcClientConf
	if err := conf.FillDefault(&c); err != nil {
		t.Fatal(err)
	}
	c.Etcd = discov.EtcdConf{Hosts: []string{endpoint}, Key: "unavailable.ai.rpc"}
	ready := make(chan bool, 1)
	go func() { ready <- newAIAgentClient(c, service.DevMode) != nil }()
	select {
	case available := <-ready:
		if !available {
			t.Fatal("valid AI configuration must remain reconnectable")
		}
	case <-time.After(time.Second):
		t.Fatal("offline AI discovery must not prevent API initialization")
	}
}
