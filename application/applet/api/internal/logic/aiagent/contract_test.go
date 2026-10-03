package aiagent_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"go-zero-admin/application/ai/rpc/client/agent"
	"go-zero-admin/application/ai/rpc/pb"
	agenthandler "go-zero-admin/application/applet/api/internal/handler/aiagent"
	agentlogic "go-zero-admin/application/applet/api/internal/logic/aiagent"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/ctxJwt"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/rest/pathvar"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type agentContractClient struct {
	requests []proto.Message
	contexts []context.Context
	failure  error
	empty    bool
}

var _ agent.Agent = (*agentContractClient)(nil)

func (c *agentContractClient) record(ctx context.Context, in proto.Message) {
	c.requests = append(c.requests, in)
	c.contexts = append(c.contexts, ctx)
}
func (c *agentContractClient) GetAgentInfo(ctx context.Context, in *pb.AgentInfoRequest, _ ...grpc.CallOption) (*pb.AgentInfo, error) {
	c.record(ctx, in)
	if c.failure != nil {
		return nil, c.failure
	}
	if c.empty {
		return &pb.AgentInfo{}, nil
	}
	return &pb.AgentInfo{Enabled: true, Configured: true, Provider: "deepseek", Model: "test-model", MaxInputChars: 2000, MaxSteps: 8, MaxRunSeconds: 120, Tools: []*pb.AgentToolInfo{{Name: "query_audit_logs", Label: "审计日志", Description: "授权只读查询", Available: true}}}, nil
}
func (c *agentContractClient) run() (*pb.AgentRun, error) {
	if c.failure != nil {
		return nil, c.failure
	}
	if c.empty {
		return &pb.AgentRun{}, nil
	}
	return &pb.AgentRun{ID: "run-id", ConversationId: "conversation-id", RequestId: "request-id", Status: "succeeded", Answer: "实际答案", Error: "安全错误摘要", Provider: "deepseek", Model: "test-model", InputTokens: 17, OutputTokens: 23, CreatedAt: "2026-10-03T01:02:03Z", UpdatedAt: "2026-10-03T01:02:04Z", ToolCalls: []*pb.AgentToolCall{{Name: "query_audit_logs", Status: "failed", Error: "工具安全错误", Summary: "3条记录"}}}, nil
}
func (c *agentContractClient) CreateAgentRun(ctx context.Context, in *pb.CreateAgentRunRequest, _ ...grpc.CallOption) (*pb.AgentRun, error) {
	c.record(ctx, in)
	return c.run()
}
func (c *agentContractClient) GetAgentRun(ctx context.Context, in *pb.AgentIDRequest, _ ...grpc.CallOption) (*pb.AgentRun, error) {
	c.record(ctx, in)
	return c.run()
}
func (c *agentContractClient) CancelAgentRun(ctx context.Context, in *pb.AgentIDRequest, _ ...grpc.CallOption) (*pb.AgentRun, error) {
	c.record(ctx, in)
	return c.run()
}
func (c *agentContractClient) ListAgentConversations(ctx context.Context, in *pb.AgentPageRequest, _ ...grpc.CallOption) (*pb.AgentConversationListResponse, error) {
	c.record(ctx, in)
	if c.failure != nil {
		return nil, c.failure
	}
	if c.empty {
		return &pb.AgentConversationListResponse{}, nil
	}
	return &pb.AgentConversationListResponse{Total: 41, Items: []*pb.AgentConversation{{ID: "conversation-id", Title: "审计问题", CreatedAt: "2026-10-03T01:02:03Z", UpdatedAt: "2026-10-03T01:02:04Z"}}}, nil
}
func (c *agentContractClient) ListAgentMessages(ctx context.Context, in *pb.AgentMessageListRequest, _ ...grpc.CallOption) (*pb.AgentMessageListResponse, error) {
	c.record(ctx, in)
	if c.failure != nil {
		return nil, c.failure
	}
	if c.empty {
		return &pb.AgentMessageListResponse{}, nil
	}
	return &pb.AgentMessageListResponse{Total: 21, Items: []*pb.AgentMessage{{ID: "message-id", ConversationId: "conversation-id", RunId: "run-id", Role: "assistant", Content: "历史答案", CreatedAt: "2026-10-03T01:02:04Z"}}}, nil
}

