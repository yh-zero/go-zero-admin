package svc

import (
	"context"
	"go-zero-admin/pkg/llm"
)

// WithUsageSource 只接收编排层已验证的归属；内部分析/检索保留同一操作者的上层来源。
func WithUsageSource(ctx context.Context, ownerID int64, source, requestID string, sourceID, step int64) context.Context {
	identity, ok := llm.UsageIdentityFrom(ctx)
	if !ok || identity.UserID != ownerID {
		identity = llm.UsageIdentity{UserID: ownerID, Source: source, RequestID: requestID, SourceID: sourceID, StepIndex: -1}
	}
	if step >= 0 {
		identity.StepIndex = step
	}
	return llm.WithUsageIdentity(ctx, identity)
}
