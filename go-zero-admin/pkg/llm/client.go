// Package llm LLM 客户端封装（eino + OpenAI 兼容协议）
package llm

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// Client LLM 客户端（eino ChatModel 封装）
type Client struct {
	model                       model.ChatModel
	timeout                     time.Duration
	hook                        UsageHook
	provider, modelName, apiKey string
}

// Config LLM 配置
type Config struct {
	BaseURL   string
	APIKey    string
	Model     string
	Timeout   int64 // 秒
	Provider  string
	UsageHook UsageHook
}

// New 创建 LLM 客户端（eino openai 适配器，DeepSeek/通义/任何 OpenAI 兼容厂商）
func New(cfg Config) (*Client, error) {
	m, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		BaseURL:    cfg.BaseURL,
		APIKey:     cfg.APIKey,
		Model:      cfg.Model,
		HTTPClient: &http.Client{Transport: &usageTransport{}},
	})
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Client{model: m, timeout: timeout, hook: cfg.UsageHook, provider: cfg.Provider, modelName: cfg.Model, apiKey: cfg.APIKey}, nil
}

// Chat 对话（eino 统一接口，自动处理 function calling 协议）
func (c *Client) Chat(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*schema.Message, *schema.TokenUsage, error) {
	opts := []model.Option{}
	if len(tools) > 0 {
		opts = append(opts, model.WithTools(tools))
	}
	return c.generate(ctx, messages, opts...)
}

// ChatBounded 用于工作流内单次纯文本分析；不提供工具，限制回答 token 并使用零温度减少采样变动。
func (c *Client) ChatBounded(ctx context.Context, messages []*schema.Message, maxTokens int) (*schema.Message, *schema.TokenUsage, error) {
	if maxTokens <= 0 {
		return nil, nil, errors.New("回答预算必须为正数")
	}
	return c.generate(ctx, messages, model.WithMaxTokens(maxTokens), model.WithTemperature(0))
}

func (c *Client) generate(ctx context.Context, messages []*schema.Message, opts ...model.Option) (*schema.Message, *schema.TokenUsage, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	ctx, captured := withUsageCapture(ctx)
	ctx, call, err := beginUsage(ctx, c.hook, c.provider, c.modelName, "chat", 0, 1, c.apiKey)
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.model.Generate(ctx, messages, opts...)
	if finishErr := call.finish(err, false); finishErr != nil {
		usage, _ := captured.Snapshot()
		return nil, usage, finishErr
	}
	if err != nil {
		usage, _ := captured.Snapshot()
		return nil, usage, err
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
func (c *Client) ChatStream(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*MessageStream, error) {
	reader, _, err := c.ChatStreamWithUsage(ctx, messages, tools)
	return reader, err
}

// ChatStreamWithUsage 在SDK缓存或拒绝回复时仍允许编排入口取得已收到的数字用量。
func (c *Client) ChatStreamWithUsage(ctx context.Context, messages []*schema.Message, tools []*schema.ToolInfo) (*MessageStream, *UsageCapture, error) {
	opts := []model.Option{}
	if len(tools) > 0 {
		opts = append(opts, model.WithTools(tools))
	}
	ctx, cancel := streamRequestContext(ctx, c.timeout, c.hook != nil)
	ctx, captured := withUsageCapture(ctx)
	ctx, call, err := beginUsage(ctx, c.hook, c.provider, c.modelName, "chat", 0, 1, c.apiKey)
	if err != nil {
		cancel()
		return nil, captured, err
	}
	call.cancel = streamCancel(call.cancel, cancel)
	reader, err := c.model.Stream(ctx, messages, opts...)
	if err == nil && reader == nil {
		err = errors.New("模型流响应为空")
	}
	if err != nil {
		defer cancel()
		if reader != nil {
			reader.Close()
		}
		if finishErr := call.finish(err, false); finishErr != nil {
			return nil, captured, finishErr
		}
		return nil, captured, err
	}
	return &MessageStream{reader: reader, call: call, cancel: cancel}, captured, nil
}

func streamRequestContext(ctx context.Context, timeout time.Duration, measured bool) (context.Context, context.CancelFunc) {
	if measured {
		return context.WithTimeout(ctx, timeout)
	}
	return context.WithCancel(ctx)
}

func streamCancel(inner, outer context.CancelFunc) context.CancelFunc {
	return func() {
		if inner != nil {
			inner()
		}
		outer()
	}
}

// MessageStream 持有流式调用的取消权；SDK阻塞读取HTTP时Close也能及时释放。
type MessageStream struct {
	reader     *schema.StreamReader[*schema.Message]
	call       *usageCall
	cancel     context.CancelFunc
	finishOnce sync.Once
	closeOnce  sync.Once
	finishErr  error
}

func (s *MessageStream) finish(cause error) error {
	s.finishOnce.Do(func() {
		s.finishErr = s.call.finish(cause, errors.Is(cause, io.EOF))
		s.cancel()
	})
	return s.finishErr
}

func (s *MessageStream) Recv() (*schema.Message, error) {
	message, err := s.reader.Recv()
	if err != nil {
		if finishErr := s.finish(err); finishErr != nil {
			return nil, finishErr
		}
	}
	return message, err
}

func (s *MessageStream) Close() {
	s.closeOnce.Do(func() {
		s.cancel()
		_ = s.finish(context.Canceled)
		s.reader.Close()
	})
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
