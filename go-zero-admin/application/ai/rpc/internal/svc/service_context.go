package svc

import (
	"net/url"
	"os"
	"regexp"
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

	// 仅从约定环境变量读取密钥；不从 YAML 回退，避免误将凭据提交仓库。
	var llmClient *llm.Client
	provider := strings.ToUpper(strings.TrimSpace(c.AI.Provider))
	apiKey := strings.TrimSpace(os.Getenv("AI_" + provider + "_KEY"))
	c.AI.Enabled = apiKey != ""
	if c.AI.Enabled {
		if !regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`).MatchString(provider) {
			panic("AI.Provider 配置无效")
		}
		endpoint, err := url.Parse(c.AI.BaseURL)
		if err != nil || endpoint.Hostname() == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || strings.TrimSpace(c.AI.Model) == "" {
			panic("AI 模型或地址配置无效")
		}
		if c.AI.TimeoutSeconds < 1 || c.AI.TimeoutSeconds > 90 {
			panic("AI.TimeoutSeconds 必须为 1..90")
		}
		llmClient, err = llm.New(llm.Config{
			BaseURL: c.AI.BaseURL,
			APIKey:  apiKey,
			Model:   c.AI.Model,
			Timeout: c.AI.TimeoutSeconds,
		})
		if err != nil {
			panic("AI 客户端初始化失败，请检查配置")
		}
	}

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
