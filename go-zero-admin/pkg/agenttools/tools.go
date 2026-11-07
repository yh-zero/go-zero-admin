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

// IsWorkflowOnly 识别仅允许工作流执行的工具；未声明的既有工具保持聊天可用。
func IsWorkflowOnly(t Tool) bool {
	marker, ok := t.(interface{ WorkflowOnly() bool })
	return ok && marker.WorkflowOnly()
}

// ExecutionError 仅携带可向任务用户展示的固定安全原因，不包装供应商或基础设施原始错误。
type ExecutionError struct{ Message string }

func (e *ExecutionError) Error() string { return e.Message }

var registry = map[string]Tool{}

// Register 注册工具（重复注册 panic）
func Register(t Tool) {
	if _, exists := registry[t.Name()]; exists {
		panic("tool already registered: " + t.Name())
	}
	registry[t.Name()] = t
}

// RegisterIfAbsent 注册依赖注入的工具（如 kb_search 需装配期注入检索能力）；
// 同名已注册时跳过并返回 false，供服务装配与测试重复构建 svc 场景使用。
func RegisterIfAbsent(t Tool) bool {
	if _, exists := registry[t.Name()]; exists {
		return false
	}
	registry[t.Name()] = t
	return true
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
		if IsWorkflowOnly(t) {
			continue
		}
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
func (getCurrentTime) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var input struct{}
	if err := decodeObject(args, &input); err != nil {
		return "", err
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.FixedZone("CST", 8*60*60)
	}
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
		Text *string `json:"text"`
	}
	if err := decodeObject(args, &in); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if in.Text == nil || len(*in.Text) > 16000 {
		return "", fmt.Errorf("text 必填且最多 16000 字节")
	}
	chars := len([]rune(*in.Text))
	words := len(strings.Fields(*in.Text))
	lines := len(strings.Split(*in.Text, "\n"))
	// 返回结构化 JSON，模型可直接引用字段做后续推理。
	result, _ := json.Marshal(map[string]int{"chars": chars, "words": words, "lines": lines})
	return string(result), nil
}

func decodeObject(args json.RawMessage, out any) error {
	if len(args) > 64000 || !json.Valid(args) || !strings.HasPrefix(strings.TrimSpace(string(args)), "{") {
		return fmt.Errorf("参数必须为 JSON 对象且最多 64000 字节")
	}
	decoder := json.NewDecoder(strings.NewReader(string(args)))
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

// jsonFormatter 将任意文本转为合法 JSON（模型自检格式用）。
type jsonFormatter struct{}

func (jsonFormatter) Name() string { return "json_formatter" }
func (jsonFormatter) Description() string {
	return "将文本解析并格式化输出为缩进 JSON；输入非法 JSON 时返回错误"
}
func (jsonFormatter) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{
		"text": {Type: schema.String, Desc: "要格式化的 JSON 字符串", Required: true},
	}
}
func (jsonFormatter) Execute(_ context.Context, args json.RawMessage) (string, error) {
	var in struct {
		Text *string `json:"text"`
	}
	if err := decodeObject(args, &in); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if in.Text == nil || len(*in.Text) > 16000 {
		return "", fmt.Errorf("text 必填且最多 16000 字节")
	}
	var parsed any
	if err := json.Unmarshal([]byte(*in.Text), &parsed); err != nil {
		return "", fmt.Errorf("输入不是合法 JSON: %w", err)
	}
	formatted, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return "", fmt.Errorf("格式化失败: %w", err)
	}
	return string(formatted), nil
}

func init() {
	Register(getCurrentTime{})
	Register(textStats{})
	Register(jsonFormatter{})
}
