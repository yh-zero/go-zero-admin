package aiservicelogic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
)

var workflowStepID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func unsafeReferencePart(part string) bool {
	return part == "__proto__" || part == "prototype" || part == "constructor" || part == ""
}

func checkWorkflowReferences(value any, prior map[string]bool, depth int) error {
	if depth > 8 {
		return fmt.Errorf("参数嵌套最多 8 层")
	}
	switch item := value.(type) {
	case map[string]any:
		if ref, exists := item["$ref"]; exists {
			text, ok := ref.(string)
			if !ok || len(item) != 1 {
				return fmt.Errorf("变量引用须为单键 $ref 对象")
			}
			parts := strings.Split(text, ".")
			if len(parts) < 2 || len(parts) > 8 {
				return fmt.Errorf("变量路径无效")
			}
			for _, part := range parts {
				if unsafeReferencePart(part) {
					return fmt.Errorf("变量路径无效")
				}
			}
			if parts[0] != "input" && (parts[0] != "steps" || !prior[parts[1]]) {
				return fmt.Errorf("只能引用本次输入或前序步骤结果")
			}
			return nil
		}
		for key, child := range item {
			if unsafeReferencePart(key) {
				return fmt.Errorf("参数字段无效")
			}
			if err := checkWorkflowReferences(child, prior, depth+1); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range item {
			if err := checkWorkflowReferences(child, prior, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func readWorkflowJSON(raw string) (any, error) {
	var value any
	d := json.NewDecoder(strings.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&value); err != nil {
		return nil, err
	}
	return value, nil
}

func resolveWorkflowInput(def *workflowDefinition, index int64, input string, records []model.WorkflowStepRecord) (json.RawMessage, error) {
	runtime, err := readWorkflowJSON(input)
	if err != nil {
		return nil, fmt.Errorf("运行输入不是有效 JSON")
	}
	if _, ok := runtime.(map[string]any); !ok {
		return nil, fmt.Errorf("运行输入必须为对象")
	}
	stepInput, err := readWorkflowJSON(string(def.Steps[index].Input))
	if err != nil {
		return nil, err
	}
	resolved, err := resolveWorkflowValue(stepInput, runtime, records, 0)
	if err != nil {
		return nil, err
	}
	values, ok := resolved.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("步骤参数必须为对象")
	}
	tool, ok := agenttools.Get(def.Steps[index].Tool)
	if !ok {
		return nil, fmt.Errorf("步骤工具不可用")
	}
	rules := tool.Params()
	for name, rule := range rules {
		if rule.Required {
			if _, ok := values[name]; !ok {
				return nil, fmt.Errorf("步骤参数 %s 必填", name)
			}
		}
	}
	for name, value := range values {
		rule, ok := rules[name]
		if !ok {
			return nil, fmt.Errorf("步骤包含未知参数")
		}
		valid := false
		switch string(rule.Type) {
		case "string":
			_, valid = value.(string)
		case "boolean":
			_, valid = value.(bool)
		case "number", "integer":
			n, yes := value.(json.Number)
			if yes {
				f, e := strconv.ParseFloat(string(n), 64)
				valid = e == nil && !math.IsInf(f, 0) && !math.IsNaN(f) && (string(rule.Type) != "integer" || math.Trunc(f) == f && math.Abs(f) <= 9007199254740991)
			}
		case "object":
			_, valid = value.(map[string]any)
		case "array":
			_, valid = value.([]any)
		}
		if !valid {
			return nil, fmt.Errorf("步骤参数 %s 的变量类型不匹配", name)
		}
		if len(rule.Enum) > 0 {
			encoded, _ := json.Marshal(value)
			matched := false
			for _, option := range rule.Enum {
				candidate, _ := json.Marshal(option)
				if bytes.Equal(encoded, candidate) {
					matched = true
					break
				}
			}
			if !matched {
				return nil, fmt.Errorf("步骤参数 %s 不在可选范围", name)
			}
		}
	}
	raw, err := json.Marshal(values)
	if err != nil || len(raw) > 64000 {
		return nil, fmt.Errorf("解析后步骤参数最多 64000 字节")
	}
	return raw, nil
}

func resolveWorkflowValue(value, runtime any, records []model.WorkflowStepRecord, depth int) (any, error) {
	if depth > 8 {
		return nil, fmt.Errorf("参数嵌套最多 8 层")
	}
	switch item := value.(type) {
	case map[string]any:
		if reference, exists := item["$ref"]; exists {
			ref, ok := reference.(string)
			if !ok || len(item) != 1 {
				return nil, fmt.Errorf("变量引用无效")
			}
			parts := strings.Split(ref, ".")
			if len(parts) < 2 || len(parts) > 8 {
				return nil, fmt.Errorf("变量路径无效")
			}
			for _, part := range parts {
				if unsafeReferencePart(part) {
					return nil, fmt.Errorf("变量路径无效")
				}
			}
			var selected any
			path := parts[1:]
			if parts[0] == "input" {
				selected = runtime
			} else if parts[0] == "steps" {
				found := false
				for _, record := range records {
					if record.Id == parts[1] && record.Status == "succeeded" {
						found = true
						if len(parts) == 2 {
							return record.Output, nil
						}
						var err error
						selected, err = readWorkflowJSON(record.Output)
						if err != nil {
							return nil, fmt.Errorf("前序输出不是可读取字段的 JSON")
						}
						break
					}
				}
				if !found {
					return nil, fmt.Errorf("前序步骤尚无已提交结果")
				}
				path = parts[2:]
			} else {
				return nil, fmt.Errorf("变量路径无效")
			}
			for _, part := range path {
				object, ok := selected.(map[string]any)
				if !ok {
					return nil, fmt.Errorf("变量字段不存在")
				}
				selected, ok = object[part]
				if !ok || selected == nil {
					return nil, fmt.Errorf("变量字段不存在")
				}
			}
			return selected, nil
		}
		out := make(map[string]any, len(item))
		for key, child := range item {
			if unsafeReferencePart(key) {
				return nil, fmt.Errorf("参数字段无效")
			}
			resolved, err := resolveWorkflowValue(child, runtime, records, depth+1)
			if err != nil {
				return nil, err
			}
			out[key] = resolved
		}
		return out, nil
	case []any:
		out := make([]any, len(item))
		for i, child := range item {
			resolved, err := resolveWorkflowValue(child, runtime, records, depth+1)
			if err != nil {
				return nil, err
			}
			out[i] = resolved
		}
		return out, nil
	default:
		return value, nil
	}
}
