package aiservicelogic

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/internal/model"
)

func isGroundedFollowup(question string) bool {
	for _, marker := range []string{"这个", "那个", "这些", "那些", "上述", "前面", "上面", "刚才", "上一步", "上一", "前者", "后者", "其中", "它", "继续", "再详细", "第一个", "第二种", "第二个", "第二条", "第二步", "第三种", "第三个", "第三条", "第三步", "第一种", "第一条"} {
		if strings.Contains(question, marker) {
			return true
		}
	}
	words := " " + strings.ToLower(strings.Trim(strings.TrimSpace(question), ".?!")) + " "
	for _, marker := range []string{" it ", " that ", " those ", " them ", " the second ", " the first ", " the former ", " the latter ", " tell me more ", " continue "} {
		if strings.Contains(words, marker) {
			return true
		}
	}
	return false
}

// 只接续同一知识库最近一次有真实来源的完整问答。旧记录可通过原来源确认归属。
func groundedReference(rows []*model.MessageRow, kbID, docID int64, question string) string {
	if len(rows) != 2 {
		return ""
	}
	user, answer := rows[0], rows[1]
	if user.Role != "user" || answer.Role != "assistant" || user.CallId == "" || user.CallId != answer.CallId {
		return ""
	}
	var record callRecord
	if json.Unmarshal([]byte(answer.ToolCalls), &record) != nil || record.Version != 1 || len(record.Sources) == 0 || record.KbID != 0 && record.KbID != kbID || docID > 0 && record.DocID != docID {
		return ""
	}
	for _, hit := range record.Sources {
		if hit.KbID != kbID || docID > 0 && hit.DocID != docID {
			return ""
		}
	}
	const prevQ, prevA = "\n上轮问题：", "\n上轮回答："
	room := 2000 - len(question) - len(prevQ) - len(prevA)
	if room < 16 {
		return ""
	}
	prefix := func(s string, n int) string {
		if len(s) <= n {
			return s
		}
		for n > 0 && !utf8.RuneStart(s[n]) {
			n--
		}
		return s[:n]
	}
	q := prefix(user.Content, min(320, room/2))
	a := prefix(answer.Content, min(640, room-len(q)))
	return prevQ + q + prevA + a
}
