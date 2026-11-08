package aiservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/config"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/llm"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type aiTestStream struct {
	grpc.ServerStream
	ctx    context.Context
	chunks []*aipb.StreamChunk
	onSend func(*aipb.StreamChunk) error
}

func (s *aiTestStream) Context() context.Context { return s.ctx }
func (s *aiTestStream) Send(chunk *aipb.StreamChunk) error {
	if s.onSend != nil {
		if err := s.onSend(chunk); err != nil {
			return err
		}
	}
	s.chunks = append(s.chunks, proto.Clone(chunk).(*aipb.StreamChunk))
	return nil
}
func (s *aiTestStream) last() *aipb.StreamChunk {
	if len(s.chunks) == 0 {
		return &aipb.StreamChunk{}
	}
	return s.chunks[len(s.chunks)-1]
}

func aiIntegrationContext(t *testing.T, handler http.HandlerFunc) (*svc.ServiceContext, int64) {
	t.Helper()
	dsn, redisAddress := os.Getenv("GOZERO_AI_TEST_DSN"), os.Getenv("GOZERO_AI_TEST_REDIS")
	if dsn == "" || redisAddress == "" {
		t.Skip("未设置隔离 AI 数据库和 Redis，未执行集成验证")
	}
	sqlx.DisableLog()
	conn := sqlx.NewMysql(dsn)
	rdb, err := redis.NewRedis(redis.RedisConf{Host: redisAddress, Type: "node"})
	if err != nil {
		t.Fatal("隔离 Redis 初始化失败")
	}
	if !rdb.PingCtx(context.Background()) {
		t.Fatal("隔离 Redis 不可达")
	}
	cacheConfig := cache.CacheConf{{RedisConf: redis.RedisConf{Host: redisAddress, Type: "node"}, Weight: 100}}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := llm.New(llm.Config{BaseURL: server.URL + "/v1", APIKey: "local-test-only", Model: "local-test", Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	owner := time.Now().UnixNano()
	t.Cleanup(func() {
		if _, err := conn.ExecCtx(context.Background(), "DELETE FROM ai_message WHERE conversation_id IN (SELECT id FROM ai_conversation WHERE user_id=?)", owner); err != nil {
			t.Error("清理本轮隔离消息失败")
		}
		if _, err := conn.ExecCtx(context.Background(), "DELETE FROM ai_conversation WHERE user_id=?", owner); err != nil {
			t.Error("清理本轮隔离会话失败")
		}
	})
	return &svc.ServiceContext{Config: config.Config{AI: config.ConfigAI{TimeoutSeconds: 2, MaxContextTokens: 32000}}, DB: conn, Redis: rdb, LLM: client,
		AiConversationModel: model.NewAiConversationModel(conn, cacheConfig), AiMessageModel: model.NewAiMessageModel(conn, cacheConfig)}, owner
}

func runAITestStream(t *testing.T, ctx context.Context, service *svc.ServiceContext, request *aipb.SendMessageRequest, onSend func(*aipb.StreamChunk) error) *aiTestStream {
	t.Helper()
	stream := &aiTestStream{ctx: ctx, onSend: onSend}
	if err := NewSendMessageStreamLogic(ctx, service).SendMessageStream(request, stream); err != nil {
		t.Fatal(err)
	}
	return stream
}

func TestAIStreamIntegrationFinalResultAndReplay(t *testing.T) {
	var calls atomic.Int32
	service, owner := aiIntegrationContext(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"本地结果\"},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":3,\"total_tokens\":5}}\n\ndata: [DONE]\n\n")
	})
	request := &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), Content: "本地输入"}
	first := runAITestStream(t, context.Background(), service, request, nil).last()
	if first.Type != "done" || first.Result.GetConversation().GetMessageCount() != 2 || first.Result.GetAiMessage().GetStatus() != "succeeded" || first.Result.GetAiMessage().GetContent() != "本地结果" {
		t.Fatal("新发送缺少完整持久化结果")
	}
	rows, err := service.AiMessageModel.ListPageOwned(context.Background(), first.Result.ConversationId, owner, 0, 20)
	if err != nil || len(rows) != 2 || rows[0].Id != first.Result.AiMessage.Id || rows[0].Content != first.Result.AiMessage.Content || rows[0].Tokens != first.Tokens {
		t.Fatal("终帧与数据库消息不一致")
	}
	service.LLM = nil
	replay := runAITestStream(t, context.Background(), service, request, nil).last()
	if replay.Type != "done" || !proto.Equal(first.Result, replay.Result) || calls.Load() != 1 {
		t.Fatal("模型关闭后应重放同一持久化结果且不再次调用")
	}
	changed := proto.Clone(request).(*aipb.SendMessageRequest)
	changed.Content = "另一个意图"
	if got := runAITestStream(t, context.Background(), service, changed, nil).last(); got.Type != "error" || got.ErrorCode != 400 {
		t.Fatal("复用请求 ID 更改内容必须拒绝")
	}
	changed.RequestId = uuid.NewString()
	if got := runAITestStream(t, context.Background(), service, changed, nil).last(); got.Type != "error" || got.ErrorCode != 503 {
		t.Fatal("模型关闭的新请求必须拒绝")
	}
	list, total, err := service.AiConversationModel.ListOwned(context.Background(), owner, 1, 20)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatal("模型关闭不应预留新消息或空会话")
	}
	foreign := &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner + 1}, RequestId: uuid.NewString(), ConversationId: first.Result.ConversationId, Content: "越权输入"}
	// 模型开启后才能进入预留阶段的归属校验；复用本地客户端，不访问外部服务。
	service.LLM, _ = llm.New(llm.Config{BaseURL: "http://127.0.0.1:1/v1", APIKey: "local-test-only", Model: "local-test", Timeout: 1})
	if got := runAITestStream(t, context.Background(), service, foreign, nil).last(); got.Type != "error" || got.ErrorCode != 404 {
		t.Fatal("其他用户的会话必须按不存在返回")
	}
}

