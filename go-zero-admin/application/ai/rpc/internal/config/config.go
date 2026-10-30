package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	Mysql      struct{ DataSource string }
	CacheRedis cache.CacheConf

	AI struct {
		Enabled        bool   `json:",default=false"`
		Provider       string `json:",optional"`
		Model          string `json:",optional"`
		BaseURL        string `json:",optional"`
		TimeoutSeconds int64  `json:",default=60"`
		// 模型密钥：yaml 留空，经环境变量 API_KEY_DEEPSEEK / API_KEY_QWEN 注入（compose 或 shell）
		DeepSeekKey string `json:",optional"`
		QwenKey     string `json:",optional"`
	}
}
