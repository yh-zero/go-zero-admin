package handler

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	resthandler "github.com/zeromicro/go-zero/rest/handler"
	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/client/auditservice"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/client/aiservice"
	"go-zero-admin/pkg/xerr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const aiStreamPath = "/v1/sys/ai/messages/stream"
const aiStreamRequest = `{"request_id":"2a80bedc-1e29-4f39-a420-986ec11fd35b","conversation_id":9,"content":"问题"}`

func TestRegisteredAIRoutesSSEIsolation(t *testing.T) {
	installRouteResponseHandlers(t)
	routes := registeredAIRoutes(t, &svc.ServiceContext{Validator: svc.NewValidator()})
	streamRoutes := 0
	for _, route := range routes {
		t.Run(route.Method+" "+route.Path, func(t *testing.T) {
			request := httptest.NewRequest(route.Method, route.Path, strings.NewReader("{"))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Accept", "text/event-stream")
			recorder := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
			route.Handler(recorder, request)

			wantSSE := route.Method == http.MethodPost && route.Path == aiStreamPath
			if wantSSE {
				streamRoutes++
			}
			if got := len(recorder.deadlines); got != boolInt(wantSSE) {
				t.Fatalf("清除写入期限次数 = %d，期望 %d（仅发送路由启用框架 SSE）", got, boolInt(wantSSE))
			}
			for _, deadline := range recorder.deadlines {
				if !deadline.IsZero() {
					t.Fatalf("SSE 写入期限必须清除，实际为 %v", deadline)
				}
			}
			if contentType := recorder.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
				t.Fatalf("开流前错误必须返回 JSON，实际 Content-Type = %q", contentType)
			}
		})
	}
	if streamRoutes != 1 {
		t.Fatalf("发送路由数量 = %d，期望 1", streamRoutes)
	}
}

func TestRegisteredAIStreamPreOpenErrorsRemainJSON(t *testing.T) {
	installRouteResponseHandlers(t)
	for _, test := range []struct {
		name      string
		body      string
		wantCode  uint32
		wantCalls int
	}{
		{name: "解析失败", body: "{", wantCode: xerr.REQUEST_PARAM_ERROR},
		{name: "校验失败", body: `{"request_id":"invalid","conversation_id":9,"content":"问题"}`, wantCode: xerr.REQUEST_PARAM_ERROR},
		{name: "建立RPC失败", body: aiStreamRequest, wantCode: xerr.SERVICE_UNAVAILABLE, wantCalls: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			client := &routeAIClient{send: func(context.Context, *aipb.SendMessageRequest) (aipb.AiService_SendMessageStreamClient, error) {
				calls++
				return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
			}}
			route := registeredAIRoute(t, client, http.MethodPost, aiStreamPath)
			request := streamRequest(context.Background())
			request.Body = io.NopCloser(strings.NewReader(test.body))
			request.ContentLength = int64(len(test.body))
			recorder := httptest.NewRecorder()
			route.Handler(recorder, withRouteIdentity(request))
			assertJSONEnvelope(t, recorder, test.wantCode)
			if recorder.Flushed || strings.Contains(recorder.Body.String(), "event:") {
				t.Fatal("开流前错误不能写入或 Flush SSE 事件")
			}
			if calls != test.wantCalls {
				t.Fatalf("RPC 调用次数 = %d，期望 %d", calls, test.wantCalls)
			}
		})
	}
}

func TestRegisteredAIJSONSuccessRemainsJSON(t *testing.T) {
	installRouteResponseHandlers(t)
	route := registeredAIRoute(t, &routeAIClient{}, http.MethodGet, "/v1/sys/ai/tools")
	request := withRouteIdentity(httptest.NewRequest(http.MethodGet, route.Path, nil))
	request.Header.Set("Accept", "text/event-stream")
	recorder := &deadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	route.Handler(recorder, request)
	assertJSONEnvelope(t, recorder.ResponseRecorder, xerr.SUCCESS)
	if len(recorder.deadlines) != 0 || recorder.Flushed || recorder.Header().Get("Cache-Control") == "no-cache" {
		t.Fatal("普通 AI JSON 接口不应带有 SSE 写入期限、Flush 或缓存设置")
	}
	if !strings.Contains(recorder.Body.String(), `"name":"local_tool"`) {
		t.Fatalf("普通 AI 返回内容丢失：%s", recorder.Body.String())
	}
}

