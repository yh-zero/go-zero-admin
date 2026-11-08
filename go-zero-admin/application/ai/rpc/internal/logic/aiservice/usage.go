package aiservicelogic

import (
	"context"
	"errors"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

func usageAccess(ctx context.Context, s *svc.ServiceContext, actor *aipb.Actor, admin bool) error {
	if err := requireAiActor(actor); err != nil {
		return err
	}
	if s.UsageStore == nil {
		return xerr.NewErrCode(503)
	}
	if admin {
		allowed, err := s.UsageStore.IsAdmin(ctx, actor.UserId)
		if err != nil {
			return usageModelError(err)
		}
		if !allowed {
			return xerr.NewErrCode(403)
		}
	}
	return nil
}
func usageModelError(err error) error {
	switch {
	case errors.Is(err, model.ErrUsageInvalid):
		return xerr.NewErrCode(400)
	case errors.Is(err, model.ErrUsageConflict):
		return xerr.NewErrCodeMsg(409, "额度策略已变化，请读取当前版本核对")
	case errors.Is(err, model.ErrUsageQuota):
		return xerr.NewErrCodeMsg(429, "每日供应商请求额度已用完")
	case errors.Is(err, model.ErrUsageCapacity):
		return xerr.NewErrCodeMsg(429, "AI供应商请求容量已满，请稍后重试")
	default:
		return xerr.NewErrCodeMsg(503, "用量账本暂不可用，请稍后核对")
	}
}
func usageFilter(in *aipb.UsageQueryRequest, admin bool) model.UsageFilter {
	owner := in.Actor.GetUserId()
	if admin {
		owner = in.UserId
	}
	return model.UsageFilter{OwnerId: owner, AllUsers: admin, FromDay: in.FromDay, ToDay: in.ToDay, Source: in.Source, Operation: in.Operation, Provider: in.Provider, Model: in.Model, Status: in.Status, UsageState: in.UsageState, Page: in.GetPage().GetPage(), PageSize: in.GetPage().GetPageSize()}
}
func usageSummaryMessage(r *model.UsageSummary) *aipb.UsageSummaryInfo {
	if r == nil {
		return nil
	}
	out := &aipb.UsageSummaryInfo{Period: &aipb.UsagePeriod{FromDay: r.FromDay, ToDay: r.ToDay, Timezone: r.Timezone}, RecordingStartedAt: r.RecordingStartedAt, ConfirmedRequests: r.ConfirmedRequests, ReservedRequests: r.ReservedRequests, NotSentRequests: r.NotSentRequests, UncertainRequests: r.UncertainRequests, SucceededRequests: r.SucceededRequests, FailedRequests: r.FailedRequests, CancelledRequests: r.CancelledRequests, CompleteRequests: r.CompleteRequests, PartialRequests: r.PartialRequests, UnknownRequests: r.UnknownRequests, InvalidRequests: r.InvalidRequests, KnownPromptTokens: r.KnownPromptTokens, KnownCompletionTokens: r.KnownCompletionTokens, KnownEmbeddingTokens: r.KnownEmbeddingTokens, KnownTotalTokens: r.KnownTotalTokens, Groups: make([]*aipb.UsageGroup, 0, len(r.Groups))}
	for _, g := range r.Groups {
		out.Groups = append(out.Groups, &aipb.UsageGroup{Source: g.Source, Operation: g.Operation, Provider: g.Provider, Model: g.Model, ConfirmedRequests: g.ConfirmedRequests, ReservedRequests: g.ReservedRequests, NotSentRequests: g.NotSentRequests, UncertainRequests: g.UncertainRequests, SucceededRequests: g.SucceededRequests, FailedRequests: g.FailedRequests, CancelledRequests: g.CancelledRequests, CompleteRequests: g.CompleteRequests, PartialRequests: g.PartialRequests, UnknownRequests: g.UnknownRequests, InvalidRequests: g.InvalidRequests, KnownPromptTokens: g.KnownPromptTokens, KnownCompletionTokens: g.KnownCompletionTokens, KnownEmbeddingTokens: g.KnownEmbeddingTokens, KnownTotalTokens: g.KnownTotalTokens})
	}
	return out
}
func usageCallMessages(rows []*model.UsageCallRow) []*aipb.UsageCallInfo {
	out := make([]*aipb.UsageCallInfo, 0, len(rows))
	for _, r := range rows {
		out = append(out, &aipb.UsageCallInfo{Id: r.Id, AttemptId: r.AttemptId, OwnerId: r.OwnerId, Source: r.Source, Operation: r.Operation, RequestId: r.RequestId, SourceId: r.SourceId, StepIndex: r.StepIndex, BatchIndex: r.BatchIndex, Provider: r.Provider, Model: r.Model, QuotaDay: r.QuotaDay, QuotaCharged: r.QuotaCharged, DispatchConfirmed: r.DispatchConfirmed, Status: r.Status, UsageState: r.UsageState, PromptTokens: r.PromptTokens, CompletionTokens: r.CompletionTokens, TotalTokens: r.TotalTokens, HttpStatus: r.HttpStatus, ProviderRequestId: r.ProviderRequestId, ErrorCode: r.ErrorCode, StartedAt: r.StartedAt, DispatchedAt: r.DispatchedAt, CompletedAt: r.CompletedAt})
	}
	return out
}
func usagePolicyMessage(ctx context.Context, s *svc.ServiceContext, r *model.UsagePolicyView, admin bool) *aipb.UsagePolicyInfo {
	if r == nil {
		return nil
	}
	out := &aipb.UsagePolicyInfo{OwnerId: r.OwnerId, QuotaDay: r.QuotaDay, DailyRequestLimit: r.DailyRequestLimit, UsedRequests: r.UsedRequests, RemainingRequests: r.RemainingRequests, Unlimited: r.Unlimited, Inherited: r.Inherited, DailyTokenWarning: r.DailyTokenWarning, KnownTokens: r.KnownTokens, WarningReached: r.WarningReached, IncompleteRequests: r.IncompleteRequests, Version: r.Version, GlobalConcurrentLimit: r.GlobalConcurrentLimit}
	if admin && r.OwnerId == 0 && s.UsageGovernor != nil {
		active, err := s.UsageGovernor.Active(ctx)
		out.ActiveCalls = active
		out.ActiveKnown = err == nil
	}
	return out
}
