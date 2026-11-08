package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
)

type recordedUsage struct {
	attempt UsageAttempt
	result  UsageObservation
}
type recordingUsageHook struct {
	mu     sync.Mutex
	rows   []recordedUsage
	done   chan struct{}
	before func(context.Context)
}

func (h *recordingUsageHook) Begin(ctx context.Context, attempt UsageAttempt) error {
	if h.before != nil {
		h.before(ctx)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rows = append(h.rows, recordedUsage{attempt: attempt})
	return nil
}

func TestUsageOnlyFrameBeforeBrokenJSONRetainsKnownPartial(t *testing.T) {
	hook := &recordingUsageHook{done: make(chan struct{}, 1)}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"部分答案\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\ndata: {broken-json\n\n")
		w.(http.Flusher).Flush()
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			_ = conn.Close()
		}
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, APIKey: "local-test-only", Provider: "qwen", Model: "local-test", Timeout: 2, UsageHook: hook})
	if err != nil {
		t.Fatal(err)
	}
	reader, err := client.ChatStream(usageTestContext(), []*schema.Message{schema.UserMessage("测试")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err = reader.Recv(); err != nil {
			break
		}
	}
	reader.Close()
	select {
	case <-hook.done:
	case <-time.After(2 * time.Second):
		t.Fatal("破损流未结算")
	}
	rows := hook.snapshot()
	if len(rows) != 1 || rows[0].result.TotalTokens == nil || *rows[0].result.TotalTokens != 15 || rows[0].result.UsageState != "partial" {
		t.Fatalf("SDK缓存usage-only帧不能丢计量：%+v", rows)
	}
}

func TestUsageProviderDeadlineFitsActiveLease(t *testing.T) {
	hook := &recordingUsageHook{before: func(ctx context.Context) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 90*time.Second {
			t.Error("启用计量的供应商请求必须在90秒内结束，防止120秒活跃租约提前过期")
		}
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": embedTestData(1)})
	}))
	defer server.Close()
	embedder, err := NewEmbedder(EmbedConfig{BaseURL: server.URL, APIKey: "local-test-only", Provider: "qwen", Model: "text-embedding-v4", Dims: 1, Timeout: 300, UsageHook: hook})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = embedder.Embed(usageTestContext(), []string{"测试"}); err != nil {
		t.Fatal(err)
	}
}
func (h *recordingUsageHook) Dispatched(context.Context, string) error { return nil }
func (h *recordingUsageHook) Finish(_ context.Context, id string, observation UsageObservation) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.rows {
		if h.rows[i].attempt.AttemptID == id {
			h.rows[i].result = observation
		}
	}
	if h.done != nil {
		select {
		case h.done <- struct{}{}:
		default:
		}
	}
	return nil
}
func (h *recordingUsageHook) snapshot() []recordedUsage {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]recordedUsage(nil), h.rows...)
}
func usageTestContext() context.Context {
	return WithUsageIdentity(context.Background(), UsageIdentity{UserID: 17, Source: "analysis", RequestID: "c163b723-345d-4841-9446-45ad90c296ea", SourceID: 31, StepIndex: 0})
}