func agentJWTContext() context.Context {
	// json.Number matches go-zero's decoded claims; no live token is needed.
	return context.WithValue(context.Background(), ctxJwt.CtxKeyJwtData, map[string]any{
		"ID": json.Number("27"), "AuthorityId": json.Number("4"),
		"SessionVersion": json.Number("9"), "SessionID": "trusted-device",
	})
}
func trustedActor() *pb.SessionRequest {
	return &pb.SessionRequest{UserID: 27, AuthorityId: 4, SessionVersion: 9, SessionID: "trusted-device"}
}
func assertJSON(t *testing.T, actual any, expected string) {
	t.Helper()
	data, err := json.Marshal(actual)
	if err != nil {
		t.Fatal(err)
	}
	var got, want any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(expected), &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("JSON contract changed:\n got %s\nwant %s", data, expected)
	}
}

const runJSON = `{"id":"run-id","conversationId":"conversation-id","requestId":"request-id","status":"succeeded","answer":"实际答案","error":"安全错误摘要","provider":"deepseek","model":"test-model","inputTokens":17,"outputTokens":23,"createdAt":"2026-10-03T01:02:03Z","updatedAt":"2026-10-03T01:02:04Z","toolCalls":[{"name":"query_audit_logs","status":"failed","error":"工具安全错误","summary":"3条记录"}]}`

func TestAgentSixRPCRequestsUseTrustedJWTAndConvertAllDTOFields(t *testing.T) {
	ctx := agentJWTContext()
	client := &agentContractClient{}
	s := &svc.ServiceContext{AIAgentRPC: client}
	cases := []struct {
		name   string
		invoke func() (any, error)
		input  proto.Message
		json   string
	}{
		{"info", func() (any, error) { return agentlogic.NewGetAgentInfoLogic(ctx, s).GetAgentInfo() }, &pb.AgentInfoRequest{Actor: trustedActor()}, `{"enabled":true,"configured":true,"provider":"deepseek","model":"test-model","tools":[{"name":"query_audit_logs","label":"审计日志","description":"授权只读查询","available":true}],"maxInputChars":2000,"maxSteps":8,"maxRunSeconds":120}`},
		{"create", func() (any, error) {
			return agentlogic.NewCreateAgentRunLogic(ctx, s).CreateAgentRun(&types.CreateAgentRunRequest{ConversationId: "conversation-id", RequestId: "request-id", Message: "查询审计"})
		}, &pb.CreateAgentRunRequest{Actor: trustedActor(), ConversationId: "conversation-id", RequestId: "request-id", Message: "查询审计"}, runJSON},
		{"get", func() (any, error) {
			return agentlogic.NewGetAgentRunLogic(ctx, s).GetAgentRun(&types.AgentIDRequest{ID: "run-id"})
		}, &pb.AgentIDRequest{Actor: trustedActor(), ID: "run-id"}, runJSON},
		{"cancel", func() (any, error) {
			return agentlogic.NewCancelAgentRunLogic(ctx, s).CancelAgentRun(&types.AgentIDRequest{ID: "run-id"})
		}, &pb.AgentIDRequest{Actor: trustedActor(), ID: "run-id"}, runJSON},
		{"conversations", func() (any, error) {
			return agentlogic.NewListAgentConversationsLogic(ctx, s).ListAgentConversations(&types.AgentPageRequest{PageNo: 3, PageSize: 20})
		}, &pb.AgentPageRequest{Actor: trustedActor(), PageNo: 3, PageSize: 20}, `{"items":[{"id":"conversation-id","title":"审计问题","createdAt":"2026-10-03T01:02:03Z","updatedAt":"2026-10-03T01:02:04Z"}],"total":41}`},
		{"messages", func() (any, error) {
			return agentlogic.NewListAgentMessagesLogic(ctx, s).ListAgentMessages(&types.AgentMessageListRequest{ID: "conversation-id", PageNo: 2, PageSize: 20})
		}, &pb.AgentMessageListRequest{Actor: trustedActor(), ID: "conversation-id", PageNo: 2, PageSize: 20}, `{"items":[{"id":"message-id","conversationId":"conversation-id","runId":"run-id","role":"assistant","content":"历史答案","createdAt":"2026-10-03T01:02:04Z"}],"total":21}`},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			before := len(client.requests)
			result, err := test.invoke()
			if err != nil {
				t.Fatal(err)
			}
			if len(client.requests) != before+1 || !proto.Equal(client.requests[before], test.input) {
				t.Fatalf("RPC request lost trusted actor or fields: got=%v want=%v", client.requests, test.input)
			}
			if client.contexts[before] != ctx {
				t.Fatal("request context was dropped")
			}
			assertJSON(t, result, test.json)
		})
	}
}

