package aiservicelogic

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
)

// 调用元数据只在存储层保存；对外工具记录仍为数组字符串。
type callRecord struct {
	Version       int                      `json:"version"`
	Calls         []schema.ToolCall        `json:"calls"`
	Fingerprint   string                   `json:"fingerprint"`
	KbID          int64                    `json:"kb_id,omitempty"`
	DocID         int64                    `json:"doc_id,omitempty"`
	Grounded      *bool                    `json:"grounded,omitempty"`
	Error         string                   `json:"error,omitempty"`
	ErrorCode     uint32                   `json:"error_code,omitempty"`
	UsageComplete bool                     `json:"usage_complete"`
	Sources       []agenttools.KbSearchHit `json:"sources,omitempty"`
	Feedback      string                   `json:"feedback,omitempty"`
	FeedbackNote  string                   `json:"feedback_note,omitempty"`
}

func (r callRecord) marshal() string { data, _ := json.Marshal(r); return string(data) }

func requestFingerprint(conversationId int64, content string, kbID int64, grounded bool, docID int64) string {
	values := []any{conversationId, content}
	// 省略新增字段时保持原字节序列，已完成的请求仍按原指纹重查。
	if kbID != 0 || grounded {
		values = append(values, kbID, grounded)
	}
	if docID > 0 {
		values = append(values, docID)
	}
	data, _ := json.Marshal(values)
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func callResponse(pair *model.CallPair) *aipb.SendMessageResponse {
	response := &aipb.SendMessageResponse{ConversationId: pair.ConversationId, UserMessage: messageProto(pair.User), AiMessage: messageProto(pair.Assistant)}
	if pair.Conversation != nil {
		response.Conversation = conversationProto(pair.Conversation)
	}
	return response
}