func TestUsageKeepsTokensWhenSDKRejectsEmptyChoices(t *testing.T) {
	hook := &recordingUsageHook{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Request-Id", "<script>unsafe</script>")
		_, _ = io.WriteString(w, `{"choices":[],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, APIKey: "local-test-only", Provider: "qwen", Model: "local-test", Timeout: 2, UsageHook: hook})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = client.ChatBounded(usageTestContext(), []*schema.Message{schema.UserMessage("测试")}, 20); err == nil {
		t.Fatal("空choices必须仍被SDK拒绝")
	}
	rows := hook.snapshot()
	if len(rows) != 1 {
		t.Fatalf("真实provider调用账本=%d，期望1", len(rows))
	}
	got := rows[0].result
	if got.TotalTokens == nil || *got.TotalTokens != 15 || got.UsageState != "complete" || !got.DispatchConfirmed || got.Status != "failed" {
		t.Fatalf("SDK拒绝不能丢弃已知usage：%+v", got)
	}
	if got.ProviderRequestID != "" {
		t.Fatal("恶意provider request_id不能存储或回显")
	}
}

func TestUsageEmbeddingKeepsFirstBatchWhenSecondFails(t *testing.T) {
	hook := &recordingUsageHook{}
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if calls.Add(1) == 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(w, `{"error":{"message":"provider-private-error"}}`)
			return
		}
		var input embedRequest
		_ = json.NewDecoder(r.Body).Decode(&input)
		data := embedTestData(len(input.Input))
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data, "usage": map[string]any{"prompt_tokens": 7, "total_tokens": 7}})
	}))
	defer server.Close()
	embedder, err := NewEmbedder(EmbedConfig{BaseURL: server.URL, APIKey: "local-test-only", Provider: "qwen", Model: "text-embedding-v4", Dims: 1, Timeout: 2, UsageHook: hook})
	if err != nil {
		t.Fatal(err)
	}
	if vectors, err := embedder.Embed(usageTestContext(), make([]string, 11)); err == nil || vectors != nil {
		t.Fatal("第二批失败仍须整体失败")
	}
	rows := hook.snapshot()
	if len(rows) != 2 {
		t.Fatalf("Embedding真实批次数=%d，期望2", len(rows))
	}
	if rows[0].result.TotalTokens == nil || *rows[0].result.TotalTokens != 7 || rows[0].result.CompletionTokens != nil || rows[0].result.UsageState != "complete" {
		t.Fatalf("首批已知Embedding用量丢失：%+v", rows[0].result)
	}
	if rows[1].result.UsageState != "unknown" || rows[1].result.TotalTokens != nil || !rows[1].result.DispatchConfirmed || rows[1].result.Status != "failed" {
		t.Fatalf("503不能冒充零用量：%+v", rows[1].result)
	}
}

func TestUsageSDKPreflightFailureReleasesUnsentReservation(t *testing.T) {
	hook := &recordingUsageHook{}
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, APIKey: "local-test-only", Provider: "qwen", Model: "o1-local-test", Timeout: 2, UsageHook: hook})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = client.ChatBounded(usageTestContext(), []*schema.Message{schema.UserMessage("测试")}, 20); err == nil {
		t.Fatal("SDK必须拒绝推理模型不支持的temperature")
	}
	rows := hook.snapshot()
	if len(rows) != 1 || rows[0].result.Status != "not_sent" || rows[0].result.DispatchConfirmed || calls.Load() != 0 {
		t.Fatalf("未发HTTP的失败应释放预留：rows=%+v calls=%d", rows, calls.Load())
	}
}

func TestUsageRedirectDoesNotCancelLogicalCall(t *testing.T) {
	for _, status := range []int{http.StatusFound, http.StatusTemporaryRedirect} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("status_%d_hook_%t", status, enabled), func(t *testing.T) {
				var actual atomic.Int64
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/actual" {
						w.Header().Set("Location", "/actual")
						w.WriteHeader(status)
						return
					}
					actual.Add(1)
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"choices":[{"index":0,"message":{"role":"assistant","content":"完成"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":1,"total_tokens":3}}`)
				}))
				defer server.Close()
				hook := &recordingUsageHook{}
				var observer UsageHook
				if enabled {
					observer = hook
				}
				client, err := New(Config{BaseURL: server.URL, APIKey: "local-test-only", Provider: "qwen", Model: "local-test", Timeout: 2, UsageHook: observer})
				if err != nil {
					t.Fatal(err)
				}
				message, usage, err := client.ChatBounded(usageTestContext(), []*schema.Message{schema.UserMessage("测试")}, 20)
				if err != nil || message == nil || message.Content != "完成" || usage == nil || usage.TotalTokens != 3 || actual.Load() != 1 {
					t.Fatalf("合法重定向必须完成最终调用：message=%v usage=%v actual=%d err=%v", message, usage, actual.Load(), err)
				}
				if enabled {
					rows := hook.snapshot()
					if len(rows) != 2 || rows[0].attempt.AttemptID == rows[1].attempt.AttemptID || !rows[0].result.DispatchConfirmed || !rows[1].result.DispatchConfirmed || rows[1].result.TotalTokens == nil || *rows[1].result.TotalTokens != 3 {
						t.Fatalf("每跳必须独立结算且保留最终用量：%+v", rows)
					}
				}
			})
		}
	}
}

type usageRoundTripFunc func(*http.Request) (*http.Response, error)

