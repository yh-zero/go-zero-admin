package aiservicelogic

import "go-zero-admin/application/ai/rpc/internal/svc"

// contextBudget 控制多轮对话历史的最大 prompt token 预算。
// 历史按时间正序从新到旧累积，直到逼近预算上限；最新的消息始终保留。
type contextBudget struct {
	maxTokens int64
	used      int64
}

func newContextBudget(svcCtx *svc.ServiceContext) *contextBudget {
	max := svcCtx.Config.AI.MaxContextTokens
	if max < 1000 {
		max = 32000
	}
	return &contextBudget{maxTokens: max}
}

// tryAdd 估算单条消息的 token 占用（按字符数 / 4 粗估，中文约 1.5 token/字）。
// 返回 true 表示加入后仍在预算内。
func (b *contextBudget) tryAdd(content string) bool {
	// 粗略估算：英文 ~4 字符/token，中文 ~1.5 字符/token，取保守值 3
	estimated := int64(len(content)) / 3
	if b.used+estimated > b.maxTokens {
		return false
	}
	b.used += estimated
	return true
}
