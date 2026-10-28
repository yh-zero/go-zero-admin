package agentlogic

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/zeromicro/go-zero/zrpc"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/application/ai/rpc/pb"
	"go-zero-admin/application/applet/rpc/client/agenttools"
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
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fixtureOwner struct {
	ID        int64 `gorm:"primaryKey"`
	Enable    int64
	DeletedAt gorm.DeletedAt
}

func (fixtureOwner) TableName() string { return "sys_users" }

type fixtureClient struct {
	zrpc.Client
	conn *grpc.ClientConn
}

func (c fixtureClient) Conn() *grpc.ClientConn { return c.conn }

type businessStub struct {
	appletpb.UnimplementedUserServer
	appletpb.UnimplementedCasbinServer
	appletpb.UnimplementedAgentToolsServer
	revoked       atomic.Bool
	unavailable   atomic.Bool
	toolCalls     atomic.Int32
	sessionChecks atomic.Int32
	mu            sync.Mutex
	denied        map[string]bool
	lastActor     *appletpb.SessionRequest
	lastArguments string
	lastTool      string
	tool          func(context.Context, *appletpb.AgentToolRequest) (*appletpb.AgentToolResult, error)
}

func (b *businessStub) CheckSession(ctx context.Context, in *appletpb.SessionRequest) (*appletpb.CheckSessionResponse, error) {
	b.sessionChecks.Add(1)
	if b.unavailable.Load() {
		return nil, status.Error(codes.Unavailable, "private-business-address secret-credential")
	}
	b.mu.Lock()
	b.lastActor = proto.Clone(in).(*appletpb.SessionRequest)
	b.mu.Unlock()
	return &appletpb.CheckSessionResponse{Valid: !b.revoked.Load() && in.UserID > 0 && in.AuthorityId > 0 && in.SessionVersion == 1}, nil
}

func (b *businessStub) Enforce(ctx context.Context, in *appletpb.EnforceRequest) (*appletpb.EnforceResponse, error) {
	if b.unavailable.Load() {
		return nil, status.Error(codes.Unavailable, "private-business-address secret-credential")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return &appletpb.EnforceResponse{Pass: !b.denied[in.Path+" "+in.Method]}, nil
}

func (b *businessStub) deny(path, method string, denied bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.denied[path+" "+method] = denied
}

func (b *businessStub) call(ctx context.Context, name string, in *appletpb.AgentToolRequest) (*appletpb.AgentToolResult, error) {
	b.toolCalls.Add(1)
	b.mu.Lock()
	b.lastActor = proto.Clone(in.Actor).(*appletpb.SessionRequest)
	b.lastArguments, b.lastTool = in.ArgumentsJson, name
	call := b.tool
	b.mu.Unlock()
	if call != nil {
		return call(ctx, in)
	}
	return &appletpb.AgentToolResult{Content: `{"references":0,"status":"active"}`, Summary: "业务查询完成", Count: 1}, nil
}

func (b *businessStub) QueryAgentAudit(ctx context.Context, in *appletpb.AgentToolRequest) (*appletpb.AgentToolResult, error) {
	return b.call(ctx, "query_audit_logs", in)
}
func (b *businessStub) GetAgentFileStatus(ctx context.Context, in *appletpb.AgentToolRequest) (*appletpb.AgentToolResult, error) {
	return b.call(ctx, "get_file_status", in)
}
func (b *businessStub) ListAgentDevices(ctx context.Context, in *appletpb.AgentToolRequest) (*appletpb.AgentToolResult, error) {
	return b.call(ctx, "list_my_devices", in)
}

// AI fixtures have only task tables, transaction audit and the shared owner lock.
// Business data exists exclusively behind the actual bufconn gRPC boundary.
func fixture(t *testing.T) (*svc.ServiceContext, *pb.SessionRequest, *businessStub) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&fixtureOwner{}, &audit.Event{}, &agentjobs.Conversation{}, &agentjobs.Message{}, &agentjobs.Run{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]fixtureOwner{{ID: 10, Enable: 1}, {ID: 30, Enable: 1}}).Error; err != nil {
		t.Fatal(err)
	}
	b := &businessStub{denied: map[string]bool{}}
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	appletpb.RegisterUserServer(server, b)
	appletpb.RegisterCasbinServer(server, b)
	appletpb.RegisterAgentToolsServer(server, b)
	go func() { _ = server.Serve(listener) }()
	conn, err := grpc.NewClient("passthrough:///business", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close(); server.Stop(); _ = listener.Close() })
	client := fixtureClient{conn: conn}
	s := &svc.ServiceContext{DB: &orm.DB{DB: db}, AgentConfig: aiagent.Config{Enabled: true, MaxInputChars: 2000, MaxToolOutputBytes: 16384}, AppletUserRPC: user.NewUser(client), AppletCasbinRPC: casbin.NewCasbin(client), AppletAgentToolsRPC: agenttools.NewAgentTools(client)}
	s.AgentJobs, err = agentjobs.New(db, agentjobs.Config{Enabled: true}, func(ctx context.Context, j agentjobs.Job) (agentjobs.Result, error) { return execute(ctx, s, j) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.AgentJobs.Close)
	return s, &pb.SessionRequest{UserID: 10, AuthorityId: 2, SessionVersion: 1, SessionID: "10000000-0000-4000-8000-000000000099"}, b
}

