package svc

import (
	"context"
	agentRPC "go-zero-admin/application/ai/rpc/client/agent"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/rpcsecurity"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AI is optional. Missing configuration or an offline AI service must not
// prevent ordinary business endpoints from starting.
func newAIAgentClient(c zrpc.RpcClientConf, mode string) agentRPC.Agent {
	if c.Target == "" && len(c.Endpoints) == 0 && len(c.Etcd.Hosts) == 0 {
		return nil
	}
	if rpcsecurity.ValidateClient(c, mode) != nil {
		logx.Error("AI RPC disabled: invalid client credentials")
		return nil
	}
	c.NonBlock = true
	// Validate synchronously, but do not create the go-zero Etcd subscriber here:
	// its initial read retries indefinitely even with NonBlock=true.
	if _, err := c.BuildTarget(); err != nil {
		logx.Error("AI RPC disabled: invalid client configuration")
		return nil
	}
	timeout := time.Duration(c.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return newAsyncAIAgentClient(timeout, func() agentRPC.Agent {
		delay := time.Second
		logged := false
		for {
			client, err := zrpc.NewClient(c, zrpc.WithUnaryClientInterceptor(audit.ClientInterceptor()))
			if err == nil {
				return agentRPC.NewAgent(client)
			}
			if !logged {
				logx.Error("AI RPC initialization failed; retrying in background")
				logged = true
			}
			time.Sleep(delay)
			delay = min(delay*2, 30*time.Second)
		}
	})
}

// One initializer is shared by every request. A go-zero subscriber waiting for
// unavailable Etcd cannot be canceled; it lives with the API process and can
// finish when discovery recovers. Requests never start additional initializers.
type asyncAIAgentClient struct {
	ready   chan struct{}
	client  agentRPC.Agent
	timeout time.Duration
}

var _ agentRPC.Agent = (*asyncAIAgentClient)(nil)

func newAsyncAIAgentClient(timeout time.Duration, initialize func() agentRPC.Agent) *asyncAIAgentClient {
	c := &asyncAIAgentClient{ready: make(chan struct{}), timeout: timeout}
	go func() {
		c.client = initialize()
		close(c.ready)
	}()
	return c
}

func callAI[T any](ctx context.Context, c *asyncAIAgentClient, call func(context.Context, agentRPC.Agent) (T, error)) (T, error) {
	var zero T
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	select {
	case <-ctx.Done():
		return zero, status.FromContextError(ctx.Err()).Err()
	case <-c.ready:
	}
	if err := ctx.Err(); err != nil {
		return zero, status.FromContextError(err).Err()
	}
	if c.client == nil {
		return zero, status.Error(codes.Unavailable, "AI RPC client is unavailable")
	}
	return call(ctx, c.client)
}

func (c *asyncAIAgentClient) GetAgentInfo(ctx context.Context, in *pb.AgentInfoRequest, opts ...grpc.CallOption) (*pb.AgentInfo, error) {
	return callAI(ctx, c, func(ctx context.Context, client agentRPC.Agent) (*pb.AgentInfo, error) {
		return client.GetAgentInfo(ctx, in, opts...)
	})
}

func (c *asyncAIAgentClient) CreateAgentRun(ctx context.Context, in *pb.CreateAgentRunRequest, opts ...grpc.CallOption) (*pb.AgentRun, error) {
	return callAI(ctx, c, func(ctx context.Context, client agentRPC.Agent) (*pb.AgentRun, error) {
		return client.CreateAgentRun(ctx, in, opts...)
	})
}

func (c *asyncAIAgentClient) GetAgentRun(ctx context.Context, in *pb.AgentIDRequest, opts ...grpc.CallOption) (*pb.AgentRun, error) {
	return callAI(ctx, c, func(ctx context.Context, client agentRPC.Agent) (*pb.AgentRun, error) {
		return client.GetAgentRun(ctx, in, opts...)
	})
}

func (c *asyncAIAgentClient) CancelAgentRun(ctx context.Context, in *pb.AgentIDRequest, opts ...grpc.CallOption) (*pb.AgentRun, error) {
	return callAI(ctx, c, func(ctx context.Context, client agentRPC.Agent) (*pb.AgentRun, error) {
		return client.CancelAgentRun(ctx, in, opts...)
	})
}

func (c *asyncAIAgentClient) ListAgentConversations(ctx context.Context, in *pb.AgentPageRequest, opts ...grpc.CallOption) (*pb.AgentConversationListResponse, error) {
	return callAI(ctx, c, func(ctx context.Context, client agentRPC.Agent) (*pb.AgentConversationListResponse, error) {
		return client.ListAgentConversations(ctx, in, opts...)
	})
}

func (c *asyncAIAgentClient) ListAgentMessages(ctx context.Context, in *pb.AgentMessageListRequest, opts ...grpc.CallOption) (*pb.AgentMessageListResponse, error) {
	return callAI(ctx, c, func(ctx context.Context, client agentRPC.Agent) (*pb.AgentMessageListResponse, error) {
		return client.ListAgentMessages(ctx, in, opts...)
	})
}
