package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	Mysql      struct{ DataSource string }
	CacheRedis cache.CacheConf

	Qdrant ConfigQdrant
	AI     ConfigAI
}

// ConfigQdrant 知识库向量索引；Address 去空白后非空即启用（无独立开关），未配置则知识库摄取/检索降级。
type ConfigQdrant struct {
	Address string `json:",optional"`
	APIKey  string `json:",optional"`
}

// ConfigAI AI 模型配置；密钥支持直填或 ${AI_DEEPSEEK_KEY}/${AI_QWEN_KEY} 占位符（conf.UseEnv 展开）。
type ConfigAI struct {
	Enabled          bool   `json:",default=false"` // 兼容输入；svc 按 Provider 对应去空白后的密钥计算并回写实际状态。
	Provider         string `json:",optional"`
	Model            string `json:",optional"`
	BaseURL          string `json:",optional"`
	TimeoutSeconds   int64  `json:",default=60"`
	MaxContextTokens int64  `json:",default=32000"` // 多轮对话历史的最大 prompt token 预算
	DeepSeekKey      string `json:",optional"`
	QwenKey          string `json:",optional"`

	// Embedding（知识库向量化）：固定走 EmbeddingProvider（当前仅 qwen），密钥取 QwenKey；与聊天 Provider 独立。
	EmbeddingProvider   string `json:",optional"`
	EmbeddingModel      string `json:",optional"`
	EmbeddingDimensions int64  `json:",default=1024"`
	EmbeddingBaseURL    string `json:",optional"`
}