func findTool(t *testing.T, s *svc.ServiceContext, a *pb.SessionRequest, name string) aiagent.Tool {
	t.Helper()
	tools, err := availableTools(context.Background(), s, a)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatal("expected tool is unavailable", name)
	return aiagent.Tool{}
}

func TestToolsCrossRealBusinessRPCAndRecheckPermissions(t *testing.T) {
	s, a, b := fixture(t)
	for _, item := range []struct{ name, args string }{{"query_audit_logs", `{}`}, {"get_file_status", `{"fileId":1}`}, {"list_my_devices", `{}`}} {
		tool := findTool(t, s, a, item.name)
		b.sessionChecks.Store(0)
		out, err := tool.Call(context.Background(), json.RawMessage(item.args))
		if err != nil || out.Count != 1 {
			t.Fatal(out, err)
		}
		if b.sessionChecks.Load() != 2 {
			t.Fatal("tool must verify the session once before and once after the RPC", b.sessionChecks.Load())
		}
		b.mu.Lock()
		if b.lastTool != item.name || b.lastArguments != item.args || b.lastActor.SessionID != a.SessionID || b.lastActor.SessionVersion != a.SessionVersion || b.lastActor.UserID != a.UserID || b.lastActor.AuthorityId != a.AuthorityId {
			t.Error("RPC lost actor or literal arguments")
		}
		b.mu.Unlock()
	}
	if b.toolCalls.Load() != 3 || s.DB.Migrator().HasTable("sys_file_resources") || s.DB.Migrator().HasTable("sys_device_sessions") {
		t.Fatal("AI queried a local business table")
	}
	file := findTool(t, s, a, "get_file_status")
	b.deny("/v1/sys/files/list", "GET", true)
	if _, err := file.Call(context.Background(), json.RawMessage(`{"fileId":1}`)); err == nil || b.toolCalls.Load() != 3 {
		t.Fatal("revoked permission reached remote tool")
	}
	b.deny("/v1/sys/files/list", "GET", false)
	b.mu.Lock()
	b.tool = func(context.Context, *appletpb.AgentToolRequest) (*appletpb.AgentToolResult, error) {
		b.revoked.Store(true)
		return &appletpb.AgentToolResult{Content: `{"private":"late result"}`}, nil
	}
	b.mu.Unlock()
	if out, err := file.Call(context.Background(), json.RawMessage(`{"fileId":1}`)); err == nil || out.Content != "" {
		t.Fatal("revoked session leaked a late tool result")
	}
}