func (f usageRoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestUsageCancellationBeforeDispatchRemainsUncertain(t *testing.T) {
	hook := &recordingUsageHook{}
	parent, cancel := context.WithCancel(usageTestContext())
	defer cancel()
	ctx, call, err := beginUsage(parent, hook, "test", "local-test", "chat", 0, 1, "local-test-only")
	if err != nil {
		t.Fatal(err)
	}
	transport := &usageTransport{base: usageRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		cancel()
		return nil, r.Context().Err()
	})}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://unused.test/chat/completions", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, tripErr := transport.RoundTrip(req)
	if tripErr == nil {
		t.Fatal("取消桩必须返回取消错误")
	}
	if err := call.finish(tripErr, false); err != nil {
		t.Fatal(err)
	}
	rows := hook.snapshot()
	if len(rows) != 1 || rows[0].result.Status != "cancelled" || rows[0].result.DispatchConfirmed || rows[0].result.UsageState != "unknown" || rows[0].result.TotalTokens != nil {
		t.Fatalf("RoundTrip已开始但未确认dispatch不能退款或补零：%+v", rows)
	}
}

func TestUsageEmbeddingRedirectKeepsFinalBatchUsage(t *testing.T) {
	hook := &recordingUsageHook{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/actual" {
			w.Header().Set("Location", "/actual")
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": embedTestData(1), "usage": map[string]any{"prompt_tokens": 7, "total_tokens": 7}})
	}))
	defer server.Close()
	embedder, err := NewEmbedder(EmbedConfig{BaseURL: server.URL, APIKey: "local-test-only", Provider: "test", Model: "text-embedding-v4", Dims: 1, Timeout: 2, UsageHook: hook})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := embedder.Embed(usageTestContext(), []string{"测试"}); err != nil {
		t.Fatal(err)
	}
	rows := hook.snapshot()
	if len(rows) != 2 || rows[1].result.TotalTokens == nil || *rows[1].result.TotalTokens != 7 || rows[1].result.UsageState != "complete" || rows[1].result.Status != "succeeded" {
		t.Fatalf("Embedding重定向最终批次须在类型解析后结算已知用量：%+v", rows)
	}
}

func TestUsageStreamRedirectKeepsCaptureAndReleasesContext(t *testing.T) {
	hook := &recordingUsageHook{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/actual" {
			w.Header().Set("Location", "/actual")
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"完成\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":1,\"total_tokens\":3}}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, APIKey: "local-test-only", Provider: "test", Model: "local-test", Timeout: 2, UsageHook: hook})
	if err != nil {
		t.Fatal(err)
	}
	reader, captured, err := client.ChatStreamWithUsage(usageTestContext(), []*schema.Message{schema.UserMessage("测试")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for {
		if _, err := reader.Recv(); err != nil {
			if err != io.EOF {
				t.Fatal(err)
			}
			break
		}
	}
	usage, complete := captured.Snapshot()
	if usage == nil || usage.TotalTokens != 3 || complete {
		t.Fatalf("流式最终已知值只计一次；首跳缺失用量仍不冒充全部已知：usage=%v complete=%t", usage, complete)
	}
	rows := hook.snapshot()
	if len(rows) != 2 || rows[1].result.Status != "succeeded" || rows[1].result.UsageState != "complete" || rows[1].result.TotalTokens == nil || *rows[1].result.TotalTokens != 3 {
		t.Fatalf("SSE重定向必须独立结算最终attempt：%+v", rows)
	}
}

func TestUsageRedirectStillHonorsCallerCancellation(t *testing.T) {
	hook := &recordingUsageHook{}
	arrived, released, stop := make(chan struct{}), make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/actual" {
			w.Header().Set("Location", "/actual")
			w.WriteHeader(http.StatusTemporaryRedirect)
			return
		}
		// 读完POST体后net/http才启动探测对端断开的后台读取。
		_, _ = io.Copy(io.Discard, r.Body)
		close(arrived)
		select {
		case <-r.Context().Done():
			close(released)
		case <-stop:
		}
	}))
	defer server.Close()
	defer close(stop)
	client, err := New(Config{BaseURL: server.URL, APIKey: "local-test-only", Provider: "test", Model: "local-test", Timeout: 2, UsageHook: hook})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(usageTestContext())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, _, err := client.ChatBounded(ctx, []*schema.Message{schema.UserMessage("测试")}, 20)
		done <- err
	}()
	select {
	case <-arrived:
	case <-time.After(2 * time.Second):
		t.Fatal("未到达重定向最终端点")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("取消后不能返回成功")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("重定向链未响应调用者取消")
	}
	select {
	case <-released:
	case <-time.After(2 * time.Second):
		t.Fatal("最终HTTP请求取消后资源未释放")
	}
	rows := hook.snapshot()
	if len(rows) != 2 || rows[1].result.Status != "cancelled" || rows[1].result.UsageState != "unknown" || rows[1].result.TotalTokens != nil {
		t.Fatalf("取消最终跳必须保守结算unknown：%+v", rows)
	}
}