func TestAgentCreateHandlerIgnoresForgedActorAndKeepsFirstConversationOptional(t *testing.T) {
	for _, body := range []string{
		`{"requestId":"request-id","message":"查询审计","actor":{"userID":999,"authorityId":1},"userId":999,"sessionID":"forged-device"}`,
		`{"conversationId":"","requestId":"request-id","message":"查询审计"}`,
	} {
		client := &agentContractClient{}
		r := httptest.NewRequest(http.MethodPost, "/v1/ai/runs", strings.NewReader(body)).WithContext(agentJWTContext())
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		agenthandler.CreateAgentRunHandler(&svc.ServiceContext{AIAgentRPC: client})(w, r)
		want := &pb.CreateAgentRunRequest{Actor: trustedActor(), RequestId: "request-id", Message: "查询审计"}
		if w.Code != http.StatusOK || len(client.requests) != 1 || !proto.Equal(client.requests[0], want) {
			t.Fatalf("trusted identity/optional conversation changed: %s requests=%v", w.Body.String(), client.requests)
		}
		var envelope struct {
			Code   int             `json:"code"`
			Result json.RawMessage `json:"result"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
			t.Fatal(err)
		}
		if envelope.Code != 200 {
			t.Fatalf("success envelope changed: %s", w.Body.String())
		}
		assertJSON(t, envelope.Result, runJSON)
	}
}

func TestAgentHandlersKeepPathIDAndHistoryPaginationDefaults(t *testing.T) {
	client := &agentContractClient{empty: true}
	s := &svc.ServiceContext{AIAgentRPC: client}
	cases := []struct {
		name, method, path, id string
		handler                http.HandlerFunc
		expected               proto.Message
	}{
		{"get", http.MethodGet, "/v1/ai/runs/path-id?ID=forged", "path-id", agenthandler.GetAgentRunHandler(s), &pb.AgentIDRequest{Actor: trustedActor(), ID: "path-id"}},
		{"cancel", http.MethodPost, "/v1/ai/runs/path-id/cancel?ID=forged", "path-id", agenthandler.CancelAgentRunHandler(s), &pb.AgentIDRequest{Actor: trustedActor(), ID: "path-id"}},
		{"conversation-defaults", http.MethodGet, "/v1/ai/conversations", "", agenthandler.ListAgentConversationsHandler(s), &pb.AgentPageRequest{Actor: trustedActor(), PageNo: 1, PageSize: 20}},
		{"message-defaults", http.MethodGet, "/v1/ai/conversations/path-id/messages", "path-id", agenthandler.ListAgentMessagesHandler(s), &pb.AgentMessageListRequest{Actor: trustedActor(), ID: "path-id", PageNo: 1, PageSize: 20}},
		{"message-explicit", http.MethodGet, "/v1/ai/conversations/path-id/messages?pageNo=4&pageSize=50", "path-id", agenthandler.ListAgentMessagesHandler(s), &pb.AgentMessageListRequest{Actor: trustedActor(), ID: "path-id", PageNo: 4, PageSize: 50}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest(test.method, test.path, nil).WithContext(agentJWTContext())
			if test.id != "" {
				r = pathvar.WithVars(r, map[string]string{"id": test.id})
			}
			before := len(client.requests)
			w := httptest.NewRecorder()
			test.handler(w, r)
			if w.Code != http.StatusOK || len(client.requests) != before+1 {
				t.Fatalf("request rejected: %s", w.Body.String())
			}
			if !proto.Equal(client.requests[before], test.expected) {
				t.Fatalf("path/defaults mismatch got=%v want=%v", client.requests[before], test.expected)
			}
		})
	}
}

func TestAgentAllHandlersPropagateBusinessErrors(t *testing.T) {
	failure := status.Convert(xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "AI请求参数无效")).Err()
	client := &agentContractClient{failure: failure}
	s := &svc.ServiceContext{AIAgentRPC: client}
	for _, handler := range []http.HandlerFunc{agenthandler.GetAgentInfoHandler(s), agenthandler.CreateAgentRunHandler(s), agenthandler.GetAgentRunHandler(s), agenthandler.CancelAgentRunHandler(s), agenthandler.ListAgentConversationsHandler(s), agenthandler.ListAgentMessagesHandler(s)} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/v1/ai/contract", strings.NewReader(`{"requestId":"request-id","message":"查询审计"}`)).WithContext(agentJWTContext())
		r.Header.Set("Content-Type", "application/json")
		r = pathvar.WithVars(r, map[string]string{"id": "path-id"})
		before := len(client.requests)
		handler(w, r)
		var result struct {
			Code    uint32 `json:"code"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if w.Code != http.StatusOK || len(client.requests) != before+1 || result.Code != xerr.REUQEST_PARAM_ERROR || result.Message != "AI请求参数无效" {
			t.Fatalf("business error lost: %s", w.Body.String())
		}
	}
}

func TestAgentHandlersHandleEmptyPathWithoutPanicking(t *testing.T) {
	failure := status.Convert(xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "AI请求参数无效")).Err()
	for _, endpoint := range []struct {
		name    string
		factory func(*svc.ServiceContext) http.HandlerFunc
	}{
		{"get", agenthandler.GetAgentRunHandler}, {"cancel", agenthandler.CancelAgentRunHandler}, {"messages", agenthandler.ListAgentMessagesHandler},
	} {
		for _, vars := range []map[string]string{nil, {"id": ""}} {
			t.Run(endpoint.name, func(t *testing.T) {
				client := &agentContractClient{failure: failure}
				w := httptest.NewRecorder()
				r := httptest.NewRequest(http.MethodGet, "/v1/ai/invalid", nil).WithContext(agentJWTContext())
				if vars != nil {
					r = pathvar.WithVars(r, vars)
				}
				endpoint.factory(&svc.ServiceContext{AIAgentRPC: client})(w, r)
				if w.Code >= http.StatusInternalServerError {
					t.Fatalf("empty ID caused server error: %s", w.Body.String())
				}
				if len(client.requests) > 0 {
					switch request := client.requests[0].(type) {
					case *pb.AgentIDRequest:
						if request.ID != "" {
							t.Fatal("empty path ID was replaced")
						}
					case *pb.AgentMessageListRequest:
						if request.ID != "" {
							t.Fatal("empty path ID was replaced")
						}
					}
					var result struct {
						Code    uint32 `json:"code"`
						Message string `json:"message"`
					}
					if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
						t.Fatal(err)
					}
					if result.Code != xerr.REUQEST_PARAM_ERROR || result.Message != "AI请求参数无效" {
						t.Fatalf("invalid-ID error lost: %s", w.Body.String())
					}
				}
			})
		}
	}
}

