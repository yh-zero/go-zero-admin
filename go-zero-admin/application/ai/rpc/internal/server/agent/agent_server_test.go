package server

import (
	"context"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/application/applet/rpc/client/casbin"
	"go-zero-admin/application/applet/rpc/client/user"
	appletpb "go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
	"go-zero-admin/pkg/aiagent"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/orm"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type sessionClient struct {
	user.User
	unavailable atomic.Bool
}

func (s *sessionClient) CheckSession(ctx context.Context, in *appletpb.SessionRequest, _ ...grpc.CallOption) (*appletpb.CheckSessionResponse, error) {
	if s.unavailable.Load() {
		return nil, status.Error(codes.Unavailable, "secret-database-address")
	}
	return &appletpb.CheckSessionResponse{Valid: in.UserID > 0 && in.AuthorityId > 0 && in.SessionVersion == 1}, ctx.Err()
}

type permissionClient struct{ casbin.Casbin }

func (*permissionClient) Enforce(ctx context.Context, _ *appletpb.EnforceRequest, _ ...grpc.CallOption) (*appletpb.EnforceResponse, error) {
	return &appletpb.EnforceResponse{Pass: true}, ctx.Err()
}

type answerRunner struct{}

func (answerRunner) Run(context.Context, aiagent.Request, aiagent.EventHandler) (aiagent.Result, error) {
	return aiagent.Result{FinalAnswer: "只读检查完成", Provider: "qwen", Model: "fixture", Usage: aiagent.Usage{PromptTokens: 7, CompletionTokens: 3}}, nil
}

type ownerRow struct {
	ID        int64 `gorm:"primaryKey"`
	Enable    int64
	DeletedAt gorm.DeletedAt
}

func (ownerRow) TableName() string { return "sys_users" }

// Exercises the generated ai.Agent registration, protobuf messages and all six
// methods. Task persistence is real; business authorization is a safe stub.
func TestIndependentAIServiceActualRPCFlowAndUnavailable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	defer sqlDB.Close()
	if err := db.AutoMigrate(&ownerRow{}, &audit.Event{}, &agentjobs.Conversation{}, &agentjobs.Message{}, &agentjobs.Run{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]ownerRow{{ID: 10, Enable: 1}, {ID: 30, Enable: 1}}).Error; err != nil {
		t.Fatal(err)
	}
	users := &sessionClient{}
	s := &svc.ServiceContext{DB: &orm.DB{DB: db}, AgentConfig: aiagent.Config{Enabled: true, MaxInputChars: 2000}, AgentRunner: answerRunner{}, AppletUserRPC: users, AppletCasbinRPC: &permissionClient{}}
	s.AgentJobs, err = agentjobs.New(db, agentjobs.Config{Enabled: true, PollInterval: 5 * time.Millisecond}, func(ctx context.Context, j agentjobs.Job) (agentjobs.Result, error) {
		return agentjobs.Result{Text: "只读检查完成", Provider: "qwen", Model: "fixture", InputTokens: 7, OutputTokens: 3, ExtraJSON: "[]"}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.AgentJobs.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.AgentJobs.Close()
	listener := bufconn.Listen(1 << 20)
	defer listener.Close()
	server := grpc.NewServer()
	pb.RegisterAgentServer(server, NewAgentServer(s))
	defer server.Stop()
	go func() { _ = server.Serve(listener) }()
	conn, err := grpc.NewClient("passthrough:///ai", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	client := pb.NewAgentClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	a := &pb.SessionRequest{UserID: 10, AuthorityId: 2, SessionVersion: 1, SessionID: "10000000-0000-4000-8000-000000000099"}
	info, err := client.GetAgentInfo(ctx, &pb.AgentInfoRequest{Actor: a})
	if err != nil || !info.Enabled || !info.Configured || len(info.Tools) != 3 {
		t.Fatal(info, err)
	}
	created, err := client.CreateAgentRun(ctx, &pb.CreateAgentRunRequest{Actor: a, Message: "检查我的设备", RequestId: "10000000-0000-4000-8000-000000000001"})
	if err != nil {
		t.Fatal(err)
	}
	var run *pb.AgentRun
	for {
		run, err = client.GetAgentRun(ctx, &pb.AgentIDRequest{Actor: a, ID: created.ID})
		if err != nil {
			t.Fatal(err)
		}
		if agentjobs.IsTerminal(run.Status) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if run.Status != agentjobs.StatusSucceeded || run.Answer == "" || run.InputTokens != 7 {
		t.Fatal(run)
	}
	conversations, err := client.ListAgentConversations(ctx, &pb.AgentPageRequest{Actor: a, PageNo: 1, PageSize: 20})
	if err != nil || conversations.Total != 1 {
		t.Fatal(conversations, err)
	}
	messages, err := client.ListAgentMessages(ctx, &pb.AgentMessageListRequest{Actor: a, ID: run.ConversationId, PageNo: 1, PageSize: 20})
	if err != nil || messages.Total != 2 {
		t.Fatal(messages, err)
	}
	stopped, err := client.CancelAgentRun(ctx, &pb.AgentIDRequest{Actor: a, ID: run.ID})
	if err != nil || stopped.Status != agentjobs.StatusSucceeded {
		t.Fatal(stopped, err)
	}
	other := &pb.SessionRequest{UserID: 30, AuthorityId: 1, SessionVersion: 1}
	if _, err := client.GetAgentRun(ctx, &pb.AgentIDRequest{Actor: other, ID: run.ID}); err == nil {
		t.Fatal("admin read another owner's task")
	}
	cancelledCtx, cancelRequest := context.WithCancel(ctx)
	cancelRequest()
	if _, err := client.GetAgentInfo(cancelledCtx, &pb.AgentInfoRequest{Actor: a}); status.Code(err) != codes.Canceled {
		t.Fatal("cancelled gRPC call accepted", err)
	}
	users.unavailable.Store(true)
	if _, err := client.GetAgentInfo(ctx, &pb.AgentInfoRequest{Actor: a}); err == nil || strings.Contains(err.Error(), "secret-") || !strings.Contains(err.Error(), "暂不可用") {
		t.Fatal("unsafe unavailable dependency response", err)
	}
}
