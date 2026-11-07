package aiservicelogic

import (
	"encoding/json"
	"testing"

	"go-zero-admin/application/ai/rpc/internal/model"
)

func TestWorkflowBindingsResolveTypedPriorOutput(t *testing.T) {
	definition, err := parseWorkflow(`{"version":2,"steps":[{"id":"analyse","name":"分析","tool":"text_stats","input":{"text":{"$ref":"input.text"}}},{"id":"format","name":"整理","tool":"text_stats","input":{"text":{"$ref":"steps.analyse.analysis"}}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	var first map[string]any
	raw, err := resolveWorkflowInput(definition, 0, `{"text":"本次文本"}`, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(raw, &first)
	if first["text"] != "本次文本" {
		t.Fatalf("运行文本未注入: %s", raw)
	}
	records := []model.WorkflowStepRecord{{Index: 0, Id: "analyse", Status: "succeeded", Output: `{"analysis":"真实前一步"}`}}
	raw, err = resolveWorkflowInput(definition, 1, `{}`, records)
	if err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(raw, &first)
	if first["text"] != "真实前一步" {
		t.Fatalf("前序结果未绑定: %s", raw)
	}
}

func TestWorkflowBindingsRejectUnsafeOrUnresolvedReferences(t *testing.T) {
	for _, ref := range []string{"steps.future.analysis", "input.__proto__", "steps.first.constructor", "input.missing", "input.count"} {
		t.Run(ref, func(t *testing.T) {
			raw, _ := json.Marshal(map[string]any{"version": 2, "steps": []any{map[string]any{"id": "first", "name": "统计", "tool": "text_stats", "input": map[string]any{"text": map[string]any{"$ref": ref}}}}})
			d, err := parseWorkflow(string(raw))
			if err == nil {
				_, err = resolveWorkflowInput(d, 0, `{"count":4}`, nil)
			}
			if err == nil {
				t.Fatal("非法、未解析或类型错误引用必须失败")
			}
		})
	}
	for _, raw := range []string{
		`{"version":99,"steps":[{"id":"a","name":"统计","tool":"text_stats","input":{"text":"x"}}]}`,
		`{"version":2,"steps":[{"id":"a","name":"统计","tool":"text_stats","input":{"text":"x"}},{"id":"a","name":"统计","tool":"text_stats","input":{"text":"y"}}]}`,
		`{"version":2,"steps":[{"id":"a","name":"统计","tool":"text_stats","input":{"text":{"$ref":"input.text","fallback":"unsafe"}}}]}`,
	} {
		if _, err := parseWorkflow(raw); err == nil {
			t.Fatalf("非法定义应拒绝: %s", raw)
		}
	}
}

func TestWorkflowBindingsLegacyAndWholeOutput(t *testing.T) {
	d, err := parseWorkflow(`{"steps":[{"name":"旧步骤","tool":"text_stats","input":{"text":"保留原值"}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = resolveWorkflowInput(d, 0, `{}`, nil); err != nil {
		t.Fatal(err)
	}
	d, err = parseWorkflow(`{"version":2,"steps":[{"id":"a","name":"统计","tool":"text_stats","input":{"text":"x"}},{"id":"b","name":"整理","tool":"json_formatter","input":{"text":{"$ref":"steps.a"}}}]}`)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := resolveWorkflowInput(d, 1, `{}`, []model.WorkflowStepRecord{{Index: 0, Id: "a", Status: "succeeded", Output: `{"chars":1}`}})
	if err != nil || string(raw) != `{"text":"{\"chars\":1}"}` {
		t.Fatalf("完整输出须保留字符串: %s %v", raw, err)
	}
}
