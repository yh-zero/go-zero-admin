// Package llm LLM 客户端封装（eino + OpenAI 兼容协议）
package llm

import (
	"context"
	"errors"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Client LLM 客户端（eino ChatModel 封装）
type Client struct {
	model   model.ChatModel
	timeout time.Duration
}

// Config LLM 配置
type Config struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout int64 // 秒
}

// New 创建 LLM 客户端（eino openai 适配器，DeepSeek/通义/任何 OpenAI 兼容厂商）
func New(cfg Config) (*Client, error) {
	m, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		BaseURL: cfg.BaseURL,
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
	})
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{model: m, timeout: timeout}, nil
}

// Chat 对话（eino 统一接口，自动处理 function calling 协议）
func (c *Client) Chat(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.Message, *schema.TokenUsage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	opts := []model.Option{}
	if len(tools) > 0 {
		opts = append(opts, model.WithTools(tools))
	}
	resp, err := c.model.Generate(ctx, messages, opts...)
	if err != nil {
		return nil, nil, err
	}
	if resp == nil {
		return nil, nil, errors.New("模型响应为空")
	}
	if resp.ResponseMeta == nil {
		return resp, nil, nil
	}
	return resp, resp.ResponseMeta.Usage, nil
}

// ChatStream 流式对话（eino StreamReader，Agent 标配体验）
func (c *Client) ChatStream(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.StreamReader[*schema.Message], error) {
	opts := []model.Option{}
	if len(tools) > 0 {
		opts = append(opts, model.WithTools(tools))
	}
	return c.model.Stream(ctx, messages, opts...)
}

// ToolInfo 便捷构造函数
func ToolInfo(name, description string, params map[string]*schema.ParameterInfo) *schema.ToolInfo {
	return &schema.ToolInfo{
		Name:        name,
		Desc:        description,
		ParamsOneOf: schema.NewParamsOneOfByParams(params),
	}
}

// UserMessage 便捷构造用户消息
func UserMessage(content string) *schema.Message {
	return schema.UserMessage(content)
}

// SystemMessage 便捷构造系统消息
func SystemMessage(content string) *schema.Message {
	return schema.SystemMessage(content)
}
