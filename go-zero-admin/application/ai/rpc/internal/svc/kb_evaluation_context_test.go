package svc

import (
	"encoding/json"
	"strings"
	"testing"

	"go-zero-admin/application/ai/rpc/internal/config"
)

func TestEvaluationContextReflectsEffectiveBudgetWithoutSecrets(t *testing.T) {
	c := config.ConfigAI{Provider: "deepseek", Model: "deepseek-chat", BaseURL: "https://api.deepseek.com/v1",
		EmbeddingProvider: "qwen", EmbeddingModel: "text-embedding-v4", EmbeddingBaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		EmbeddingDimensions: 1024, MaxContextTokens: 32000, TimeoutSeconds: 60, DeepSeekKey: "PRIVATE_CHAT_MARKER", QwenKey: "PRIVATE_EMBED_MARKER"}
	for _, budget := range []int64{32000, 1600} {
		c.MaxContextTokens = budget
		encoded, err := evaluationContextForBuild(c, strings.Repeat("a", 64))
		if err != nil {
			t.Fatal(err)
		}
		var value struct {
			BinarySHA string `json:"binary_sha"`
			Model     struct {
				Provider, Name string
				BaseURL        string `json:"base_url"`
			} `json:"model"`
			Config struct {
				Context int64 `json:"max_context_tokens"`
				Answer  int64 `json:"analysis_answer_token_cap"`
				Recall  int   `json:"recall_top_k"`
				Return  int   `json:"return_top_k"`
			} `json:"config"`
		}
		if json.Unmarshal([]byte(encoded), &value) != nil || value.BinarySHA != strings.Repeat("a", 64) || value.Model.Name != c.Model || value.Model.BaseURL != c.BaseURL || value.Config.Context != budget || value.Config.Answer != min(int64(2048), budget/4) || value.Config.Recall != 20 || value.Config.Return != 5 {
			t.Fatalf("实际模型和预算元数据错误：%s", encoded)
		}
		if strings.Contains(encoded, "PRIVATE_") || strings.Contains(encoded, "DeepSeekKey") || strings.Contains(encoded, "QwenKey") {
			t.Fatal("元数据泄漏了配置凭据")
		}
	}
	c.BaseURL = "https://user:password@example.test/v1"
	if _, err := evaluationContextForBuild(c, strings.Repeat("a", 64)); err == nil {
		t.Fatal("含凭据的端点不能进入可见元数据")
	}
}
