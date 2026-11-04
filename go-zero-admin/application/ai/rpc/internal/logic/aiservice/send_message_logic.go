package aiservicelogic

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/xerr"
)

type SendMessageLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendMessageLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendMessageLogic {
	return &SendMessageLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// callRecord 只在存储层保存，外部 tool_calls 始终映射成数组字符串。
type callRecord struct {
	Version       int               `json:"version"`
	Calls         []schema.ToolCall `json:"calls"`
	Fingerprint   string            `json:"fingerprint"`
	Error         string            `json:"error,omitempty"`
	ErrorCode     uint32            `json:"error_code,omitempty"`
	UsageComplete bool              `json:"usage_complete"`
}

func (r callRecord) marshal() string { data, _ := json.Marshal(r); return string(data) }
func requestFingerprint(conversationId int64, content string) string {
	data, _ := json.Marshal([]any{conversationId, content})
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func (l *SendMessageLogic) SendMessage(in *aipb.SendMessageRequest) (*aipb.SendMessageResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	requestId, err := uuid.Parse(in.RequestId)
	if err != nil || len(in.RequestId) != 36 || !strings.EqualFold(requestId.String(), in.RequestId) || in.ConversationId < 0 || strings.TrimSpace(in.Content) == "" || len(in.Content) > 16000 {
		return nil, xerr.NewErrCodeMsg(400, "request_id 须为规范 UUID，消息须为 1..16000 字节且会话 ID 不得为负")
	}
	callId := fmt.Sprintf("%d:%s", in.Actor.UserId, requestId.String())
	record := callRecord{Version: 1, Calls: make([]schema.ToolCall, 0), Fingerprint: requestFingerprint(in.ConversationId, in.Content)}
	if pair, err := l.svcCtx.AiMessageModel.FindCall(l.ctx, in.Actor.UserId, callId); err == nil {
		return replayCall(pair, record.Fingerprint)
	} else if err != model.ErrNotFound {
		return nil, aiModelError(err)
	}
	if l.svcCtx.LLM == nil {
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "AI 模型未启用或未配置密钥")
	}
	pair, fresh, err := l.svcCtx.AiMessageModel.ReserveCall(l.ctx, in.Actor.UserId, in.ConversationId, callId, in.Content, record.marshal())
	if err != nil {
		return nil, aiModelError(err)
	}
	if !fresh {
		return replayCall(pair, record.Fingerprint)
	}
	timeout := l.svcCtx.Config.AI.TimeoutSeconds
	if timeout < 1 || timeout > 90 {
		timeout = 60
	}
	ctx, cancel := context.WithTimeout(l.ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	var tokens int64
	fail := func(cause error) (*aipb.SendMessageResponse, error) {
		if code, ok := xerr.AsCodeError(cause); ok {
			record.ErrorCode = code.GetErrCode()
			record.Error = code.GetErrMsg()
		} else {
			record.ErrorCode = xerr.SERVER_COMMON_ERROR
			record.Error = "AI 调用未完成，请使用新的 request_id 重试"
		}
		cleanup, done := context.WithTimeout(context.WithoutCancel(l.ctx), 5*time.Second)
		defer done()
		if err := l.svcCtx.AiMessageModel.FinishCall(cleanup, in.Actor.UserId, pair, "failed", "", record.marshal(), tokens); err != nil {
			return nil, aiModelError(err)
		}
		return nil, xerr.NewErrCodeMsg(record.ErrorCode, record.Error)
	}
	lock := redis.NewRedisLock(l.svcCtx.Redis, fmt.Sprintf("ai:conversation:send:%d", pair.ConversationId))
	lock.SetExpire(120)
	locked, err := lock.AcquireCtx(ctx)
	if err != nil {
		return fail(xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE))
	}
	if !locked {
		return fail(xerr.NewErrCodeMsg(400, "会话已有消息处理中，请使用新的 request_id 重试"))
	}
	defer func() {
		cleanup, done := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer done()
		if _, err := lock.ReleaseCtx(cleanup); err != nil {
			l.Error("AI 会话锁释放失败")
		}
	}()
	// Redis 锁取得前会话可能已删除；公共可见性与预留调用的结算权限分别校验。
	if _, err := l.svcCtx.AiConversationModel.FindOwned(ctx, pair.ConversationId, in.Actor.UserId); err != nil {
		return fail(aiModelError(err))
	}
	history, err := l.svcCtx.AiMessageModel.ListOwned(ctx, pair.ConversationId, in.Actor.UserId, 20)
	if err != nil {
		return fail(aiModelError(err))
	}
	messages := make([]*schema.Message, 0, len(history)+1)
	for _, row := range history {
		if row.Role == "user" {
			messages = append(messages, schema.UserMessage(row.Content))
		} else if row.Role == "assistant" {
			messages = append(messages, schema.AssistantMessage(row.Content, nil))
		}
	}
	messages = append(messages, schema.UserMessage(in.Content))
	answer, trace, known, complete, err := askModelWithProgress(ctx, l.svcCtx.LLM, messages, func(total int64, calls []schema.ToolCall, usageComplete bool) error {
		tokens = total
		record.Calls = calls
		record.UsageComplete = usageComplete
		// running 始终 incomplete；下一轮外部响应与落库之间的崩溃用量未知。
		checkpoint := record
		checkpoint.UsageComplete = false
		if err := l.svcCtx.AiMessageModel.SaveCallUsage(ctx, in.Actor.UserId, pair, tokens, checkpoint.marshal()); err != nil {
			return aiModelError(err)
		}
		return nil
	})
	tokens = known
	record.UsageComplete = complete
	if trace != "" {
		_ = json.Unmarshal([]byte(trace), &record.Calls)
	}
	if err != nil {
		return fail(err)
	}
	cleanup, done := context.WithTimeout(context.WithoutCancel(l.ctx), 5*time.Second)
	defer done()
	if err := l.svcCtx.AiMessageModel.FinishCall(cleanup, in.Actor.UserId, pair, "succeeded", answer, record.marshal(), tokens); err != nil {
		return nil, aiModelError(err)
	}
	pair.User.Status = "succeeded"
	pair.User.ToolCalls = record.marshal()
	pair.Assistant.Status = "succeeded"
	pair.Assistant.Content = answer
	pair.Assistant.Tokens = tokens
	pair.Assistant.ToolCalls = record.marshal()
	return callResponse(pair), nil
}
func callResponse(pair *model.CallPair) *aipb.SendMessageResponse {
	return &aipb.SendMessageResponse{ConversationId: pair.ConversationId, UserMessage: messageProto(pair.User), AiMessage: messageProto(pair.Assistant)}
}
func replayCall(pair *model.CallPair, fingerprint string) (*aipb.SendMessageResponse, error) {
	var record callRecord
	if json.Unmarshal([]byte(pair.Assistant.ToolCalls), &record) != nil || record.Version != 1 || record.Fingerprint != fingerprint {
		return nil, xerr.NewErrCodeMsg(400, "request_id 已用于不同的会话或消息内容")
	}
	switch pair.Assistant.Status {
	case "succeeded":
		return callResponse(pair), nil
	case "failed":
		if record.ErrorCode == 0 || record.Error == "" {
			return nil, xerr.NewErrCodeMsg(503, "AI 调用失败，请使用新的 request_id 重试")
		}
		return nil, xerr.NewErrCodeMsg(record.ErrorCode, record.Error)
	default:
		return nil, xerr.NewErrCodeMsg(400, "消息仍在处理中或执行已中断；不会自动重放，请查询会话并使用新的 request_id 提交新意图")
	}
}

func askModel(ctx context.Context, client *llm.Client, messages []*schema.Message) (string, string, int64, error) {
	answer, trace, tokens, _, err := askModelWithProgress(ctx, client, messages, func(int64, []schema.ToolCall, bool) error { return nil })
	return answer, trace, tokens, err
}
func askModelWithProgress(ctx context.Context, client *llm.Client, messages []*schema.Message, persist func(int64, []schema.ToolCall, bool) error) (answer, trace string, tokens int64, complete bool, err error) {
	calls := make([]schema.ToolCall, 0)
	complete = true
	defer func() { data, _ := json.Marshal(calls); trace = string(data) }()
	for round := 0; round < 5; round++ {
		response, usage, callErr := client.Chat(ctx, messages, agenttools.LLMTools())
		if callErr != nil {
			return "", "", tokens, false, xerr.NewErrCodeMsg(503, "模型调用失败，请稍后重试")
		}
		if usage == nil || usage.TotalTokens == 0 && usage.PromptTokens == 0 && usage.CompletionTokens == 0 {
			// 当前 Eino adapter 把缺失/null usage 转为非nil的全零结构，无法据此认定已知零。
			complete = false
		} else if usage.TotalTokens < 0 {
			return "", "", tokens, false, xerr.NewErrCodeMsg(500, "模型用量响应无效")
		} else {
			tokens += int64(usage.TotalTokens)
		}
		if err := persist(tokens, calls, complete); err != nil {
			return "", "", tokens, false, err
		}
		if tokens > 10_000_000 {
			return "", "", tokens, complete, xerr.NewErrCodeMsg(500, "模型用量响应超出限制")
		}
		if response == nil {
			return "", "", tokens, false, xerr.NewErrCodeMsg(503, "模型响应为空")
		}
		if len(response.ToolCalls) == 0 {
			if strings.TrimSpace(response.Content) == "" || len(response.Content) > 60000 {
				return "", "", tokens, complete, xerr.NewErrCodeMsg(500, "模型回复为空或超过大小限制")
			}
			return response.Content, "", tokens, complete, nil
		}
		if len(calls)+len(response.ToolCalls) > 16 {
			return "", "", tokens, complete, xerr.NewErrCodeMsg(500, "模型请求过多工具调用")
		}
		for _, call := range response.ToolCalls {
			if len(call.Function.Arguments) > 64000 {
				return "", "", tokens, complete, xerr.NewErrCodeMsg(500, "工具参数超过大小限制")
			}
		}
		proposed := append(append([]schema.ToolCall{}, calls...), response.ToolCalls...)
		data, marshalErr := json.Marshal(proposed)
		if marshalErr != nil || len(data) > 60000 {
			return "", "", tokens, complete, xerr.NewErrCodeMsg(500, "工具调用记录超过大小限制")
		}
		calls = proposed
		if err := persist(tokens, calls, complete); err != nil {
			return "", "", tokens, false, err
		}
		messages = append(messages, response)
		for _, call := range response.ToolCalls {
			result := "工具不存在"
			if tool, ok := agenttools.Get(call.Function.Name); ok {
				value, toolErr := tool.Execute(ctx, json.RawMessage(call.Function.Arguments))
				if toolErr != nil {
					result = "工具参数不合法或执行失败"
				} else {
					result = value
				}
			}
			if len(result) > 16000 {
				runes := []rune(result)
				result = string(runes[:min(len(runes), 4000)])
			}
			messages = append(messages, schema.ToolMessage(result, call.ID))
		}
	}
	return "", "", tokens, complete, xerr.NewErrCodeMsg(500, "已达到工具调用轮数限制，请简化问题")
}