func TestBusinessUnavailableStopsAgentAndHidesTransportDetails(t *testing.T) {
	s, a, b := fixture(t)
	b.unavailable.Store(true)
	_, err := NewGetAgentInfoLogic(context.Background(), s).GetAgentInfo(&pb.AgentInfoRequest{Actor: a})
	if err == nil || strings.Contains(err.Error(), "private-") || strings.Contains(err.Error(), "secret-") || !strings.Contains(err.Error(), "暂不可用") {
		t.Fatal("unsafe dependency failure", err)
	}
	if _, err := availableTools(context.Background(), s, a); err == nil {
		t.Fatal("dependency outage silently became an empty tool list")
	}
}

func TestRemoteToolRejectsMalformedArgumentsAndUnsafeResponse(t *testing.T) {
	s, a, b := fixture(t)
	file := findTool(t, s, a, "get_file_status")
	for _, raw := range []string{`null`, `[]`, `{"fileId":`, strings.Repeat(" ", 4097)} {
		if _, err := file.Call(context.Background(), json.RawMessage(raw)); !errors.Is(err, aiagent.ErrToolArguments) {
			t.Fatal("malformed args accepted", err)
		}
	}
	if b.toolCalls.Load() != 0 {
		t.Fatal("malformed argument crossed business RPC")
	}
	b.mu.Lock()
	b.tool = func(context.Context, *appletpb.AgentToolRequest) (*appletpb.AgentToolResult, error) {
		return nil, status.Error(codes.Internal, "private-object-key secret-key")
	}
	b.mu.Unlock()
	if out, err := file.Call(context.Background(), json.RawMessage(`{"fileId":1}`)); !errors.Is(err, aiagent.ErrToolFailed) || out.Content != "" || strings.Contains(err.Error(), "secret") {
		t.Fatal("remote error leaked", out, err)
	}
	for _, response := range []*appletpb.AgentToolResult{nil, {Content: `{}`, Summary: strings.Repeat("a", 1025)}, {Content: `{"data":"` + strings.Repeat("a", 16384) + `"}`}, {Content: `{}`, Count: -1}, {Content: `{}`, Count: 1000001}, {Content: `bad-json`}} {
		response := response
		b.mu.Lock()
		b.tool = func(context.Context, *appletpb.AgentToolRequest) (*appletpb.AgentToolResult, error) {
			return response, nil
		}
		b.mu.Unlock()
		if out, err := file.Call(context.Background(), json.RawMessage(`{"fileId":1}`)); !errors.Is(err, aiagent.ErrBudgetExceeded) || out.Content != "" {
			t.Fatal("unsafe remote output accepted", err)
		}
	}
}

func TestCancellationCrossesBusinessToolRPC(t *testing.T) {
	s, a, b := fixture(t)
	file := findTool(t, s, a, "get_file_status")
	started, cancelled := make(chan struct{}), make(chan struct{})
	b.mu.Lock()
	b.tool = func(ctx context.Context, _ *appletpb.AgentToolRequest) (*appletpb.AgentToolResult, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return nil, ctx.Err()
	}
	b.mu.Unlock()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := file.Call(ctx, json.RawMessage(`{"fileId":1}`)); result <- err }()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("tool RPC did not start")
	}
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("AI callback ignored cancellation")
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("business RPC did not receive cancellation")
	}
}

type eventRunner func(context.Context, aiagent.Request, aiagent.EventHandler) (aiagent.Result, error)

func (f eventRunner) Run(ctx context.Context, r aiagent.Request, h aiagent.EventHandler) (aiagent.Result, error) {
	return f(ctx, r, h)
}

