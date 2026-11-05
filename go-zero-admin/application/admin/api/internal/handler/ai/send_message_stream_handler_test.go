package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	zerohandler "github.com/zeromicro/go-zero/rest/handler"
	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/client/aiservice"
	"go-zero-admin/pkg/xerr"
	"google.golang.org/grpc"
)

func TestWriteSSEEventFlushesImmediately(t *testing.T) {
	recorder := httptest.NewRecorder()
	if err := writeSSEEvent(recorder, "delta", map[string]any{"content": "内容\n\"引用\""}); err != nil {
		t.Fatal(err)
	}
	if !recorder.Flushed {
		t.Fatal("事件写入后必须立即 Flush")
	}
	if want := "event: delta\ndata: {\"content\":\"内容\\n\\\"引用\\\"\"}\n\n"; recorder.Body.String() != want {
		t.Fatalf("事件编码 = %q，预期 %q", recorder.Body.String(), want)
	}
}

func TestWriteSSEEventEncodingFailureDoesNotWrite(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := writeSSEEvent(recorder, "delta", map[string]any{"content": make(chan string)})
	var unsupported *json.UnsupportedTypeError
	if !errors.As(err, &unsupported) {
		t.Fatalf("编码错误 = %v，预期不支持的 JSON 类型", err)
	}
	if recorder.Body.Len() != 0 || recorder.Flushed {
		t.Fatal("编码失败不能写入或 Flush 不完整事件")
	}
}

func TestWriteSSEFrameWriteFailuresDoNotFlush(t *testing.T) {
	writeErr := errors.New("写入失败")
	for _, test := range []struct {
		name  string
		short bool
		err   error
		want  error
	}{
		{name: "write_error", err: writeErr, want: writeErr},
		{name: "short_write", short: true, want: io.ErrShortWrite},
		{name: "short_write_with_error", short: true, err: writeErr, want: writeErr},
	} {
		t.Run(test.name, func(t *testing.T) {
			writer := &sseWriteFailureWriter{ResponseWriter: httptest.NewRecorder(), short: test.short, err: test.err}
			if err := writeSSEFrame(writer, ": ping\n\n"); !errors.Is(err, test.want) {
				t.Fatalf("写帧错误 = %v，预期 %v", err, test.want)
			}
			if writer.flushCalls != 0 {
				t.Fatal("写入失败不能继续 Flush")
			}
		})
	}
}

func TestWriteSSEEventReturnsFlushError(t *testing.T) {
	flushErr := errors.New("Flush 失败")
	recorder := httptest.NewRecorder()
	writer := &sseFlushErrorWriter{ResponseWriter: recorder, err: flushErr}
	if err := writeSSEEvent(writer, "delta", map[string]any{"content": "内容"}); !errors.Is(err, flushErr) {
		t.Fatalf("事件错误 = %v，预期 %v", err, flushErr)
	}
	if want := "event: delta\ndata: {\"content\":\"内容\"}\n\n"; recorder.Body.String() != want {
		t.Fatalf("Flush 前事件内容 = %q，预期 %q", recorder.Body.String(), want)
	}
}

func TestWriteSSEEventReturnsFlushErrorThroughTrace(t *testing.T) {
	flushErr := errors.New("底层 Flush 失败")
	recorder := httptest.NewRecorder()
	writer := &sseFlushErrorWriter{ResponseWriter: recorder, err: flushErr}
	var eventErr error
	var written int
	handler := zerohandler.TraceHandler("sse-test", "/stream")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		countingWriter := &sseCountingWriter{ResponseWriter: w}
		eventErr = writeSSEEvent(countingWriter, "delta", map[string]any{"content": "内容"})
		written = countingWriter.written
	}))
	handler.ServeHTTP(writer, httptest.NewRequest(http.MethodPost, "/stream", nil))
	if !errors.Is(eventErr, flushErr) {
		t.Fatalf("经 TraceHandler 包装后的事件错误 = %v，预期 %v", eventErr, flushErr)
	}
	if written != recorder.Body.Len() || written == 0 {
		t.Fatal("写入必须仍经过外层 writer，保留响应统计")
	}
}

