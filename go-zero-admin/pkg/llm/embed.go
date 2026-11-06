// Embedding 封装（OpenAI 兼容 /embeddings 协议，DashScope 兼容模式等厂商通用）
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Embedder 文本向量化客户端
type Embedder struct {
	baseURL string
	apiKey  string
	model   string
	dims    int
	client  *http.Client
}

// EmbedConfig Embedding 配置
type EmbedConfig struct {
	BaseURL string // 如 https://dashscope.aliyuncs.com/compatible-mode/v1
	APIKey  string
	Model   string // 如 text-embedding-v4
	Dims    int    // 向量维度，响应会校验一致
	Timeout int64  // 秒
}

// embedBatchSize DashScope 等厂商单次批量上限为 25 条，统一按此分批
const embedBatchSize = 25

// NewEmbedder 创建 Embedding 客户端；配置无效返回错误，由调用方决定降级
func NewEmbedder(cfg EmbedConfig) (*Embedder, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" || strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.Model) == "" || cfg.Dims <= 0 {
		return nil, errors.New("embedding 配置不完整")
	}
	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &Embedder{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:  cfg.APIKey,
		model:   cfg.Model,
		dims:    cfg.Dims,
		client:  &http.Client{Timeout: timeout},
	}, nil
}

// Dims 返回配置的向量维度
func (e *Embedder) Dims() int { return e.dims }

// Embed 批量向量化，按厂商上限分批；任一失败整体失败（摄取方可整体重试）
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	out := make([][]float32, 0, len(texts))
	for start := 0; start < len(texts); start += embedBatchSize {
		end := start + embedBatchSize
		if end > len(texts) {
			end = len(texts)
		}
		vecs, err := e.embedBatch(ctx, texts[start:end])
		if err != nil {
			return nil, err
		}
		out = append(out, vecs...)
	}
	return out, nil
}

type embedRequest struct {
	Model      string   `json:"model"`
	Input      []string `json:"input"`
	Dimensions int      `json:"dimensions,omitempty"`
}

type embedResponse struct {
	Data []struct {
		Index     int       `json:"index"`
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (e *Embedder) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	body, err := json.Marshal(embedRequest{Model: e.model, Input: texts, Dimensions: e.dims})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		// 只保留状态码，不回显厂商响应体（可能含敏感信息）
		return nil, fmt.Errorf("embedding 服务返回 HTTP %d", resp.StatusCode)
	}
	var parsed embedResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("embedding 响应解析失败: %w", err)
	}
	if parsed.Error != nil {
		return nil, errors.New("embedding 服务返回业务错误")
	}
	if len(parsed.Data) != len(texts) {
		return nil, fmt.Errorf("embedding 返回数量 %d 与请求 %d 不一致", len(parsed.Data), len(texts))
	}
	vecs := make([][]float32, len(texts))
	for _, d := range parsed.Data {
		if d.Index < 0 || d.Index >= len(texts) {
			return nil, fmt.Errorf("embedding 响应索引越界: %d", d.Index)
		}
		if len(d.Embedding) != e.dims {
			return nil, fmt.Errorf("embedding 维度 %d 与配置 %d 不一致", len(d.Embedding), e.dims)
		}
		vecs[d.Index] = d.Embedding
	}
	return vecs, nil
}
