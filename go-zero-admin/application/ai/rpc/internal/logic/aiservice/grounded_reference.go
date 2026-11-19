package aiservicelogic

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/internal/model"
)

// 英文按词匹配，保留 it 与行业缩写 IT 的大小写区别，标点也可以结束指代词。
var englishFollowup = regexp.MustCompile(`\b(?:[Ii]t|(?i:that|those|them|the\s+(?:second|first|former|latter)|tell\s+me\s+more|continue))\b`)

func isGroundedFollowup(question string) bool {
	command := strings.Trim(strings.TrimSpace(question), "。！？，.!?,")
	switch strings.TrimSuffix(strings.TrimPrefix(command, "请"), "吧") {
	case "继续", "继续回答":
		return true
	}
	for _, marker := range []string{"这个", "那个", "这些", "那些", "上述", "前面", "上面", "刚才", "上一步", "上一", "前者", "后者", "其中", "它", "继续说", "继续讲", "继续解释", "继续展开", "再详细", "第一个", "第二种", "第二个", "第二条", "第二步", "第三种", "第三个", "第三条", "第三步", "第一种", "第一条"} {
		if strings.Contains(question, marker) {
			return true
		}
	}
	return englishFollowup.MatchString(question)
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
