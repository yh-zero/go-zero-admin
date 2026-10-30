// Package agenttools Agent 工具注册表（eino 适配）
// 新增工具：实现 Tool 接口并在 init() 注册。工具在服务端执行，LLM 只产出调用意图。
package agenttools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

// Tool Agent 可调用工具
type Tool interface {
	Name() string
	Description() string
	Params() map[string]*schema.ParameterInfo
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

var registry = map[string]Tool{}

// Register 注册工具（重复注册 panic）
func Register(t Tool) {
	if _, exists := registry[t.Name()]; exists {
		panic("tool already registered: " + t.Name())
	}
	registry[t.Name()] = t
}

// Get 按名取工具
func Get(name string) (Tool, bool) {
	t, ok := registry[name]
	return t, ok
}

// LLMTools 生成 eino function calling 工具定义列表
func LLMTools() []*schema.ToolInfo {
	out := make([]*schema.ToolInfo, 0, len(registry))
	for _, t := range registry {
		out = append(out, &schema.ToolInfo{
			Name:        t.Name(),
			Desc:        t.Description(),
			ParamsOneOf: schema.NewParamsOneOfByParams(t.Params()),
		})
	}
	return out
}

// List 注册表清单
func List() []Tool {
	out := make([]Tool, 0, len(registry))
	for _, t := range registry {
		out = append(out, t)
	}
	return out
}

// ========== 内置工具 ==========

// getCurrentTime 当前时间工具
type getCurrentTime struct{}

func (getCurrentTime) Name() string        { return "get_current_time" }
func (getCurrentTime) Description() string { return "获取当前服务器时间（Asia/Shanghai）" }
func (getCurrentTime) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{}
}
func (getCurrentTime) Execute(context.Context, json.RawMessage) (string, error) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	return time.Now().In(loc).Format("2006-01-02 15:04:05 MST"), nil
}

// textStats 文本统计工具（演示带参数工具）
type textStats struct{}

func (textStats) Name() string        { return "text_stats" }
func (textStats) Description() string { return "统计文本的字符数、单词数、行数" }
func (textStats) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"text": {Type: schema.String, Desc: "要统计的文本", Required: true},
	}
}
func (textStats) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	chars := len([]rune(in.Text))
	words := len(strings.Fields(in.Text))
	lines := len(strings.Split(in.Text, "\n"))
	return fmt.Sprintf("字符数 %d，单词数 %d，行数 %d", chars, words, lines), nil
}

func init() {
	Register(getCurrentTime{})
	Register(textStats{})
}