func TestAgentEmptyCollectionsSerializeAsArrays(t *testing.T) {
	client := &agentContractClient{empty: true}
	s := &svc.ServiceContext{AIAgentRPC: client}
	ctx := agentJWTContext()
	info, err := agentlogic.NewGetAgentInfoLogic(ctx, s).GetAgentInfo()
	if err != nil {
		t.Fatal(err)
	}
	if info.Tools == nil {
		t.Fatal("tools must serialize as [], not null")
	}
	run, err := agentlogic.NewGetAgentRunLogic(ctx, s).GetAgentRun(&types.AgentIDRequest{ID: "run-id"})
	if err != nil {
		t.Fatal(err)
	}
	if run.ToolCalls == nil {
		t.Fatal("toolCalls must serialize as [], not null")
	}
	conversations, err := agentlogic.NewListAgentConversationsLogic(ctx, s).ListAgentConversations(&types.AgentPageRequest{PageNo: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, conversations, `{"items":[],"total":0}`)
	messages, err := agentlogic.NewListAgentMessagesLogic(ctx, s).ListAgentMessages(&types.AgentMessageListRequest{ID: "conversation-id", PageNo: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	assertJSON(t, messages, `{"items":[],"total":0}`)
}

func TestAgentUnavailableServiceProducesSafeErrorsForEveryEndpoint(t *testing.T) {
	for _, client := range []*agentContractClient{nil, {failure: status.Error(codes.Unavailable, "private-service-address secret-value")}, {failure: status.Error(codes.DeadlineExceeded, "private-service-address secret-value")}} {
		s := &svc.ServiceContext{}
		if client != nil {
			s.AIAgentRPC = client
		}
		for _, handler := range []http.HandlerFunc{agenthandler.GetAgentInfoHandler(s), agenthandler.CreateAgentRunHandler(s), agenthandler.GetAgentRunHandler(s), agenthandler.CancelAgentRunHandler(s), agenthandler.ListAgentConversationsHandler(s), agenthandler.ListAgentMessagesHandler(s)} {
			r := httptest.NewRequest(http.MethodPost, "/v1/ai/contract", strings.NewReader(`{"requestId":"request-id","message":"查询审计"}`)).WithContext(agentJWTContext())
			r.Header.Set("Content-Type", "application/json")
			r = pathvar.WithVars(r, map[string]string{"id": "path-id"})
			w := httptest.NewRecorder()
			handler(w, r)
			var envelope struct {
				Code    uint32 `json:"code"`
				Message string `json:"message"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if w.Code != http.StatusOK || envelope.Code != xerr.SERVER_COMMON_ERROR || envelope.Message != "AI服务暂不可用，请稍后重试" || strings.Contains(w.Body.String(), "secret-value") {
				t.Fatalf("AI unavailable must remain safe and preserve HTTP contract: %s", w.Body.String())
			}
		}
	}
}