func TestRegisteredAIStreamHTTPEventsAndFlush(t *testing.T) {
	installRouteResponseHandlers(t)
	release := make(chan struct{}, 1)
	defer close(release)
	requests := make(chan *aipb.SendMessageRequest, 1)
	client := &routeAIClient{send: func(ctx context.Context, request *aipb.SendMessageRequest) (aipb.AiService_SendMessageStreamClient, error) {
		requests <- request
		next := 0
		return &routeAIStream{ctx: ctx, recv: func() (*aipb.StreamChunk, error) {
			next++
			switch next {
			case 1:
				return &aipb.StreamChunk{Type: "meta", ConversationId: 9, UserMessageId: 1, AiMessageId: 2}, nil
			case 2:
				select {
				case <-release:
					return &aipb.StreamChunk{Type: "delta", Content: "结果"}, nil
				case <-ctx.Done():
					return nil, ctx.Err()
				}
			case 3:
				return &aipb.StreamChunk{Type: "done", Tokens: 12, UsageComplete: true, Result: &aipb.SendMessageResponse{ConversationId: 9, AiMessage: &aipb.ChatMessage{Id: 2, Content: "结果", Status: "succeeded"}}}, nil
			default:
				return nil, io.EOF
			}
		}}, nil
	}}
	server, audits := newAIStreamHTTPServer(t, client)
	response := doAIStreamRequest(t, server, context.Background())
	defer response.Body.Close()
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/event-stream") {
		t.Fatalf("流响应 Content-Type = %q", contentType)
	}
	if response.Header.Get("Cache-Control") != "no-cache" || response.Header.Get("X-Accel-Buffering") != "no" {
		t.Fatalf("SSE 缓存或代理缓冲设置缺失：%v", response.Header)
	}
	reader := bufio.NewReader(response.Body)
	meta := readRouteSSEEvent(t, reader, "meta")
	if meta["conversation_id"] != float64(9) || meta["ai_message_id"] != float64(2) {
		t.Fatalf("meta 内容错误：%v", meta)
	}
	// 后续 RPC 帧仍在等待时，客户端已收到 meta，证明审计包装未吞掉 Flush。
	release <- struct{}{}
	if delta := readRouteSSEEvent(t, reader, "delta"); delta["content"] != "结果" {
		t.Fatalf("delta 内容错误：%v", delta)
	}
	done := readRouteSSEEvent(t, reader, "done")
	result, _ := done["result"].(map[string]any)
	message, _ := result["ai_message"].(map[string]any)
	if done["tokens"] != float64(12) || done["usage_complete"] != true || message["status"] != "succeeded" {
		t.Fatalf("done 未保留用量和权威消息结果：%v", done)
	}
	if tail, err := io.ReadAll(reader); err != nil || len(tail) != 0 {
		t.Fatalf("done 后应关闭流，额外内容=%q，错误=%v", tail, err)
	}
	request := <-requests
	if request.Actor == nil || request.Actor.UserId != 7 || request.ConversationId != 9 || request.Content != "问题" || request.RequestId != "2a80bedc-1e29-4f39-a420-986ec11fd35b" {
		t.Fatalf("API 未透传身份与发送参数：%v", request)
	}
	waitRouteSignal(t, audits, "审计完成")
}

func TestRegisteredAIStreamHTTPRPCInterruption(t *testing.T) {
	installRouteResponseHandlers(t)
	client := &routeAIClient{send: func(ctx context.Context, _ *aipb.SendMessageRequest) (aipb.AiService_SendMessageStreamClient, error) {
		return &routeAIStream{ctx: ctx, recv: func() (*aipb.StreamChunk, error) {
			return nil, status.Error(codes.Unavailable, "private upstream detail")
		}}, nil
	}}
	server, audits := newAIStreamHTTPServer(t, client)
	response := doAIStreamRequest(t, server, context.Background())
	defer response.Body.Close()
	reader := bufio.NewReader(response.Body)
	event := readRouteSSEEvent(t, reader, "error")
	message, _ := event["msg"].(string)
	if event["code"] != float64(500) || message == "" || strings.Contains(message, "private upstream detail") {
		t.Fatalf("RPC 中断必须返回通用 SSE 错误：%v", event)
	}
	if tail, err := io.ReadAll(reader); err != nil || len(tail) != 0 {
		t.Fatalf("error 后应关闭流，额外内容=%q，错误=%v", tail, err)
	}
	waitRouteSignal(t, audits, "审计完成")
}

