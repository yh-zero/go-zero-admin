package aiservicelogic

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/schema"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
)

// 调用元数据只在存储层保存；对外工具记录仍为数组字符串。
type callRecord struct {
	Version       int               `json:"version"`
	Calls         []schema.ToolCall `json:"calls"`
	Fingerprint   string            `json:"fingerprint"`
	Error         string            `json:"error,omitempty"`
	ErrorCode     uint32            `json:"error_code,omitempty"`
	UsageComplete bool              `json:"usage_complete"`
}

func (r callRecord) marshal() string { data, _ := json.Marshal(r); return string(data) }

func requestFingerprint(conversationId int64, content string) string {
	data, _ := json.Marshal([]any{conversationId, content})
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func callResponse(pair *model.CallPair) *aipb.SendMessageResponse {
	response := &aipb.SendMessageResponse{ConversationId: pair.ConversationId, UserMessage: messageProto(pair.User), AiMessage: messageProto(pair.Assistant)}
	if pair.Conversation != nil {
		response.Conversation = conversationProto(pair.Conversation)
	}
	return response
}
