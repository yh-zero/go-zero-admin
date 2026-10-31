package aiservicelogic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
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

func (l *SendMessageLogic) SendMessage(in *aipb.SendMessageRequest) (*aipb.SendMessageResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	content := strings.TrimSpace(in.Content)
	if content == "" || len(content) > 16000 || in.ConversationId < 0 {
		return nil, xerr.NewErrCodeMsg(400, "消息内容须为 1..16000 字节，会话 ID 不得为负")
	}
	conversationId := in.ConversationId
	if conversationId > 0 {
		if _, err := l.svcCtx.AiConversationModel.FindOwned(l.ctx, conversationId, in.Actor.UserId); err != nil {
			return nil, aiModelError(err)
		}
	}
	if l.svcCtx.LLM == nil {
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "AI 模型未启用或未配置密钥")
	}
	timeout := l.svcCtx.Config.AI.TimeoutSeconds
	if timeout < 1 || timeout > 90 {
		timeout = 60
	}
	ctx, cancel := context.WithTimeout(l.ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	if conversationId == 0 {
		title := []rune(content)
		if len(title) > 64 {
			title = title[:64]
		}
		result, err := l.svcCtx.AiConversationModel.Insert(ctx, &model.AiConversation{UserId: uint64(in.Actor.UserId), Title: string(title)})
		if err != nil {
			return nil, aiModelError(err)
		}
		conversationId, err = result.LastInsertId()
		if err != nil {
			return nil, aiModelError(err)
		}
	}
	lock := redis.NewRedisLock(l.svcCtx.Redis, fmt.Sprintf("ai:conversation:send:%d", conversationId))
	lock.SetExpire(120)
	locked, err := lock.AcquireCtx(ctx)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	if !locked {
		return nil, xerr.NewErrCodeMsg(400, "会话已有消息处理中，请稍后重试")
	}
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer done()
		if _, err := lock.ReleaseCtx(cleanupCtx); err != nil {
			l.Error("AI 会话锁释放失败")
		}
	}()
	if _, err := l.svcCtx.AiConversationModel.FindOwned(ctx, conversationId, in.Actor.UserId); err != nil {
		return nil, aiModelError(err)
	}
	history, err := l.svcCtx.AiMessageModel.ListOwned(ctx, conversationId, in.Actor.UserId, 20)
	if err != nil {
		return nil, aiModelError(err)
	}
	messages := make([]*schema.Message, 0, len(history)+1)
	for _, row := range history {
		switch row.Role {
		case "user":
			messages = append(messages, schema.UserMessage(row.Content))
		case "assistant":
			messages = append(messages, schema.AssistantMessage(row.Content, nil))
		}
	}
	messages = append(messages, schema.UserMessage(content))
	answer, trace, tokens, err := askModel(ctx, l.svcCtx.LLM, messages)
	if err != nil {
		return nil, err
	}
	userId, assistantId, err := l.svcCtx.AiConversationModel.AppendTurn(ctx, conversationId, in.Actor.UserId, content, answer, trace, tokens)
	if err != nil {
		return nil, aiModelError(err)
	}
	now := time.Now().Unix()
	return &aipb.SendMessageResponse{ConversationId: conversationId, UserMessage: &aipb.ChatMessage{Id: userId, Role: "user", Content: content, CreatedAt: now}, AiMessage: &aipb.ChatMessage{Id: assistantId, Role: "assistant", Content: answer, ToolCalls: trace, Tokens: tokens, CreatedAt: now}}, nil
}

func askModel(ctx context.Context, client *llm.Client, messages []*schema.Message) (string, string, int64, error) {
	var calls []schema.ToolCall
	var tokens int64
	for round := 0; round < 5; round++ {
		response, usage, err := client.Chat(ctx, messages, agenttools.LLMTools())
		if err != nil {
			return "", "", 0, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "模型调用失败，请稍后重试")
		}
		if usage != nil && usage.TotalTokens > 0 {
			tokens += int64(usage.TotalTokens)
		}
		if tokens > 10_000_000 {
			return "", "", 0, xerr.NewErrCodeMsg(500, "模型用量响应超出限制")
		}
		if len(response.ToolCalls) == 0 {
			if strings.TrimSpace(response.Content) == "" || len(response.Content) > 60000 {
				return "", "", 0, xerr.NewErrCodeMsg(500, "模型回复为空或超过大小限制")
			}
			if calls == nil {
				calls = make([]schema.ToolCall, 0)
			}
			trace, err := json.Marshal(calls)
			if err != nil || len(trace) > 60000 {
				return "", "", 0, xerr.NewErrCodeMsg(500, "工具调用记录超过大小限制")
			}
			return response.Content, string(trace), tokens, nil
		}
		if len(response.ToolCalls) > 16 {
			return "", "", 0, xerr.NewErrCodeMsg(500, "模型请求过多工具调用")
		}
		messages = append(messages, response)
		for _, call := range response.ToolCalls {
			if len(call.Function.Arguments) > 64000 {
				return "", "", 0, xerr.NewErrCodeMsg(500, "工具参数超过大小限制")
			}
			calls = append(calls, call)
			result := "工具不存在"
			if tool, ok := agenttools.Get(call.Function.Name); ok {
				if json.Valid([]byte(call.Function.Arguments)) {
					value, err := tool.Execute(ctx, json.RawMessage(call.Function.Arguments))
					if err != nil {
						result = "工具参数不合法或执行失败"
					} else {
						result = value
					}
				} else {
					result = "工具参数不是有效 JSON"
				}
			}
			if len(result) > 16000 {
				runes := []rune(result)
				result = string(runes[:min(len(runes), 4000)])
			}
			messages = append(messages, schema.ToolMessage(result, call.ID))
		}
	}
	return "", "", 0, xerr.NewErrCodeMsg(500, "已达到工具调用轮数限制，请简化问题")
}