func TestUsageMissingIdentityDoesNotCallProvider(t *testing.T) {
	hook := &recordingUsageHook{}
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`)
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, APIKey: "local-test-only", Provider: "qwen", Model: "local-test", UsageHook: hook})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = client.Chat(context.Background(), []*schema.Message{schema.UserMessage("测试")}, nil); err == nil || calls.Load() != 0 || len(hook.snapshot()) != 0 {
		t.Fatalf("缺身份的付费调用必须在HTTP前拒绝：err=%v calls=%d", err, calls.Load())
	}
}

func TestUsageStreamCancellationKeepsUnknownUsage(t *testing.T) {
	hook := &recordingUsageHook{done: make(chan struct{}, 1)}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"开始\"}}]}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()
	client, err := New(Config{BaseURL: server.URL, APIKey: "local-test-only", Provider: "qwen", Model: "local-test", Timeout: 2, UsageHook: hook})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(usageTestContext())
	reader, err := client.ChatStream(ctx, []*schema.Message{schema.UserMessage("测试")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reader.Recv(); err != nil {
		t.Fatal(err)
	}
	cancel()
	reader.Close()
	select {
	case <-hook.done:
	case <-time.After(2 * time.Second):
		t.Fatal("取消后没有结算unknown账本或释放活跃租约")
	}
	rows := hook.snapshot()
	if len(rows) != 1 || rows[0].result.UsageState != "unknown" || rows[0].result.TotalTokens != nil || !rows[0].result.DispatchConfirmed {
		t.Fatalf("流取消不能按零用量记录：%+v", rows)
	}
}

func TestUsageStreamTerminalMetadataAndInterruptedStream(t *testing.T) {
	for _, complete := range []bool{true, false} {
		t.Run(map[bool]string{true: "终帧用量", false: "缺终帧断流"}[complete], func(t *testing.T) {
			hook := &recordingUsageHook{done: make(chan struct{}, 1)}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"完成\"}}]}\n\n")
				if complete {
					_, _ = io.WriteString(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\ndata: [DONE]\n\n")
					w.(http.Flusher).Flush()
					<-r.Context().Done()
					return
				}
				w.(http.Flusher).Flush()
				conn, _, err := w.(http.Hijacker).Hijack()
				if err == nil {
					_ = conn.Close()
				}
			}))
			defer server.Close()
			client, err := New(Config{BaseURL: server.URL, APIKey: "local-test-only", Provider: "qwen", Model: "local-test", Timeout: 2, UsageHook: hook})
			if err != nil {
				t.Fatal(err)
			}
			reader, err := client.ChatStream(usageTestContext(), []*schema.Message{schema.UserMessage("测试")}, nil)
			if err != nil {
				t.Fatal(err)
			}
			for {
				if _, err = reader.Recv(); err != nil {
					break
				}
			}
			reader.Close()
			select {
			case <-hook.done:
			case <-time.After(2 * time.Second):
				t.Fatal("流终止没有独立结算账本")
			}
			rows := hook.snapshot()
			if len(rows) != 1 {
				t.Fatalf("调用数=%d", len(rows))
			}
			if complete {
				if rows[0].result.UsageState != "complete" || rows[0].result.TotalTokens == nil || *rows[0].result.TotalTokens != 15 {
					t.Fatalf("完整终帧不可记为unknown：%+v", rows[0].result)
				}
			} else if rows[0].result.UsageState != "unknown" || rows[0].result.TotalTokens != nil {
				t.Fatalf("断流不可记为零：%+v", rows[0].result)
			}
		})
	}
}
