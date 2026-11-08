package svc

import (
	"context"
	"go-zero-admin/pkg/llm"
	"testing"
)

func TestUsageSourcePreservesParentAndNeverTransfersOwners(t *testing.T) {
	parent := llm.WithUsageIdentity(context.Background(), llm.UsageIdentity{UserID: 7, Source: "evaluation", RequestID: "evaluation-request", SourceID: 19, StepIndex: -1})
	ctx := WithUsageSource(parent, 7, "analysis", "task-request", 31, 2)
	got, ok := llm.UsageIdentityFrom(ctx)
	if !ok || got.Source != "evaluation" || got.SourceID != 19 || got.StepIndex != 2 {
		t.Fatalf("内部分析/检索不能丢失评测来源：%+v", got)
	}
	ctx = WithUsageSource(parent, 8, "kb_ingest", "job-request", 42, -1)
	got, ok = llm.UsageIdentityFrom(ctx)
	if !ok || got.UserID != 8 || got.Source != "kb_ingest" || got.RequestID != "job-request" || got.SourceID != 42 {
		t.Fatalf("不同归属不能继承另一人的计量身份：%+v", got)
	}
}