func TestAIStreamIntegrationConcurrentRequestDoesNotCallModelTwice(t *testing.T) {
	var calls atomic.Int32
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	service, owner := aiIntegrationContext(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		entered <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"唯一结果\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	})
	request := &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), Content: "并发输入"}
	finished := make(chan *aiTestStream, 1)
	go func() {
		stream := &aiTestStream{ctx: context.Background()}
		_ = NewSendMessageStreamLogic(stream.ctx, service).SendMessageStream(request, stream)
		finished <- stream
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		t.Fatal("本地模型未收到请求")
	}
	duplicate := runAITestStream(t, context.Background(), service, request, nil).last()
	close(release)
	first := <-finished
	if duplicate.Type != "error" || duplicate.ErrorCode != 400 || first.last().Type != "done" || calls.Load() != 1 {
		t.Fatal("并发同请求必须仅发起一次模型调用")
	}
}

func TestAIStreamIntegrationFailureAndDeadlinePersistFailed(t *testing.T) {
	for _, slow := range []bool{false, true} {
		t.Run(fmt.Sprintf("slow=%v", slow), func(t *testing.T) {
			service, owner := aiIntegrationContext(t, func(w http.ResponseWriter, r *http.Request) {
				if slow {
					select {
					case <-r.Context().Done():
					case <-time.After(1500 * time.Millisecond):
					}
					return
				}
				http.Error(w, "local failure", http.StatusServiceUnavailable)
			})
			service.Config.AI.TimeoutSeconds = 1
			request := &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), Content: "失败测试"}
			started := time.Now()
			stream := runAITestStream(t, context.Background(), service, request, nil)
			if stream.last().Type != "error" || time.Since(started) > 3*time.Second {
				t.Fatal("失败或超时未在截止时间内结束")
			}
			pair, err := service.AiMessageModel.FindCall(context.Background(), owner, fmt.Sprintf("%d:%s", owner, request.RequestId))
			if err != nil || pair.User.Status != "failed" || pair.Assistant.Status != "failed" || pair.Assistant.Content != "" {
				t.Fatal("失败后消息对必须原子失败且助手正文为空")
			}
		})
	}
}

