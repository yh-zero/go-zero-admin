package svc

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	agentRPC "go-zero-admin/application/ai/rpc/client/agent"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/discov"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
	etcdpb "go.etcd.io/etcd/api/v3/etcdserverpb"
	"go.etcd.io/etcd/api/v3/mvccpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestAIInitializationWaitUsesOneWorkerAndBoundsEveryEndpoint(t *testing.T) {
	var initialized atomic.Int32
	unblock := make(chan struct{})
	c := newAsyncAIAgentClient(25*time.Millisecond, func() agentRPC.Agent {
		initialized.Add(1)
		<-unblock
		return nil
	})
	defer close(unblock)
	requests := []func(context.Context) error{
		func(ctx context.Context) error { _, err := c.GetAgentInfo(ctx, &pb.AgentInfoRequest{}); return err },
		func(ctx context.Context) error {
			_, err := c.CreateAgentRun(ctx, &pb.CreateAgentRunRequest{})
			return err
		},
		func(ctx context.Context) error { _, err := c.GetAgentRun(ctx, &pb.AgentIDRequest{}); return err },
		func(ctx context.Context) error { _, err := c.CancelAgentRun(ctx, &pb.AgentIDRequest{}); return err },
		func(ctx context.Context) error {
			_, err := c.ListAgentConversations(ctx, &pb.AgentPageRequest{})
			return err
		},
		func(ctx context.Context) error {
			_, err := c.ListAgentMessages(ctx, &pb.AgentMessageListRequest{})
			return err
		},
	}
	var pending sync.WaitGroup
	for _, request := range requests {
		pending.Add(1)
		go func() {
			defer pending.Done()
			if err := request(context.Background()); status.Code(err) != codes.DeadlineExceeded {
				t.Errorf("uninitialized endpoint did not respect RPC timeout: %v", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			started := time.Now()
			if err := request(ctx); status.Code(err) != codes.Canceled || time.Since(started) > 100*time.Millisecond {
				t.Errorf("uninitialized endpoint did not respect cancellation: %v", err)
			}
		}()
	}
	pending.Wait()
	if initialized.Load() != 1 {
		t.Fatalf("concurrent requests started %d initializers", initialized.Load())
	}
}

type deadlineAIAgent struct {
	agentRPC.Agent
	deadline chan time.Time
}

func (c *deadlineAIAgent) GetAgentInfo(ctx context.Context, _ *pb.AgentInfoRequest, _ ...grpc.CallOption) (*pb.AgentInfo, error) {
	deadline, _ := ctx.Deadline()
	c.deadline <- deadline
	<-ctx.Done()
	return nil, status.FromContextError(ctx.Err()).Err()
}

func TestAIInitializationAndRPCShareOneTimeoutBudget(t *testing.T) {
	observed := make(chan time.Time, 1)
	c := newAsyncAIAgentClient(100*time.Millisecond, func() agentRPC.Agent {
		time.Sleep(40 * time.Millisecond)
		return &deadlineAIAgent{deadline: observed}
	})
	started := time.Now()
	if _, err := c.GetAgentInfo(context.Background(), &pb.AgentInfoRequest{}); status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("underlying RPC did not retain timeout: %v", err)
	}
	deadline := <-observed
	if deadline.Sub(started) > 120*time.Millisecond {
		t.Fatal("RPC timeout restarted after waiting for initialization")
	}
}

type recoveringAIDiscovery struct {
	etcdpb.UnimplementedKVServer
	etcdpb.UnimplementedMaintenanceServer
	etcdpb.UnimplementedWatchServer
	online   atomic.Bool
	checks   atomic.Int32
	endpoint string
	failed   chan struct{}
	once     sync.Once
}

func (d *recoveringAIDiscovery) Status(context.Context, *etcdpb.StatusRequest) (*etcdpb.StatusResponse, error) {
	d.checks.Add(1)
	if !d.online.Load() {
		d.once.Do(func() { close(d.failed) })
		return nil, status.Error(codes.FailedPrecondition, "discovery is recovering")
	}
	return &etcdpb.StatusResponse{Version: "3.5.21"}, nil
}

