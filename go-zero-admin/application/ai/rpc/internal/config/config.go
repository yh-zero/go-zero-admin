package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	Mysql      struct{ DataSource string }
	CacheRedis cache.CacheConf

	AI ConfigAI
}

// ConfigAI AI 模型配置；密钥支持直填或 ${AI_DEEPSEEK_KEY}/${AI_QWEN_KEY} 占位符（conf.UseEnv 展开）。
type ConfigAI struct {
	Enabled        bool   `json:",default=false"` // 兼容输入；svc 按 Provider 对应去空白后的密钥计算并回写实际状态。
	Provider       string `json:",optional"`
	Model          string `json:",optional"`
	BaseURL        string `json:",optional"`
	TimeoutSeconds int64  `json:",default=60"`
	DeepSeekKey    string `json:",optional"`
	QwenKey        string `json:",optional"`
}