func TestAIStreamIntegrationLaterFailureKeepsKnownUsageIncomplete(t *testing.T) {
	var requests atomic.Int32
	service, owner := aiIntegrationContext(t, func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) > 1 {
			http.Error(w, "local failure", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		writeModelFrame(t, w, map[string]any{"choices": []any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "usage-call", "type": "function", "function": map[string]any{"name": "text_stats", "arguments": `{"text":"本地工具"}`}}}}, "finish_reason": "tool_calls"}}, "usage": map[string]any{"prompt_tokens": 10, "completion_tokens": 3, "total_tokens": 13}})
		fmt.Fprint(w, "data: [DONE]\n\n")
	})
	request := &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), Content: "次轮失败用量"}
	stream := runAITestStream(t, context.Background(), service, request, nil)
	pair, err := service.AiMessageModel.FindCall(context.Background(), owner, fmt.Sprintf("%d:%s", owner, request.RequestId))
	if err != nil {
		t.Fatal(err)
	}
	var record callRecord
	if json.Unmarshal([]byte(pair.Assistant.ToolCalls), &record) != nil {
		t.Fatal("失败调用元数据无效")
	}
	if stream.last().Type != "error" || requests.Load() != 2 || pair.Assistant.Status != "failed" || pair.Assistant.Content != "" || pair.Assistant.Tokens != 13 || record.UsageComplete {
		t.Fatalf("未知次轮必须保留13个已知tokens并标不完整: status=%s tokens=%d complete=%v", pair.Assistant.Status, pair.Assistant.Tokens, record.UsageComplete)
	}
}

func TestAIStreamIntegrationBrokenFrameKeepsReceivedUsage(t *testing.T) {
	service, owner := aiIntegrationContext(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"PRIVATE_REPLY\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5,\"total_tokens\":15}}\n\ndata: {broken-json\n\n")
	})
	request := &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), Content: "坏帧用量"}
	stream := runAITestStream(t, context.Background(), service, request, nil)
	pair, err := service.AiMessageModel.FindCall(context.Background(), owner, fmt.Sprintf("%d:%s", owner, request.RequestId))
	if err != nil {
		t.Fatal(err)
	}
	var record callRecord
	if json.Unmarshal([]byte(pair.Assistant.ToolCalls), &record) != nil || stream.last().Type != "error" || pair.Assistant.Status != "failed" || pair.Assistant.Content != "" || pair.Assistant.Tokens != 15 || record.UsageComplete {
		t.Fatal("收到usage后坏帧须在原失败消息保存15 tokens且不完整，不保存部分正文")
	}
}

func TestAIStreamIntegrationDeletedConversationIsNotRestored(t *testing.T) {
	service, owner := aiIntegrationContext(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"局部内容\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	})
	var conversationId int64
	stream := runAITestStream(t, context.Background(), service, &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), Content: "删除测试"}, func(chunk *aipb.StreamChunk) error {
		if chunk.Type == "meta" {
			conversationId = chunk.ConversationId
		}
		if chunk.Type == "delta" {
			return service.AiConversationModel.DeleteOwned(context.Background(), conversationId, owner)
		}
		return nil
	})
	if stream.last().Type != "done" || stream.last().Result == nil || stream.last().Result.Conversation != nil {
		t.Fatal("发送期间删除的会话不能由终帧恢复")
	}
	if _, err := service.AiConversationModel.FindOwned(context.Background(), conversationId, owner); !errors.Is(err, model.ErrNotFound) {
		t.Fatal("已删除会话被恢复")
	}
}

func TestAIStreamIntegrationTrimsEscapedHistoryBeforeBudgetGuard(t *testing.T) {
	var calls atomic.Int32
	service, owner := aiIntegrationContext(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"预算正常\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	})
	service.Config.AI.MaxContextTokens = 1000
	created, err := service.DB.ExecCtx(context.Background(), "INSERT INTO ai_conversation(user_id,title) VALUES (?,?)", owner, "预算回归")
	if err != nil {
		t.Fatal("创建隔离预算会话失败")
	}
	id, _ := created.LastInsertId()
	toolData, _ := json.Marshal(agenttools.LLMTools())
	history := strings.Repeat("\"", (3000-len(toolData)-len("预算测试"))*3/4)
	if _, err := service.DB.ExecCtx(context.Background(), "INSERT INTO ai_message(conversation_id,role,content) VALUES (?,'user',?),(?,'assistant','旧结果')", id, history, id); err != nil {
		t.Fatal("创建隔离预算历史失败")
	}
	request := &aipb.SendMessageRequest{Actor: &aipb.Actor{UserId: owner}, RequestId: uuid.NewString(), ConversationId: id, Content: "预算测试"}
	if got := runAITestStream(t, context.Background(), service, request, nil).last(); got.Type != "done" || calls.Load() != 1 {
		t.Fatal("应按同一口径裁剪历史后正常调用，不能在发送前误拒绝")
	}
}