func TestRegisteredAIStreamHTTPCancellationReachesRPC(t *testing.T) {
	installRouteResponseHandlers(t)
	recvStarted := make(chan struct{})
	recvCanceled := make(chan struct{})
	client := &routeAIClient{send: func(ctx context.Context, _ *aipb.SendMessageRequest) (aipb.AiService_SendMessageStreamClient, error) {
		first := true
		return &routeAIStream{ctx: ctx, recv: func() (*aipb.StreamChunk, error) {
			if first {
				first = false
				return &aipb.StreamChunk{Type: "meta", ConversationId: 9}, nil
			}
			close(recvStarted)
			<-ctx.Done()
			close(recvCanceled)
			return nil, ctx.Err()
		}}, nil
	}}
	server, audits := newAIStreamHTTPServer(t, client)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	response := doAIStreamRequest(t, server, ctx)
	defer response.Body.Close()
	readRouteSSEEvent(t, bufio.NewReader(response.Body), "meta")
	waitRouteSignal(t, recvStarted, "RPC Recv 启动")
	cancel()
	waitRouteSignal(t, recvCanceled, "请求取消传到 RPC Recv")
	waitRouteSignal(t, audits, "取消后 handler 与审计退出")
}

type deadlineRecorder struct {
	*httptest.ResponseRecorder
	deadlines []time.Time
}

