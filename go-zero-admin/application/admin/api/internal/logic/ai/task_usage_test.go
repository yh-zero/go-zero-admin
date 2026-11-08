package ai

import (
	"testing"

	"go-zero-admin/application/ai/rpc/aipb"
)

func TestTaskUsageMappingPreservesKnownAndUnknown(t *testing.T) {
	for _, complete := range []bool{false, true} {
		task := mapAiTask(&aipb.AgentTaskInfo{Steps: []*aipb.WorkflowStepInfo{{Status: "failed", Usage: &aipb.WorkflowStepUsage{Scope: "chat", PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120, Complete: complete}}}})
		usage := task.Steps[0].Usage
		if usage == nil || usage.Scope != "chat" || usage.TotalTokens != 120 || usage.PromptTokens != 100 || usage.CompletionTokens != 20 || usage.Complete != complete {
			t.Fatalf("步骤已知用量及完整性必须原样映射: %+v", usage)
		}
	}
	if mapAiTask(&aipb.AgentTaskInfo{Steps: []*aipb.WorkflowStepInfo{{Status: "succeeded"}}}).Steps[0].Usage != nil {
		t.Fatal("旧记录不能填造用量")
	}
}
