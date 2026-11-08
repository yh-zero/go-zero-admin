package llm

import "context"

type UsageIdentity struct {
	UserID              int64
	Source, RequestID   string
	SourceID, StepIndex int64
}

type UsageAttempt struct {
	UsageIdentity
	AttemptID, Operation, Provider, Model string
	BatchIndex, InputItems                int64
}

type UsageObservation struct {
	Status, UsageState                          string
	PromptTokens, CompletionTokens, TotalTokens *int64
	HTTPStatus                                  int64
	ProviderRequestID, ErrorCode                string
	DispatchConfirmed                           bool
}

// UsageHook 在业务库预留/结算调用，在共享 Redis 控制活跃租约；实现不得保存模型正文或凭据。
type UsageHook interface {
	Begin(context.Context, UsageAttempt) error
	Dispatched(context.Context, string) error
	Finish(context.Context, string, UsageObservation) error
}

type usageIdentityKey struct{}

func WithUsageIdentity(ctx context.Context, identity UsageIdentity) context.Context {
	return context.WithValue(ctx, usageIdentityKey{}, identity)
}

func UsageIdentityFrom(ctx context.Context) (UsageIdentity, bool) {
	identity, ok := ctx.Value(usageIdentityKey{}).(UsageIdentity)
	return identity, ok && identity.UserID > 0 && identity.Source != ""
}
