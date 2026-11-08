package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type lifecycleStreamModel struct {
	model.ChatModel
	ctx    context.Context
	err    error
	reader *schema.StreamReader[*schema.Message]
}

type blockingStreamUsageHook struct {
	UsageHook
	entered chan struct{}
	release chan struct{}
}

func (*blockingStreamUsageHook) Begin(context.Context, UsageAttempt) error { return nil }
func (h *blockingStreamUsageHook) Finish(context.Context, string, UsageObservation) error {
	close(h.entered)
	<-h.release
	return nil
}

func TestChatStreamCloseCancelsBeforeSlowAccountingButEOFKeepsSuccess(t *testing.T) {
	for _, terminal := range []string{"close", "eof"} {
		t.Run(terminal, func(t *testing.T) {
			hook := &blockingStreamUsageHook{entered: make(chan struct{}), release: make(chan struct{})}
			stub := &lifecycleStreamModel{reader: schema.StreamReaderFromArray([]*schema.Message{})}
			client := &Client{model: stub, timeout: time.Minute, hook: hook}
			stream, _, err := client.ChatStreamWithUsage(usageTestContext(), nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { close(hook.release) }) }
			defer release()
			go func() {
				if terminal == "close" {
					stream.Close()
					done <- nil
				} else {
					_, err := stream.Recv()
					done <- err
				}
			}()
			select {
			case <-hook.entered:
			case <-time.After(time.Second):
				t.Fatal("本地计量结算未开始")
			}
			if terminal == "close" && stub.ctx.Err() != context.Canceled {
				t.Error("Close必须先取消模型请求，不能等待计量存储写完")
			}
			if terminal == "eof" && stub.ctx.Err() != nil {
				t.Error("正常EOF在成功计量完成前不能提前取消请求")
			}
			release()
			select {
			case err := <-done:
				if terminal == "eof" && err != io.EOF {
					t.Fatal("正常流的EOF必须保持", err)
				}
			case <-time.After(time.Second):
				t.Fatal("计量完成后流终态没有返回")
			}
			if stub.ctx.Err() != context.Canceled {
				t.Fatal("终态结算完成后必须释放请求上下文")
			}
			stream.Close()
		})
	}
}

func (m *lifecycleStreamModel) Stream(ctx context.Context, _ []*schema.Message, _ ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	m.ctx = ctx
	return m.reader, m.err
}

func TestChatStreamSetupFailureReleasesContextAndUsage(t *testing.T) {
	for _, name := range []string{"sdk_error", "nil_reader", "reader_with_error"} {
		t.Run(name, func(t *testing.T) {
			stub := &lifecycleStreamModel{}
			var writer *schema.StreamWriter[*schema.Message]
			if name != "nil_reader" {
				stub.err = errors.New("本地流初始化失败")
			}
			if name == "reader_with_error" {
				stub.reader, writer = schema.Pipe[*schema.Message](0)
				defer writer.Close()
			}
			hook := &recordingUsageHook{}
			client := &Client{model: stub, timeout: time.Minute, hook: hook}
			reader, capture, err := client.ChatStreamWithUsage(usageTestContext(), nil, nil)
			if reader != nil || capture == nil || err == nil {
				t.Fatal("流初始化错误或空reader必须返回失败并保留安全用量capture")
			}
			select {
			case <-stub.ctx.Done():
			default:
				t.Fatal("初始化失败后请求上下文必须立即释放")
			}
			rows := hook.snapshot()
			if len(rows) != 1 || rows[0].result.Status != "not_sent" {
				t.Fatal("未得到reader的模型调用必须结束计量预留")
			}
			if writer != nil && !writer.Send(nil, nil) {
				t.Fatal("SDK同时返回reader和错误时也必须关闭原reader")
			}
		})
	}
}

func TestChatStreamWithoutHookKeepsCallerDeadlineAndClosesIdempotently(t *testing.T) {
	reader, writer := schema.Pipe[*schema.Message](0)
	defer writer.Close()
	stub := &lifecycleStreamModel{reader: reader}
	client := &Client{model: stub, timeout: time.Nanosecond}
	stream, _, err := client.ChatStreamWithUsage(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, bounded := stub.ctx.Deadline(); bounded || stub.ctx.Err() != nil {
		t.Fatal("无hook流继续只遵循调用者截止时间，不额外施加Client超时")
	}
	stream.Close()
	stream.Close()
	if stub.ctx.Err() != context.Canceled || !writer.Send(nil, nil) {
		t.Fatal("Close须取消模型请求并幂等关闭原reader")
	}
}

type rejectingStreamUsageHook struct {
	UsageHook
	ctx context.Context
}

func (h *rejectingStreamUsageHook) Begin(ctx context.Context, _ UsageAttempt) error {
	h.ctx = ctx
	return errors.New("本地拒绝预留")
}

func TestChatStreamRejectedUsageReleasesContextBeforeModel(t *testing.T) {
	stub, hook := &lifecycleStreamModel{}, &rejectingStreamUsageHook{}
	client := &Client{model: stub, timeout: time.Minute, hook: hook}
	reader, _, err := client.ChatStreamWithUsage(usageTestContext(), nil, nil)
	if err == nil || reader != nil || stub.ctx != nil || hook.ctx == nil {
		t.Fatal("预留拒绝必须在调用SDK前退出")
	}
	select {
	case <-hook.ctx.Done():
	default:
		t.Fatal("未创建usage call的拒绝路径也必须释放上下文")
	}
}

func TestChatStreamContextLivesUntilEOFOrCloseAcrossRedirect(t *testing.T) {
	for _, finish := range []string{"eof", "close"} {
		t.Run(finish, func(t *testing.T) {
			release := make(chan struct{})
			defer close(release)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/actual" {
					w.Header().Set("Location", "/actual")
					w.WriteHeader(http.StatusTemporaryRedirect)
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"部分回复\"}}],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":1,\"total_tokens\":3}}\n\n")
				w.(http.Flusher).Flush()
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":1,\"total_tokens\":3}}\n\ndata: [DONE]\n\n")
			}))
			defer server.Close()
			var contexts []context.Context
			hook := &recordingUsageHook{before: func(ctx context.Context) { contexts = append(contexts, ctx) }}
			client, err := New(Config{BaseURL: server.URL, APIKey: "local-test-only", Provider: "local", Model: "local", Timeout: 5, UsageHook: hook})
			if err != nil {
				t.Fatal(err)
			}
			reader, capture, err := client.ChatStreamWithUsage(usageTestContext(), []*schema.Message{schema.UserMessage("本地流生命周期")}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(contexts) != 2 || contexts[0].Err() != nil || contexts[1].Err() != nil {
				t.Fatal("返回正常reader和经过重定向后不能提前取消逻辑调用")
			}
			if _, err := reader.Recv(); err != nil {
				t.Fatal(err)
			}
			if finish == "eof" {
				release <- struct{}{}
				for {
					_, err := reader.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			reader.Close()
			reader.Close()
			usage, complete := capture.Snapshot()
			if usage == nil || usage.TotalTokens != 3 || finish == "close" && complete {
				t.Fatal("关闭或EOF结算须保留已知用量，提前关闭不能伪报完整")
			}
			rows := hook.snapshot()
			if finish == "eof" && (len(rows) != 2 || rows[1].result.Status != "succeeded") {
				t.Fatal("正常EOF后的Close不能把成功计量降为取消")
			}
			for _, ctx := range contexts {
				select {
				case <-ctx.Done():
				case <-time.After(time.Second):
					t.Fatal("EOF或Close后逻辑调用与重定向计时器必须释放")
				}
			}
		})
	}
}
