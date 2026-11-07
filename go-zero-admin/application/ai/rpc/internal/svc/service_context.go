package svc

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"go-zero-admin/application/ai/rpc/internal/config"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/vectordb"
)

type ServiceContext struct {
	Config config.Config
	DB     sqlx.SqlConn
	Redis  *redis.Redis
	LLM    *llm.Client // 当前服务商未配置 Key 时为 nil

	// 知识库检索：Qdrant.Address 或 Embedding 未配置时为 nil，对应功能按契约降级
	VectorStore vectordb.Store
	Embedder    *llm.Embedder
	KbSearcher  agenttools.KbSearcher

	AiConversationModel model.AiConversationModel
	AiMessageModel      model.AiMessageModel
	AiWorkflowModel     model.AiWorkflowModel
	AiTaskModel         model.AiTaskModel
	AiKbModel           model.AiKbModel
	AiKbDocModel        model.AiKbDocModel
	AiKbChunkModel      model.AiKbChunkModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.Mysql.DataSource)
	rdb := redis.MustNewRedis(c.CacheRedis[0].RedisConf)

	ai := c.AI
	ai.Enabled = newLLMEnabled(ai)
	c.AI = ai
	llmClient := newLLMClient(ai)

	svcCtx := &ServiceContext{
		Config:              c,
		DB:                  conn,
		Redis:               rdb,
		LLM:                 llmClient,
		AiConversationModel: model.NewAiConversationModel(conn, c.CacheRedis),
		AiMessageModel:      model.NewAiMessageModel(conn, c.CacheRedis),
		AiWorkflowModel:     model.NewAiWorkflowModel(conn, c.CacheRedis),
		AiTaskModel:         model.NewAiTaskModel(conn, c.CacheRedis),
		AiKbModel:           model.NewAiKbModel(conn, c.CacheRedis),
		AiKbDocModel:        model.NewAiKbDocModel(conn, c.CacheRedis),
		AiKbChunkModel:      model.NewAiKbChunkModel(conn, c.CacheRedis),
	}
	svcCtx.Embedder = newEmbedder(ai)
	svcCtx.VectorStore = newVectorStore(c)
	if svcCtx.VectorStore != nil && svcCtx.Embedder != nil {
		svcCtx.KbSearcher = &kbSearcher{
			store:         svcCtx.VectorStore,
			embedder:      svcCtx.Embedder,
			chunks:        svcCtx.AiKbChunkModel,
			snippetBudget: ai.MaxContextTokens / 4,
		}
		agenttools.RegisterIfAbsent(agenttools.NewKbSearch(svcCtx.KbSearcher))
	}
	if analyzer := newKbAnalyzer(svcCtx); analyzer != nil {
		agenttools.RegisterIfAbsent(analyzer)
	}
	return svcCtx
}

// newEmbedder 组装知识库 Embedding 客户端；固定走 EmbeddingProvider（当前仅 qwen，密钥取 QwenKey），未配置返回 nil。
func newEmbedder(ai config.ConfigAI) *llm.Embedder {
	if !strings.EqualFold(strings.TrimSpace(ai.EmbeddingProvider), "qwen") {
		return nil
	}
	key := strings.TrimSpace(ai.QwenKey)
	if key == "" {
		return nil
	}
	embedder, err := llm.NewEmbedder(llm.EmbedConfig{
		BaseURL: ai.EmbeddingBaseURL,
		APIKey:  key,
		Model:   ai.EmbeddingModel,
		Dims:    int(ai.EmbeddingDimensions),
		Timeout: ai.TimeoutSeconds,
	})
	if err != nil {
		logx.Errorf("Embedding 配置无效，知识库检索降级: %v", err)
		return nil
	}
	return embedder
}

// newVectorStore 组装 Qdrant 向量索引；Address 去空白后非空即启用，连接/建集合失败降级为 nil（不 panic）。
func newVectorStore(c config.Config) vectordb.Store {
	address := strings.TrimSpace(c.Qdrant.Address)
	if address == "" {
		return nil
	}
	collection := vectordb.CollectionName(c.AI.EmbeddingProvider, c.AI.EmbeddingModel, c.AI.EmbeddingDimensions)
	store, err := vectordb.NewQdrantStore(address, c.Qdrant.APIKey, collection, c.AI.EmbeddingDimensions)
	if err != nil {
		logx.Errorf("Qdrant 初始化失败，知识库检索降级: %v", err)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := store.EnsureCollection(ctx); err != nil {
		logx.Errorf("Qdrant 集合初始化失败，知识库检索降级: %v", err)
		_ = store.Close()
		return nil
	}
	return store
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
