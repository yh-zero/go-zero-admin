package svc

import (
	"net/url"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"go-zero-admin/application/ai/rpc/internal/config"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/llm"
)

type ServiceContext struct {
	Config config.Config
	DB     sqlx.SqlConn
	Redis  *redis.Redis
	LLM    *llm.Client // 当前服务商未配置 Key 时为 nil

	AiConversationModel model.AiConversationModel
	AiMessageModel      model.AiMessageModel
	AiWorkflowModel     model.AiWorkflowModel
	AiTaskModel         model.AiTaskModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.Mysql.DataSource)
	rdb := redis.MustNewRedis(c.CacheRedis[0].RedisConf)

	ai := c.AI
	ai.Enabled = newLLMEnabled(ai)
	llmClient := newLLMClient(ai)

	return &ServiceContext{
		Config:              c,
		DB:                  conn,
		Redis:               rdb,
		LLM:                 llmClient,
		AiConversationModel: model.NewAiConversationModel(conn, c.CacheRedis),
		AiMessageModel:      model.NewAiMessageModel(conn, c.CacheRedis),
		AiWorkflowModel:     model.NewAiWorkflowModel(conn, c.CacheRedis),
		AiTaskModel:         model.NewAiTaskModel(conn, c.CacheRedis),
	}
}

// newLLMEnabled 按 Provider 取到密钥即视为启用；密钥支持 yaml 直填或环境变量占位符（conf.UseEnv 加载时已展开）。
func newLLMEnabled(ai config.ConfigAI) bool {
	provider := strings.ToUpper(strings.TrimSpace(ai.Provider))
	switch provider {
	case "DEEPSEEK":
		return strings.TrimSpace(ai.DeepSeekKey) != ""
	case "QWEN":
		return strings.TrimSpace(ai.QwenKey) != ""
	}
	return false
}

// newLLMClient 校验配置并组装 LLM client；未启用或配置无效时返回 nil，由 AI 接口返回降级提示。
func newLLMClient(ai config.ConfigAI) *llm.Client {
	if !ai.Enabled {
		return nil
	}
	endpoint, err := url.Parse(ai.BaseURL)
	if err != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || strings.TrimSpace(ai.Model) == "" {
		panic("AI 模型或地址配置无效")
	}
	if ai.TimeoutSeconds < 1 || ai.TimeoutSeconds > 90 {
		panic("AI.TimeoutSeconds 必须为 1..90")
	}
	apiKey := strings.TrimSpace(ai.DeepSeekKey)
	if strings.ToUpper(strings.TrimSpace(ai.Provider)) == "QWEN" {
		apiKey = strings.TrimSpace(ai.QwenKey)
	}
	client, err := llm.New(llm.Config{
		BaseURL: ai.BaseURL,
		APIKey:  apiKey,
		Model:   ai.Model,
		Timeout: ai.TimeoutSeconds,
	})
	if err != nil {
		panic("AI 客户端初始化失败，请检查配置")
	}
	return client
}
