package aiservicelogic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
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

const streamCleanupTimeout = 5 * time.Second

type SendMessageStreamLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendMessageStreamLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendMessageStreamLogic {
	return &SendMessageStreamLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// SendMessageStream 是唯一发送入口；重查复用请求 ID，业务失败通过 error 帧返回。
func (l *SendMessageStreamLogic) SendMessageStream(in *aipb.SendMessageRequest, stream aipb.AiService_SendMessageStreamServer) error {
	send := func(chunk *aipb.StreamChunk) error { return stream.Send(chunk) }
	sendErr := func(code uint32, msg string) error {
		// 客户端已断开时错误帧无处可去，直接结束流。
		_ = send(&aipb.StreamChunk{Type: "error", Content: msg, ErrorCode: int32(code)})
		return nil
	}

	if err := requireAiActor(in.Actor); err != nil {
		return sendErr(xerr.PERMISSION_DENIED, "未授权")
	}
	requestId, err := uuid.Parse(in.RequestId)
	if err != nil || len(in.RequestId) != 36 || !strings.EqualFold(requestId.String(), in.RequestId) || in.ConversationId < 0 || strings.TrimSpace(in.Content) == "" || len(in.Content) > 16000 {
		return sendErr(400, "request_id 须为规范 UUID，消息须为 1..16000 字节且会话 ID 不得为负")
	}
	callKey := fmt.Sprintf("%d:%s", in.Actor.UserId, requestId.String())
	record := callRecord{Version: 1, Calls: make([]schema.ToolCall, 0), Fingerprint: requestFingerprint(in.ConversationId, in.Content)}
	timeout := l.svcCtx.Config.AI.TimeoutSeconds
	if timeout < 1 || timeout > 90 {
		timeout = 60
	}
	ctx, cancel := context.WithTimeout(l.ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	// 未启用模型时仍可读取已有结果；新请求不预留消息或创建空会话。
	if l.svcCtx.LLM == nil {
		if pair, err := l.svcCtx.AiMessageModel.FindCall(ctx, in.Actor.UserId, callKey); err == nil {
			return l.replayStream(pair, in.Actor.UserId, record.Fingerprint, send)
		} else if err != model.ErrNotFound {
			return sendErr(xerr.DB_ERROR, "查询调用记录失败")
		}
		return sendErr(xerr.SERVICE_UNAVAILABLE, "AI 模型未启用或未配置密钥")
	}
	// ReserveCall 内含一次幂等读取，并以唯一索引兜底并发预留。
	pair, fresh, err := l.svcCtx.AiMessageModel.ReserveCall(ctx, in.Actor.UserId, in.ConversationId, callKey, in.Content, record.marshal())
	if err != nil {
		coded, _ := xerr.AsCodeError(aiModelError(err))
		return sendErr(coded.GetErrCode(), coded.GetErrMsg())
	}
	if !fresh {
		return l.replayStream(pair, in.Actor.UserId, record.Fingerprint, send)
	}
	if err := send(&aipb.StreamChunk{Type: "meta", ConversationId: pair.ConversationId, UserMessageId: pair.User.Id, AiMessageId: pair.Assistant.Id}); err != nil {
		return nil // 客户端已断开，运行中消息对保留供重查
	}

	var tokens int64
	fail := func(cause error) error {
		var code uint32
		var msg string
		if ce, ok := xerr.AsCodeError(cause); ok {
			code, msg = ce.GetErrCode(), ce.GetErrMsg()
		} else {
			code, msg = xerr.SERVER_COMMON_ERROR, "AI 调用未完成，请使用新的 request_id 重试"
		}
		record.ErrorCode, record.Error = code, msg
		cleanup, done := context.WithTimeout(context.WithoutCancel(l.ctx), streamCleanupTimeout)
		defer done()
		if err := l.svcCtx.AiMessageModel.FinishCall(cleanup, in.Actor.UserId, pair, "failed", "", record.marshal(), tokens); err != nil {
			l.Errorf("AI 流式失败落库失败: %v", err)
		}
		return sendErr(code, msg)
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
		cleanup, done := context.WithTimeout(context.WithoutCancel(l.ctx), streamCleanupTimeout)
		defer done()
		if _, err := lock.ReleaseCtx(cleanup); err != nil {
			l.Error("AI 会话锁释放失败")
		}
	}()
	if err := l.svcCtx.AiConversationModel.CheckOwned(ctx, pair.ConversationId, in.Actor.UserId); err != nil {
		return fail(aiModelError(err))
	}
	// 预算制：取足够大的候选池（200 条），再按 token 预算从新到旧过滤。
	history, err := l.svcCtx.AiMessageModel.ListOwned(ctx, pair.ConversationId, in.Actor.UserId, 200)
	if err != nil {
		return fail(aiModelError(err))
	}
	messages := make([]*schema.Message, 0, len(history)+1)
	budget := newContextBudget(l.svcCtx)
	toolDefinitions, _ := json.Marshal(agenttools.LLMTools())
	if !budget.tryAdd(string(toolDefinitions)+"[") || !budget.tryAddMessage(schema.UserMessage(in.Content)) {
		return fail(xerr.NewErrCodeMsg(400, "消息与工具定义超过上下文预算，请缩短消息"))
	}
	for i := len(history) - 1; i >= 0; i-- {
		row := history[i]
		var message *schema.Message
		if row.Role == "user" {
			message = schema.UserMessage(row.Content)
		} else if row.Role == "assistant" {
			message = schema.AssistantMessage(row.Content, nil)
		} else {
			continue
		}
		if !budget.tryAddMessage(message) {
			break
		}
		messages = append(messages, message)
	}
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	messages = append(messages, schema.UserMessage(in.Content))

	// 工具执行需按操作者归属过滤（如 kb_search 只查本人知识库），身份经 ctx 传递而非工具参数（LLM 输出不可信）
	answer, complete, err := askModelStream(agenttools.WithActor(ctx, in.Actor.UserId), l.svcCtx.LLM, messages, send, func(total int64, calls []schema.ToolCall, usageComplete bool) error {
		tokens = total
		record.Calls = calls
		record.UsageComplete = usageComplete
		checkpoint := record
		checkpoint.UsageComplete = false
		return l.svcCtx.AiMessageModel.SaveCallUsage(ctx, in.Actor.UserId, pair, tokens, checkpoint.marshal())
	}, budget.maxTokens)
	if err != nil {
		return fail(err)
	}
	record.UsageComplete = complete
	cleanup, done := context.WithTimeout(context.WithoutCancel(l.ctx), streamCleanupTimeout)
	defer done()
	if err := l.svcCtx.AiMessageModel.FinishCall(cleanup, in.Actor.UserId, pair, "succeeded", answer, record.marshal(), tokens); err != nil {
		return sendErr(xerr.SERVER_COMMON_ERROR, "保存 AI 回复失败")
	}
	pair.User.Status = "succeeded"
	pair.User.ToolCalls = record.marshal()
	pair.Assistant.Status = "succeeded"
	pair.Assistant.Content = answer
	pair.Assistant.Tokens = tokens
	pair.Assistant.ToolCalls = record.marshal()
	return send(&aipb.StreamChunk{Type: "done", Tokens: tokens, UsageComplete: complete, Result: callResponse(pair)})
}

// replayStream 成功补发持久化结果，失败或处理中返回原安全错误，不重新执行。
func (l *SendMessageStreamLogic) replayStream(pair *model.CallPair, ownerId int64, fingerprint string, send func(*aipb.StreamChunk) error) error {
	var record callRecord
	if json.Unmarshal([]byte(pair.Assistant.ToolCalls), &record) != nil || record.Version != 1 || record.Fingerprint != fingerprint {
		_ = send(&aipb.StreamChunk{Type: "error", Content: "request_id 已用于不同的会话或消息内容", ErrorCode: 400})
		return nil
	}
	sendErr := func(code uint32, msg string) error {
		_ = send(&aipb.StreamChunk{Type: "error", Content: msg, ErrorCode: int32(code)})
		return nil
	}
	switch pair.Assistant.Status {
	case "succeeded":
		conversation, err := l.svcCtx.AiConversationModel.FindOwned(l.ctx, pair.ConversationId, ownerId)
		if err != nil {
			coded, _ := xerr.AsCodeError(aiModelError(err))
			return sendErr(coded.GetErrCode(), coded.GetErrMsg())
		}
		pair.Conversation = conversation
		if err := send(&aipb.StreamChunk{Type: "meta", ConversationId: pair.ConversationId, UserMessageId: pair.User.Id, AiMessageId: pair.Assistant.Id}); err != nil {
			return nil
		}
		if pair.Assistant.Content != "" {
			if err := send(&aipb.StreamChunk{Type: "delta", Content: pair.Assistant.Content}); err != nil {
				return nil
			}
		}
		_ = send(&aipb.StreamChunk{Type: "done", Tokens: pair.Assistant.Tokens, UsageComplete: record.UsageComplete, Result: callResponse(pair)})
		return nil
	case "failed":
		if record.ErrorCode == 0 || record.Error == "" {
			return sendErr(503, "AI 调用失败，请使用新的 request_id 重试")
		}
		return sendErr(record.ErrorCode, record.Error)
	default:
		return sendErr(400, "消息仍在处理中或执行已中断；不会自动重放，请查询会话并使用新的 request_id 提交新意图")
	}
}

// askModelStream 流式模型调用：正文增量经 emit 实时下发，工具调用分片轮末拼合后执行。
func askModelStream(ctx context.Context, client *llm.Client, messages []*schema.Message, emit func(*aipb.StreamChunk) error, persist func(int64, []schema.ToolCall, bool) error, maxTokens int64) (string, bool, error) {
	calls := make([]schema.ToolCall, 0)
	complete := true
	var tokens int64
	tools := agenttools.LLMTools()
	toolDefinitions, _ := json.Marshal(tools)
	for round := 0; round < 5; round++ {
		prompt, _ := json.Marshal(messages)
		budget := contextBudget{maxTokens: maxTokens}
		if !budget.tryAdd(string(toolDefinitions)) || !budget.tryAdd(string(prompt)) {
			return "", complete, xerr.NewErrCodeMsg(400, "对话与工具结果超过上下文预算，请简化问题")
		}
		reader, callErr := client.ChatStream(ctx, messages, tools)
		if callErr != nil {
			return "", false, xerr.NewErrCodeMsg(503, "模型调用失败，请稍后重试")
		}
		var roundContent strings.Builder
		partial := map[int]*schema.ToolCall{}
		var usage *schema.TokenUsage
		for {
			frame, recvErr := reader.Recv()
			if recvErr == io.EOF {
				break
			}
			if recvErr != nil {
				reader.Close()
				return "", false, xerr.NewErrCodeMsg(503, "模型调用失败，请稍后重试")
			}
			if frame.Content != "" {
				if roundContent.Len()+len(frame.Content) > 60000 {
					reader.Close()
					return "", false, xerr.NewErrCodeMsg(500, "模型回复超过大小限制")
				}
				roundContent.WriteString(frame.Content)
				if err := emit(&aipb.StreamChunk{Type: "delta", Content: frame.Content}); err != nil {
					reader.Close()
					return "", false, xerr.NewErrCodeMsg(503, "AI 调用未完成，请使用新的 request_id 重试")
				}
			}
			for i := range frame.ToolCalls {
				mergeToolCallChunk(partial, frame.ToolCalls[i])
			}
			if frame.ResponseMeta != nil && frame.ResponseMeta.Usage != nil {
				usage = frame.ResponseMeta.Usage
			}
		}
		reader.Close()

		if usage == nil || usage.TotalTokens == 0 && usage.PromptTokens == 0 && usage.CompletionTokens == 0 {
			complete = false
		} else if usage.TotalTokens < 0 {
			return "", false, xerr.NewErrCodeMsg(500, "模型用量响应无效")
		} else {
			tokens += int64(usage.TotalTokens)
		}
		roundCalls := flattenToolCalls(partial)
		proposed, validationErr := mergeRoundCalls(calls, roundCalls)
		if validationErr == nil {
			calls = proposed
		}
		// 用量与本轮工具意图在同一检查点落盘，且始终早于工具执行。
		if err := persist(tokens, calls, complete); err != nil {
			return "", false, err
		}
		if tokens > 10_000_000 {
			return "", complete, xerr.NewErrCodeMsg(500, "模型用量响应超出限制")
		}
		if validationErr != nil {
			return "", complete, validationErr
		}
		if len(roundCalls) == 0 {
			content := roundContent.String()
			if strings.TrimSpace(content) == "" || len(content) > 60000 {
				return "", complete, xerr.NewErrCodeMsg(500, "模型回复为空或超过大小限制")
			}
			return content, complete, nil
		}
		messages = append(messages, schema.AssistantMessage(roundContent.String(), roundCalls))
		for _, call := range roundCalls {
			if err := emit(&aipb.StreamChunk{Type: "tool", Content: call.Function.Name, ToolStatus: "start"}); err != nil {
				return "", false, xerr.NewErrCodeMsg(503, "AI 调用未完成，请使用新的 request_id 重试")
			}
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
				// 工具输出截断到 4000 字符，防止超长结果撑爆后续模型调用。
				logx.WithContext(ctx).Infof("AI 工具 %s 输出截断: %d 字节 → 4000 字符", call.Function.Name, len(result))
				runes := []rune(result)
				result = string(runes[:min(len(runes), 4000)])
			}
			if err := emit(&aipb.StreamChunk{Type: "tool", Content: call.Function.Name, ToolStatus: "done"}); err != nil {
				return "", false, xerr.NewErrCodeMsg(503, "AI 调用未完成，请使用新的 request_id 重试")
			}
			messages = append(messages, schema.ToolMessage(result, call.ID))
		}
	}
	return "", complete, xerr.NewErrCodeMsg(500, "已达到工具调用轮数限制，请简化问题")
}

func mergeRoundCalls(previous, next []schema.ToolCall) ([]schema.ToolCall, error) {
	if len(previous)+len(next) > 16 {
		return nil, xerr.NewErrCodeMsg(500, "模型请求过多工具调用")
	}
	for _, call := range next {
		if len(call.Function.Arguments) > 64000 {
			return nil, xerr.NewErrCodeMsg(500, "工具参数超过大小限制")
		}
	}
	proposed := append(append([]schema.ToolCall{}, previous...), next...)
	data, err := json.Marshal(proposed)
	if err != nil || len(data) > 60000 {
		return nil, xerr.NewErrCodeMsg(500, "工具调用记录超过大小限制")
	}
	return proposed, nil
}

// mergeToolCallChunk 按 index 累积流式工具调用分片（首片带 ID/Name，后续片只带 Arguments 增量）。
func mergeToolCallChunk(partial map[int]*schema.ToolCall, chunk schema.ToolCall) {
	idx := 0
	if chunk.Index != nil {
		idx = *chunk.Index
	}
	call, ok := partial[idx]
	if !ok {
		call = &schema.ToolCall{Type: "function"}
		call.Index = new(int)
		*call.Index = idx
		partial[idx] = call
	}
	if chunk.ID != "" {
		call.ID = chunk.ID
	}
	if chunk.Type != "" {
		call.Type = chunk.Type
	}
	if chunk.Function.Name != "" {
		call.Function.Name = chunk.Function.Name
	}
	call.Function.Arguments += chunk.Function.Arguments
}

func flattenToolCalls(partial map[int]*schema.ToolCall) []schema.ToolCall {
	if len(partial) == 0 {
		return nil
	}
	keys := make([]int, 0, len(partial))
	for k := range partial {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	out := make([]schema.ToolCall, 0, len(keys))
	for _, k := range keys {
		out = append(out, *partial[k])
	}
	return out
}