func (d *recoveringAIDiscovery) Range(_ context.Context, in *etcdpb.RangeRequest) (*etcdpb.RangeResponse, error) {
	return &etcdpb.RangeResponse{Header: &etcdpb.ResponseHeader{Revision: 1}, Count: 1, Kvs: []*mvccpb.KeyValue{{Key: append(in.Key, []byte("instance")...), Value: []byte(d.endpoint)}}}, nil
}

func (d *recoveringAIDiscovery) Watch(stream etcdpb.Watch_WatchServer) error {
	if _, err := stream.Recv(); err != nil {
		return err
	}
	if err := stream.Send(&etcdpb.WatchResponse{Header: &etcdpb.ResponseHeader{Revision: 1}, WatchId: 1, Created: true}); err != nil {
		return err
	}
	<-stream.Context().Done()
	return stream.Context().Err()
}

type recoveredAIServer struct {
	pb.UnimplementedAgentServer
	actor atomic.Pointer[pb.SessionRequest]
}

func (s *recoveredAIServer) GetAgentInfo(_ context.Context, in *pb.AgentInfoRequest) (*pb.AgentInfo, error) {
	s.actor.Store(proto.Clone(in.Actor).(*pb.SessionRequest))
	return &pb.AgentInfo{Provider: "test-provider"}, nil
}

func (s *recoveredAIServer) CreateAgentRun(context.Context, *pb.CreateAgentRunRequest) (*pb.AgentRun, error) {
	return nil, status.Error(codes.Code(xerr.REUQEST_PARAM_ERROR), "保留业务错误")
}

func TestAIClientRecoversDiscoveryWithoutRecreatingClient(t *testing.T) {
	listen := func() net.Listener {
		t.Helper()
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		return listener
	}
	aiListener, discoveryListener := listen(), listen()
	aiService := &recoveredAIServer{}
	aiServer := grpc.NewServer()
	pb.RegisterAgentServer(aiServer, aiService)
	go func() { _ = aiServer.Serve(aiListener) }()
	t.Cleanup(aiServer.Stop)
	discovery := &recoveringAIDiscovery{endpoint: aiListener.Addr().String(), failed: make(chan struct{})}
	discoveryServer := grpc.NewServer()
	etcdpb.RegisterKVServer(discoveryServer, discovery)
	etcdpb.RegisterMaintenanceServer(discoveryServer, discovery)
	etcdpb.RegisterWatchServer(discoveryServer, discovery)
	go func() { _ = discoveryServer.Serve(discoveryListener) }()
	t.Cleanup(discoveryServer.Stop)
	var c zrpc.RpcClientConf
	if err := conf.FillDefault(&c); err != nil {
		t.Fatal(err)
	}
	c.Etcd = discov.EtcdConf{Hosts: []string{discoveryListener.Addr().String()}, Key: "recovering.ai.rpc"}
	c.Timeout = 3000
	started := time.Now()
	client := newAIAgentClient(c, service.DevMode)
	if client == nil || time.Since(started) > time.Second {
		t.Fatal("unavailable discovery blocked API initialization")
	}
	select {
	case <-discovery.failed:
	case <-time.After(2 * time.Second):
		t.Fatal("test discovery was not reached")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, err := client.GetAgentInfo(ctx, &pb.AgentInfoRequest{})
	cancel()
	if status.Code(err) != codes.DeadlineExceeded {
		t.Fatalf("discovery wait ignored request deadline: %v", err)
	}
	discovery.online.Store(true)
	actor := &pb.SessionRequest{UserID: 27, AuthorityId: 4, SessionVersion: 9, SessionID: "trusted-test-device"}
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	info, err := client.GetAgentInfo(ctx, &pb.AgentInfoRequest{Actor: actor}, grpc.WaitForReady(true))
	if err != nil || info.Provider != "test-provider" || !proto.Equal(aiService.actor.Load(), actor) {
		t.Fatalf("original AI client did not recover with trusted actor: info=%v err=%v", info, err)
	}
	if discovery.checks.Load() < 2 {
		t.Fatal("recovery did not retry failed initialization")
	}
	if _, err := client.CreateAgentRun(ctx, &pb.CreateAgentRunRequest{Actor: actor}); status.Code(err) != codes.Code(xerr.REUQEST_PARAM_ERROR) || status.Convert(err).Message() != "保留业务错误" {
		t.Fatalf("wrapper changed business RPC errors: %v", err)
	}
}
