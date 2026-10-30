package aiservicelogic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
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
	// AI 未配置时友好提示
	if l.svcCtx.LLM == nil {
		return nil, xerr.NewErrCodeMsg(400, "AI 助手未启用：请在 etc/ai.yaml 配置 AI.Enabled 并填密钥")
	}

	// 会话处理：传 0 自动创建
	convID := in.ConversationId
	if convID == 0 {
		now := time.Now()
		res, err := l.svcCtx.DB.ExecCtx(l.ctx,
			"INSERT INTO ai_conversation (user_id, title, created_at, updated_at) VALUES (?,?,?,?)",
			in.Actor.UserId, in.Content, now, now)
		if err != nil {
			return nil, xerr.NewErrCode(xerr.DB_ERROR)
		}
		convID, _ = res.LastInsertId()
	} else {
		var exists int
		_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &exists,
			"SELECT COUNT(*) FROM ai_conversation WHERE id=? AND user_id=? AND deleted_at IS NULL", convID, in.Actor.UserId)
		if exists == 0 {
			return nil, xerr.NewErrCodeMsg(404, "会话不存在")
		}
	}

	// 保存用户消息
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx,
		"INSERT INTO ai_message (conversation_id, role, content) VALUES (?,?,?)", convID, "user", in.Content)

	// 组装历史（最近 20 条）
	type msgRow struct {
		Role      string `db:"role"`
		Content   string `db:"content"`
		ToolCalls string `db:"tool_calls"`
	}
	var histRows []msgRow
	_ = l.svcCtx.DB.QueryRowsCtx(l.ctx, &histRows,
		"SELECT role, content, COALESCE(CAST(tool_calls AS CHAR),'') FROM ai_message WHERE conversation_id=? ORDER BY id DESC LIMIT 20", convID)
	var messages []*schema.Message
	for i := len(histRows) - 1; i >= 0; i-- {
		r := histRows[i]
		if r.Role == "user" {
			messages = append(messages, llm.UserMessage(r.Content))
		} else if r.Role == "assistant" {
			messages = append(messages, schema.AssistantMessage(r.Content, nil))
		}
	}

	// 多轮 function calling（最多 5 轮）
	const maxRounds = 5
	var toolCallsJSON []byte
	totalTokens := 0
	finalMsg := ""

	for round := 0; round < maxRounds; round++ {
		resp, usage, err := l.svcCtx.LLM.Chat(l.ctx, messages, agenttools.LLMTools())
		if err != nil {
			l.Errorf("llm chat round %d: %v", round, err)
			return nil, xerr.NewErrCodeMsg(500, "AI 调用失败："+err.Error())
		}
		if usage != nil {
			totalTokens += usage.TotalTokens
		}

		// 终态：无工具调用
		if len(resp.ToolCalls) == 0 {
			finalMsg = resp.Content
			break
		}

		// 执行工具并回填
		toolCallsJSON, _ = json.Marshal(resp.ToolCalls)
		messages = append(messages, resp)
		for _, tc := range resp.ToolCalls {
			tool, ok := agenttools.Get(tc.Function.Name)
			var result string
			if !ok {
				result = "错误：未注册的工具 " + tc.Function.Name
			} else {
				out, err := tool.Execute(l.ctx, json.RawMessage(tc.Function.Arguments))
				if err != nil {
					result = "工具执行错误：" + err.Error()
				} else {
					result = out
				}
			}
			messages = append(messages, schema.ToolMessage(result, tc.ID))
		}
	}

	if finalMsg == "" {
		finalMsg = "（达到最大工具调用轮数，请简化问题）"
	}

	// 保存 AI 回复
	res, _ := l.svcCtx.DB.ExecCtx(l.ctx,
		"INSERT INTO ai_message (conversation_id, role, content, tool_calls, tokens) VALUES (?,?,?,?,?)",
		convID, "assistant", finalMsg, string(toolCallsJSON), totalTokens)
	aiMsgID, _ := res.LastInsertId()
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE ai_conversation SET updated_at=NOW() WHERE id=?", convID)

	return &aipb.SendMessageResponse{
		UserMessage: &aipb.ChatMessage{Role: "user", Content: in.Content},
		AiMessage:   &aipb.ChatMessage{Id: aiMsgID, Role: "assistant", Content: finalMsg, ToolCalls: string(toolCallsJSON), Tokens: int64(totalTokens)},
	}, nil
}