func TestSendMessageStreamStopsOnInitialFlushErrorThroughTrace(t *testing.T) {
	stream := &sseTestStream{}
	client := &sseTestAIClient{stream: stream}
	svcCtx := &svc.ServiceContext{AiRpc: client, Validator: svc.NewValidator()}
	writer := &sseFlushErrorWriter{ResponseWriter: httptest.NewRecorder(), err: errors.New("开流 Flush 失败")}
	handler := zerohandler.TraceHandler("sse-test", "/stream")(SendMessageStreamHandler(svcCtx))
	handler.ServeHTTP(writer, newSSETestRequest(sseTestRequestBody))
	if stream.recvCalls.Load() != 0 {
		t.Fatal("初次 Flush 失败后必须立即退出，不能启动 RPC 接收")
	}
}

func TestSendMessageStreamCommitsStatusBeforeFirstFlush(t *testing.T) {
	client := &sseTestAIClient{stream: &sseTestStream{}}
	svcCtx := &svc.ServiceContext{AiRpc: client, Validator: svc.NewValidator()}
	recorder := httptest.NewRecorder()
	writer := &sseStatusTrackingWriter{ResponseWriter: recorder}
	SendMessageStreamHandler(svcCtx)(writer, newSSETestRequest(sseTestRequestBody))
	if writer.firstFlushStatus != http.StatusOK {
		t.Fatalf("首次 Flush 前显式状态 = %d，预期 200", writer.firstFlushStatus)
	}
	if len(writer.statusCodes) != 1 || writer.statusCodes[0] != http.StatusOK {
		t.Fatalf("开流应只显式提交一次 200，实际状态 = %v", writer.statusCodes)
	}
}