func TestExecutionRevalidatesBeforeModelAndBeforeAcceptingAnswer(t *testing.T) {
	for _, atModel := range []bool{true, false} {
		t.Run(map[bool]string{true: "before_model", false: "before_save"}[atModel], func(t *testing.T) {
			s, a, b := fixture(t)
			s.AgentRunner = eventRunner(func(ctx context.Context, _ aiagent.Request, h aiagent.EventHandler) (aiagent.Result, error) {
				b.revoked.Store(true)
				if atModel {
					if err := h(ctx, aiagent.Event{Kind: "model_started"}); err != nil {
						return aiagent.Result{}, err
					}
					t.Error("revoked session reached the paid model")
				}
				return aiagent.Result{FinalAnswer: "late answer", Provider: "qwen", Model: "fixture", Usage: aiagent.Usage{PromptTokens: 10, CompletionTokens: 5}}, nil
			})
			out, err := execute(context.Background(), s, agentjobs.Job{Run: agentjobs.Run{OwnerID: a.UserID, AuthorityID: a.AuthorityId, SessionVersion: 1, SessionID: a.SessionID, Question: "查询"}})
			if err == nil || out.Text != "" {
				t.Fatal("revoked session result accepted", out, err)
			}
			if !atModel && out.InputTokens != 10 {
				t.Fatal("paid usage lost after answer rejected")
			}
		})
	}
}

func TestLongHistoryRetainsRecentCompleteTurns(t *testing.T) {
	s, a, _ := fixture(t)
	s.AgentRunner = eventRunner(func(_ context.Context, r aiagent.Request, _ aiagent.EventHandler) (aiagent.Result, error) {
		if historyBytes(r.History) > 64*1024 || len(r.History) != 9 || r.History[0].Role != "user" || r.History[len(r.History)-1].Content != "继续查询" {
			t.Fatal("context is not bounded by complete turns", len(r.History))
		}
		return aiagent.Result{FinalAnswer: "已查询"}, nil
	})
	history := make([]agentjobs.Message, 0, 12)
	for i := 0; i < 6; i++ {
		history = append(history, agentjobs.Message{Role: "user", Content: strings.Repeat("中", 2000)}, agentjobs.Message{Role: "assistant", Content: strings.Repeat("a", 8000)})
	}
	out, err := execute(context.Background(), s, agentjobs.Job{Run: agentjobs.Run{OwnerID: a.UserID, AuthorityID: a.AuthorityId, SessionVersion: 1, Question: "继续查询"}, History: history})
	if err != nil || out.Text == "" {
		t.Fatal(out, err)
	}
}

func TestDisabledRuntimeBootsWithoutAgentSchema(t *testing.T) {
	s, a, _ := fixture(t)
	s.AgentJobs.Close()
	s.AgentJobs = nil
	// Legacy environment switches cannot override the disabled service YAML.
	t.Setenv("AI_AGENT_ENABLED", "true")
	if err := s.DB.Migrator().DropTable(&agentjobs.Message{}, &agentjobs.Run{}, &agentjobs.Conversation{}); err != nil {
		t.Fatal(err)
	}
	if err := s.StartAgentRuntime(NewAgentExecutor(s)); err != nil {
		t.Fatal("disabled feature impacted startup", err)
	}
	t.Cleanup(s.AgentJobs.Close)
	info, err := NewGetAgentInfoLogic(context.Background(), s).GetAgentInfo(&pb.AgentInfoRequest{Actor: a})
	if err != nil || info.Enabled || info.Configured || info.MaxInputChars != 2000 {
		t.Fatal(info, err)
	}
	_, err = NewCreateAgentRunLogic(context.Background(), s).CreateAgentRun(&pb.CreateAgentRunRequest{Actor: a, RequestId: "10000000-0000-4000-8000-000000000001", Message: "查询设备"})
	if err == nil || !strings.Contains(err.Error(), "尚未启用") {
		t.Fatal("disabled request was not clear", err)
	}
}

func TestRuntimeRejectsRepeatedInitialization(t *testing.T) {
	s, _, _ := fixture(t)
	jobs := s.AgentJobs
	if err := s.StartAgentRuntime(NewAgentExecutor(s)); err == nil || s.AgentJobs != jobs {
		t.Fatal("runtime replaced an existing task manager", err)
	}
}