func (w *deadlineRecorder) SetWriteDeadline(deadline time.Time) error {
	w.deadlines = append(w.deadlines, deadline)
	return nil
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func registeredAIRoutes(t *testing.T, svcCtx *svc.ServiceContext) []rest.Route {
	t.Helper()
	var config rest.RestConf
	if err := conf.FillDefault(&config); err != nil {
		t.Fatal(err)
	}
	config.Name = "ai-sse-route-test"
	config.Mode = "test"
	config.DevServer.Enabled = false
	server, err := rest.NewServer(config)
	if err != nil {
		t.Fatal(err)
	}
	RegisterHandlers(server, svcCtx)
	var routes []rest.Route
	for _, route := range server.Routes() {
		if strings.HasPrefix(route.Path, "/v1/sys/ai/") {
			routes = append(routes, route)
		}
	}
	if len(routes) == 0 {
		t.Fatal("未注册 AI 路由")
	}
	return routes
}

func registeredAIRoute(t *testing.T, client aiservice.AiService, method, path string) rest.Route {
	t.Helper()
	for _, route := range registeredAIRoutes(t, &svc.ServiceContext{AiRpc: client, Validator: svc.NewValidator()}) {
		if route.Method == method && route.Path == path {
			return route
		}
	}
	t.Fatalf("未注册路由 %s %s", method, path)
	return rest.Route{}
}

func assertJSONEnvelope(t *testing.T, recorder *httptest.ResponseRecorder, wantCode uint32) {
	t.Helper()
	wantStatus := http.StatusOK
	if wantCode == xerr.SERVICE_UNAVAILABLE {
		wantStatus = http.StatusServiceUnavailable
	}
	if recorder.Code != wantStatus || !strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("JSON 信封状态或类型错误：status=%d, headers=%v", recorder.Code, recorder.Header())
	}
	var body struct {
		Code uint32 `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || body.Code != wantCode || body.Msg == "" {
		t.Fatalf("JSON 信封错误：body=%s，err=%v", recorder.Body.String(), err)
	}
}

type routeAIClient struct {
	aiservice.AiService
	send func(context.Context, *aipb.SendMessageRequest) (aipb.AiService_SendMessageStreamClient, error)
}

func (c *routeAIClient) SendMessageStream(ctx context.Context, request *aipb.SendMessageRequest, _ ...grpc.CallOption) (aipb.AiService_SendMessageStreamClient, error) {
	return c.send(ctx, request)
}

func (c *routeAIClient) ListTools(context.Context, *aipb.ListToolsRequest, ...grpc.CallOption) (*aipb.ListToolsResponse, error) {
	return &aipb.ListToolsResponse{List: []*aipb.ToolInfo{{Name: "local_tool"}}}, nil
}

type routeAIStream struct {
	grpc.ClientStream
	ctx  context.Context
	recv func() (*aipb.StreamChunk, error)
}

func (s *routeAIStream) Context() context.Context         { return s.ctx }
func (s *routeAIStream) Recv() (*aipb.StreamChunk, error) { return s.recv() }

type routeAuditClient struct {
	auditservice.AuditService
	completed chan struct{}
}

func (c *routeAuditClient) CreateAuditLog(context.Context, *adminpb.CreateAuditLogRequest, ...grpc.CallOption) (*adminpb.NoDataResponse, error) {
	c.completed <- struct{}{}
	return &adminpb.NoDataResponse{}, nil
}

func newAIStreamHTTPServer(t *testing.T, client aiservice.AiService) (*httptest.Server, <-chan struct{}) {
	t.Helper()
	route := registeredAIRoute(t, client, http.MethodPost, aiStreamPath)
	audits := make(chan struct{}, 1)
	identity := func(w http.ResponseWriter, request *http.Request) {
		request = withRouteIdentity(request)
		middleware.MarkAuditIdentity(request.Context(), 7, "sse_test", 0)
		route.Handler(w, request)
	}
	audited := middleware.RequestMeta(middleware.Audit(&routeAuditClient{completed: audits})(identity))
	server := httptest.NewServer(resthandler.TraceHandler("ai-sse-route-test", aiStreamPath)(audited))
	t.Logf("HTTP 桩监听 %s，PID=%d，测试结束自动关闭", server.URL, os.Getpid())
	t.Cleanup(func() {
		server.Close()
		t.Logf("HTTP 桩已关闭 %s，PID=%d", server.URL, os.Getpid())
	})
	return server, audits
}

func withRouteIdentity(request *http.Request) *http.Request {
	return request.WithContext(context.WithValue(request.Context(), middleware.CtxKeyUserID, int64(7)))
}

func streamRequest(ctx context.Context) *http.Request {
	request := httptest.NewRequest(http.MethodPost, aiStreamPath, strings.NewReader(aiStreamRequest)).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	return request
}

func doAIStreamRequest(t *testing.T, server *httptest.Server, ctx context.Context) *http.Response {
	t.Helper()
	request := streamRequest(ctx)
	request.RequestURI = ""
	request.URL.Scheme = "http"
	request.URL.Host = strings.TrimPrefix(server.URL, "http://")
	client := server.Client()
	client.Timeout = 3 * time.Second
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func readRouteSSEEvent(t *testing.T, reader *bufio.Reader, wantEvent string) map[string]any {
	t.Helper()
	event, err := reader.ReadString('\n')
	if err != nil || event != "event: "+wantEvent+"\n" {
		t.Fatalf("SSE 事件 = %q，期望 %s，错误=%v", event, wantEvent, err)
	}
	data, err := reader.ReadString('\n')
	if err != nil || !strings.HasPrefix(data, "data: ") {
		t.Fatalf("SSE 数据 = %q，错误=%v", data, err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(strings.TrimPrefix(data, "data: ")), &payload); err != nil {
		t.Fatal(err)
	}
	if separator, err := reader.ReadString('\n'); err != nil || separator != "\n" {
		t.Fatalf("SSE 帧分隔符 = %q，错误=%v", separator, err)
	}
	return payload
}

func waitRouteSignal(t *testing.T, signal <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(3 * time.Second):
		t.Fatalf("等待%s超时", name)
	}
}

func installRouteResponseHandlers(t *testing.T) {
	t.Helper()
	httpx.SetErrorHandlerCtx(xerr.Handler)
	httpx.SetOkHandler(xerr.Success)
	t.Cleanup(func() {
		httpx.SetErrorHandlerCtx(nil)
		httpx.SetOkHandler(nil)
	})
}