func TestSendMessageStreamPreOpenErrorsRemainJSON(t *testing.T) {
	httpx.SetErrorHandlerCtx(xerr.Handler)
	t.Cleanup(func() { httpx.SetErrorHandlerCtx(nil) })
	for _, test := range []struct {
		name string
		body string
		err  error
		code uint32
	}{
		{name: "parse", body: "{", code: xerr.REQUEST_PARAM_ERROR},
		{name: "validation", body: `{"request_id":"invalid","conversation_id":0,"content":"问题"}`, code: xerr.REQUEST_PARAM_ERROR},
		{name: "rpc", body: sseTestRequestBody, err: xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE), code: xerr.SERVICE_UNAVAILABLE},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &sseTestAIClient{err: test.err}
			svcCtx := &svc.ServiceContext{AiRpc: client, Validator: svc.NewValidator()}
			recorder := httptest.NewRecorder()
			writer := &sseStatusTrackingWriter{ResponseWriter: recorder}
			SendMessageStreamHandler(svcCtx)(writer, newSSETestRequest(test.body))
			var payload struct {
				Code uint32 `json:"code"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil || payload.Code != test.code {
				t.Fatalf("开流前错误应保持 JSON 信封：code = %d，错误 = %v", payload.Code, err)
			}
			if !strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") || recorder.Header().Get("X-Accel-Buffering") != "" || recorder.Flushed {
				t.Fatal("开流前错误不能提前提交或 Flush SSE 响应")
			}
			if len(writer.statusCodes) != 1 {
				t.Fatalf("开流前错误应仅由 JSON 出口提交状态，实际状态 = %v", writer.statusCodes)
			}
		})
	}
}

func TestWriteSSEFrameFlushesThroughWrappedWriter(t *testing.T) {
	recorder := httptest.NewRecorder()
	writer := &sseWrappedWriter{ResponseWriter: &sseWrappedWriter{ResponseWriter: recorder}}
	if err := writeSSEFrame(writer, ": ping\n\n"); err != nil {
		t.Fatal(err)
	}
	if recorder.Body.String() != ": ping\n\n" || !recorder.Flushed {
		t.Fatal("心跳必须经 writer 包装链写入并立即 Flush")
	}
}

func TestWriteStreamChunkMapsEvents(t *testing.T) {
	for _, test := range []struct {
		name     string
		chunk    *aipb.StreamChunk
		terminal bool
		frame    string
	}{
		{name: "meta", chunk: &aipb.StreamChunk{Type: "meta", ConversationId: 1, UserMessageId: 2, AiMessageId: 3}, frame: "event: meta\ndata: {\"ai_message_id\":3,\"conversation_id\":1,\"user_message_id\":2}\n\n"},
		{name: "delta", chunk: &aipb.StreamChunk{Type: "delta", Content: "内容"}, frame: "event: delta\ndata: {\"content\":\"内容\"}\n\n"},
		{name: "tool", chunk: &aipb.StreamChunk{Type: "tool", Content: "查询", ToolStatus: "done"}, frame: "event: tool\ndata: {\"name\":\"查询\",\"status\":\"done\"}\n\n"},
		{name: "done_without_result", chunk: &aipb.StreamChunk{Type: "done"}, terminal: true, frame: "event: done\ndata: {\"result\":null,\"tokens\":0,\"usage_complete\":false}\n\n"},
		{name: "error", chunk: &aipb.StreamChunk{Type: "error", ErrorCode: 503, Content: "暂不可用"}, terminal: true, frame: "event: error\ndata: {\"code\":503,\"msg\":\"暂不可用\"}\n\n"},
		{name: "unknown", chunk: &aipb.StreamChunk{Type: "unknown"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			terminal, err := writeStreamChunk(recorder, test.chunk)
			if err != nil || terminal != test.terminal {
				t.Fatalf("终帧 = %v，错误 = %v；预期终帧 %v", terminal, err, test.terminal)
			}
			if recorder.Body.String() != test.frame {
				t.Fatalf("事件帧 = %q，预期 %q", recorder.Body.String(), test.frame)
			}
			if recorder.Flushed != (test.frame != "") {
				t.Fatal("仅已写入的事件应立即 Flush")
			}
		})
	}
}

func TestWriteStreamChunkPropagatesWriteError(t *testing.T) {
	writeErr := errors.New("写入失败")
	for _, event := range []string{"meta", "delta", "tool", "done", "error"} {
		t.Run(event, func(t *testing.T) {
			writer := &sseWriteFailureWriter{ResponseWriter: httptest.NewRecorder(), err: writeErr}
			terminal, err := writeStreamChunk(writer, &aipb.StreamChunk{Type: event})
			if !errors.Is(err, writeErr) || terminal != (event == "done" || event == "error") {
				t.Fatalf("终帧 = %v，错误 = %v；事件必须传递写入错误", terminal, err)
			}
		})
	}
}

func TestAIStreamDoneContainsCommittedMessages(t *testing.T) {
	result := &aipb.SendMessageResponse{ConversationId: 9, Conversation: &aipb.ConversationInfo{Id: 9, Title: "本地会话", MessageCount: 2}, UserMessage: &aipb.ChatMessage{Id: 1, Role: "user", Content: "问题", Status: "succeeded"}, AiMessage: &aipb.ChatMessage{Id: 2, Role: "assistant", Content: "结果", ToolCalls: "[]", Status: "succeeded"}}
	recorder := httptest.NewRecorder()
	terminal, err := writeStreamChunk(recorder, &aipb.StreamChunk{Type: "done", Tokens: 12, UsageComplete: true, Result: result})
	if err != nil || !terminal || !recorder.Flushed {
		t.Fatalf("done 必须 Flush 并结束流：终帧 = %v，错误 = %v", terminal, err)
	}
	data := strings.TrimSuffix(strings.TrimPrefix(recorder.Body.String(), "event: done\ndata: "), "\n\n")
	var payload struct {
		Tokens        int64 `json:"tokens"`
		UsageComplete bool  `json:"usage_complete"`
		Result        struct {
			ConversationId int64 `json:"conversation_id"`
			Conversation   struct {
				Id           int64  `json:"id"`
				Title        string `json:"title"`
				MessageCount int64  `json:"message_count"`
			} `json:"conversation"`
			UserMessage struct {
				Id      int64  `json:"id"`
				Content string `json:"content"`
				Status  string `json:"status"`
			} `json:"user_message"`
			AiMessage struct {
				Id        int64  `json:"id"`
				Content   string `json:"content"`
				ToolCalls string `json:"tool_calls"`
				Status    string `json:"status"`
			} `json:"ai_message"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Tokens != 12 || !payload.UsageComplete {
		t.Fatalf("done 用量未保留：%+v", payload)
	}
	if payload.Result.ConversationId != 9 || payload.Result.Conversation.Id != 9 || payload.Result.Conversation.Title != "本地会话" || payload.Result.Conversation.MessageCount != 2 {
		t.Fatalf("done 未返回当前会话摘要：%+v", payload.Result)
	}
	if payload.Result.UserMessage.Id != 1 || payload.Result.UserMessage.Content != "问题" || payload.Result.UserMessage.Status != "succeeded" {
		t.Fatalf("done 未返回已提交的用户消息：%+v", payload.Result.UserMessage)
	}
	if payload.Result.AiMessage.Id != 2 || payload.Result.AiMessage.Content != "结果" || payload.Result.AiMessage.ToolCalls != "[]" || payload.Result.AiMessage.Status != "succeeded" {
		t.Fatalf("done 未返回已提交的助手消息：%+v", payload.Result.AiMessage)
	}
}

type sseWriteFailureWriter struct {
	http.ResponseWriter
	short      bool
	err        error
	flushCalls int
}

func (w *sseWriteFailureWriter) Write(payload []byte) (int, error) {
	if w.short {
		n, _ := w.ResponseWriter.Write(payload[:len(payload)-1])
		return n, w.err
	}
	return 0, w.err
}

func (w *sseWriteFailureWriter) Flush() {
	w.flushCalls++
}

type sseFlushErrorWriter struct {
	http.ResponseWriter
	err error
}

func (w *sseFlushErrorWriter) FlushError() error {
	return w.err
}

func (w *sseFlushErrorWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type sseWrappedWriter struct {
	http.ResponseWriter
}

func (w *sseWrappedWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type sseCountingWriter struct {
	http.ResponseWriter
	written int
}

func (w *sseCountingWriter) Write(payload []byte) (int, error) {
	n, err := w.ResponseWriter.Write(payload)
	w.written += n
	return n, err
}

func (w *sseCountingWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

type sseStatusTrackingWriter struct {
	http.ResponseWriter
	statusCodes      []int
	firstFlushStatus int
}

func (w *sseStatusTrackingWriter) WriteHeader(code int) {
	w.statusCodes = append(w.statusCodes, code)
	w.ResponseWriter.WriteHeader(code)
}

func (w *sseStatusTrackingWriter) FlushError() error {
	if len(w.statusCodes) > 0 {
		w.firstFlushStatus = w.statusCodes[0]
	}
	return http.NewResponseController(w.ResponseWriter).Flush()
}

const sseTestRequestBody = `{"request_id":"00000000-0000-4000-8000-000000000001","conversation_id":0,"content":"问题"}`

func newSSETestRequest(body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/stream", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request.WithContext(context.WithValue(request.Context(), middleware.CtxKeyUserID, int64(1)))
}

type sseTestAIClient struct {
	aiservice.AiService
	stream aipb.AiService_SendMessageStreamClient
	err    error
}

func (c *sseTestAIClient) SendMessageStream(context.Context, *aipb.SendMessageRequest, ...grpc.CallOption) (aipb.AiService_SendMessageStreamClient, error) {
	return c.stream, c.err
}

type sseTestStream struct {
	grpc.ClientStream
	recvCalls atomic.Int64
}

func (s *sseTestStream) Recv() (*aipb.StreamChunk, error) {
	s.recvCalls.Add(1)
	return nil, io.EOF
}
