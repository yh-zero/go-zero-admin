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
		// 兼容旧 YAML 字段但不读取其值；密钥统一使用 API_KEY_<PROVIDER> 环境变量。
		DeepSeekKey string `json:",optional"`
		QwenKey     string `json:",optional"`
	}
}
