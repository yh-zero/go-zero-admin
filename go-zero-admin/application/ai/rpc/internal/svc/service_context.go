package svc

import (
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
	LLM    *llm.Client // AI.Enabled=false 时为 nil

	AiConversationModel model.AiConversationModel
	AiMessageModel      model.AiMessageModel
	AiWorkflowModel     model.AiWorkflowModel
	AiTaskModel         model.AiTaskModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.Mysql.DataSource)
	rdb := redis.MustNewRedis(c.CacheRedis[0].RedisConf)

	// 密钥经环境变量注入（compose environment 或 shell），yaml 留空
	var llmClient *llm.Client
	if c.AI.Enabled {
		apiKey := c.AI.DeepSeekKey
		if c.AI.Provider == "qwen" {
			apiKey = c.AI.QwenKey
		}
		if apiKey != "" {
			var err error
			llmClient, err = llm.New(llm.Config{
				BaseURL: c.AI.BaseURL,
				APIKey:  apiKey,
				Model:   c.AI.Model,
				Timeout: c.AI.TimeoutSeconds,
			})
			if err != nil {
				panic("init llm client failed: " + err.Error())
			}
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
