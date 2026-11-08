package agenttools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestKbAnalyzeFailureKeepsModelUsageWithoutRejectedReply(t *testing.T) {
	for _, tc := range []struct {
		name    string
		reply   string
		usage   *schema.TokenUsage
		chatErr error
		want    KbAnalysisUsage
	}{
		{name: "非法结构保留已知用量", reply: "PRIVATE_MODEL_BODY", usage: &schema.TokenUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120}, want: KbAnalysisUsage{Scope: "chat", PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120, Complete: true}},
		{name: "超长回复保留已知用量", reply: strings.Repeat("PRIVATE_MODEL_BODY", 800), usage: &schema.TokenUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120}, want: KbAnalysisUsage{Scope: "chat", PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120, Complete: true}},
		{name: "调用失败用量未知", chatErr: errors.New("PRIVATE_VENDOR_ERROR"), want: KbAnalysisUsage{Scope: "chat"}},
		{name: "未给用量保持未知", reply: "PRIVATE_MODEL_BODY", want: KbAnalysisUsage{Scope: "chat"}},
		{name: "分项与总量不一致", reply: "PRIVATE_MODEL_BODY", usage: &schema.TokenUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 125}, want: KbAnalysisUsage{Scope: "chat", PromptTokens: 100, CompletionTokens: 20, TotalTokens: 125}},
		{name: "非法用量不能冒充已知", reply: "PRIVATE_MODEL_BODY", usage: &schema.TokenUsage{PromptTokens: -1, CompletionTokens: 20, TotalTokens: 19}, want: KbAnalysisUsage{Scope: "chat"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			searcher, chat := analysisFixture()
			chat.response.Content, chat.usage, chat.err = tc.reply, tc.usage, tc.chatErr
			out, err := NewKbAnalyze(searcher, func(context.Context, int64, int64) error { return nil }, chat, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`))
			var measured interface{ ModelUsage() *KbAnalysisUsage }
			if err == nil || out != "" || !errors.As(err, &measured) || measured.ModelUsage() == nil || *measured.ModelUsage() != tc.want {
				t.Fatalf("安全失败仍须返回已尝试调用的计量: out=%q err=%v usage=%v", out, err, measured)
			}
			if strings.Contains(err.Error(), "PRIVATE_") || chat.calls != 1 {
				t.Fatal("不得泄露拒绝正文或自动重试模型")
			}
		})
	}
}

func TestKbAnalyzePreCallFailureDoesNotInventModelUsage(t *testing.T) {
	searcher, chat := analysisFixture()
	_, err := NewKbAnalyze(searcher, func(context.Context, int64, int64) error { return errors.New("not owned") }, chat, 32000).Execute(WithActor(context.Background(), 7), []byte(`{"kb_id":3,"query":"目标"}`))
	var measured interface{ ModelUsage() *KbAnalysisUsage }
	if err == nil || chat.calls != 0 || errors.As(err, &measured) && measured.ModelUsage() != nil {
		t.Fatal("模型调用前失败不能填造调用或零用量")
	}
}
