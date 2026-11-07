package aiservicelogic

import (
	"encoding/json"

	"github.com/cloudwego/eino/schema"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/xerr"
)

// contextBudget 统一历史、工具与回答的本地估算预算，不代表供应商计费量。
type contextBudget struct {
	maxTokens int64
}

// fit 在初轮为检索(25%)、其他工具(12.5%)及回答(12.5%)留空；后续始终保留回答空间。
// 当前问题与最新完整工具轮次不可丢弃。超大工具正文先结构化缩减，不能让历史挤占它们。
func (b *contextBudget) fit(messages []*schema.Message, definitions []byte, initial bool) ([]*schema.Message, error) {
	current := -1
	for i, message := range messages {
		if message.Role == schema.User {
			current = i
		}
	}
	fail := func() ([]*schema.Message, error) {
		return nil, xerr.NewErrCodeMsg(400, "当前消息、工具定义或必要工具调用超过上下文预算，请缩短消息")
	}
	if current < 0 {
		return fail()
	}
	// 后续压缩只修改本次模型输入，避免改变调用方保存的消息。
	out := make([]*schema.Message, len(messages))
	for i, message := range messages {
		copied := *message
		out[i] = &copied
	}
	limit := (b.maxTokens - b.maxTokens/8) * 3
	size := func(input []*schema.Message) int64 {
		data, err := json.Marshal(input)
		if err != nil {
			return limit + 1
		}
		return int64(len(definitions) + len(data))
	}
	base := size([]*schema.Message{out[current]})
	if base > limit {
		return fail()
	}
	if initial {
		limit = max(b.maxTokens/2*3, base)
	}
	// 历史按用户轮次删除，工具调用与其所有结果始终作为一组。
	for current > 0 && size(out) > limit {
		next := 1
		for next < current && out[next].Role != schema.User {
			next++
		}
		out = out[next:]
		current -= next
	}
	// 当前问题之后较早的工具轮次也可整组移除，保留最新一组以继续回答。
	for size(out) > limit {
		start := current + 1
		if start >= len(out) {
			break
		}
		end := start + 1
		for end < len(out) && out[end].Role == schema.Tool {
			end++
		}
		if end == len(out) {
			break
		}
		out = append(out[:start], out[end:]...)
	}
	for size(out) > limit {
		changed := false
		for _, message := range out[current+1:] {
			if message.Role != schema.Tool || len(message.Content) <= 2 {
				continue
			}
			message.Content = agenttools.CompactToolResult(message.Content, max(2, len(message.Content)/2))
			changed = true
		}
		if !changed {
			return fail()
		}
	}
	return out, nil
}

func newContextBudget(svcCtx *svc.ServiceContext) *contextBudget {
	max := svcCtx.Config.AI.MaxContextTokens
	if max < 1000 {
		max = 32000
	}
	return &contextBudget{maxTokens: max}
}
