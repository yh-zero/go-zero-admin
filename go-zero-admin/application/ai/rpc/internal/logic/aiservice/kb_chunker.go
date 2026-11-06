package aiservicelogic

import (
	"strings"

	"go-zero-admin/application/ai/rpc/internal/model"
)

// 分块纪律：目标约 500 token + 10% 重叠；token 按 rune 数保守估计（上界），
// 检索预算因此不会超支。段落优先，超长段落硬切。
const (
	kbChunkTargetTokens  = 500
	kbChunkOverlapTokens = 50
)

// chunkText 将文档正文切分为分块行（DocId/KbId 由调用方填充）
func chunkText(content string) []model.KbChunkRow {
	chunks := make([]model.KbChunkRow, 0)
	var current []rune

	flush := func() {
		trimmed := strings.TrimSpace(string(current))
		current = nil
		if trimmed == "" {
			return
		}
		chunks = append(chunks, model.KbChunkRow{
			Seq:        int64(len(chunks)),
			Content:    trimmed,
			TokenCount: int64(len([]rune(trimmed))),
		})
	}
	// 满块收尾：flush 后从块尾取 overlap 作为下一块开头
	flushWithOverlap := func() {
		flush()
		if len(chunks) == 0 {
			return
		}
		lastRunes := []rune(chunks[len(chunks)-1].Content)
		start := len(lastRunes) - kbChunkOverlapTokens
		if start < 0 {
			start = 0
		}
		current = append([]rune{}, lastRunes[start:]...)
	}

	for _, p := range strings.Split(content, "\n\n") {
		rs := []rune(strings.TrimSpace(p))
		for len(rs) > 0 {
			room := kbChunkTargetTokens - len(current)
			if room <= 0 {
				flushWithOverlap()
				room = kbChunkTargetTokens - len(current)
			}
			take := min(room, len(rs))
			current = append(current, rs[:take]...)
			rs = rs[take:]
			if len(current) >= kbChunkTargetTokens {
				flushWithOverlap()
			}
		}
		if len(current) > 0 {
			current = append(current, '\n', '\n')
		}
	}
	flush()
	return chunks
}
