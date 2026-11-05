package aiservicelogic

import (
	"encoding/json"

	"github.com/cloudwego/eino/schema"
	"go-zero-admin/application/ai/rpc/internal/svc"
)

// contextBudget 控制多轮对话历史的最大 prompt token 预算。
// 历史按时间正序从新到旧累积，直到逼近预算上限；最新的消息始终保留。
type contextBudget struct {
	maxTokens int64
	usedBytes int64
}

func newContextBudget(svcCtx *svc.ServiceContext) *contextBudget {
	max := svcCtx.Config.AI.MaxContextTokens
	if max < 1000 {
		max = 32000
	}
	return &contextBudget{maxTokens: max}
}

// 裁剪与逐轮检查使用同一序列化字节口径；这是估算预算，不是供应商计费量。
func (b *contextBudget) tryAdd(content string) bool {
	bytes := b.usedBytes + int64(len(content))
	if (bytes+2)/3 > b.maxTokens {
		return false
	}
	b.usedBytes = bytes
	return true
}

func (b *contextBudget) tryAddMessage(message *schema.Message) bool {
	data, err := json.Marshal(message)
	// 每条消息保留一个数组分隔字符；首条前的 '[' 由调用方预留。
	return err == nil && b.tryAdd(string(data)+",")
}
