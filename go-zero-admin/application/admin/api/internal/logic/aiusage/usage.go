package aiusage

import (
	"context"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"
)

func usageActor(ctx context.Context, s *svc.ServiceContext, request any, admin bool) (*aipb.Actor, error) {
	if request != nil {
		if err := s.Validator.StructCtx(ctx, request); err != nil {
			return nil, xerr.NewErrCode(400)
		}
	}
	actor := middleware.ActorFromContext(ctx)
	if actor.UserId <= 0 {
		return nil, xerr.NewErrCode(403)
	}
	if s.AiRpc == nil {
		return nil, xerr.NewErrCode(503)
	}
	if admin {
		if s.ProfileRpc == nil {
			return nil, xerr.NewErrCode(503)
		}
		profile, err := s.ProfileRpc.GetProfile(ctx, &adminpb.GetProfileRequest{Actor: actor})
		if err != nil {
			return nil, err
		}
		if profile.User == nil || profile.User.Status != 1 || !profile.User.IsSuperAdmin {
			return nil, xerr.NewErrCode(403)
		}
	}
	return &aipb.Actor{UserId: actor.UserId, Username: actor.Username, DeptId: actor.DeptId, Ip: actor.Ip, UserAgent: actor.UserAgent}, nil
}
func selfUsageQuery(r *types.AiUsageQueryReq) *aipb.UsageQueryRequest {
	return &aipb.UsageQueryRequest{FromDay: r.FromDay, ToDay: r.ToDay, Source: r.Source, Operation: r.Operation, Provider: r.Provider, Model: r.Model, Status: r.Status, UsageState: r.UsageState, Page: &aipb.PageRequest{Page: r.Page, PageSize: r.PageSize}}
}
func adminUsageQuery(r *types.AiUsageAdminQueryReq) *aipb.UsageQueryRequest {
	return &aipb.UsageQueryRequest{UserId: r.UserId, FromDay: r.FromDay, ToDay: r.ToDay, Source: r.Source, Operation: r.Operation, Provider: r.Provider, Model: r.Model, Status: r.Status, UsageState: r.UsageState, Page: &aipb.PageRequest{Page: r.Page, PageSize: r.PageSize}}
}
func mapUsageSummary(r *aipb.UsageSummaryInfo) *types.AiUsageSummaryInfo {
	if r == nil {
		return nil
	}
	result := &types.AiUsageSummaryInfo{RecordingStartedAt: r.RecordingStartedAt, Groups: make([]*types.AiUsageGroup, 0, len(r.Groups)), ConfirmedRequests: r.ConfirmedRequests, ReservedRequests: r.ReservedRequests, NotSentRequests: r.NotSentRequests, UncertainRequests: r.UncertainRequests, SucceededRequests: r.SucceededRequests, FailedRequests: r.FailedRequests, CancelledRequests: r.CancelledRequests, CompleteRequests: r.CompleteRequests, PartialRequests: r.PartialRequests, UnknownRequests: r.UnknownRequests, InvalidRequests: r.InvalidRequests, KnownPromptTokens: r.KnownPromptTokens, KnownCompletionTokens: r.KnownCompletionTokens, KnownEmbeddingTokens: r.KnownEmbeddingTokens, KnownTotalTokens: r.KnownTotalTokens}
	if r.Period != nil {
		result.Period = &types.AiUsagePeriod{FromDay: r.Period.FromDay, ToDay: r.Period.ToDay, Timezone: r.Period.Timezone}
	}
	for _, g := range r.Groups {
		if g != nil {
			result.Groups = append(result.Groups, &types.AiUsageGroup{Source: g.Source, Operation: g.Operation, Provider: g.Provider, Model: g.Model, ConfirmedRequests: g.ConfirmedRequests, ReservedRequests: g.ReservedRequests, NotSentRequests: g.NotSentRequests, UncertainRequests: g.UncertainRequests, SucceededRequests: g.SucceededRequests, FailedRequests: g.FailedRequests, CancelledRequests: g.CancelledRequests, CompleteRequests: g.CompleteRequests, PartialRequests: g.PartialRequests, UnknownRequests: g.UnknownRequests, InvalidRequests: g.InvalidRequests, KnownPromptTokens: g.KnownPromptTokens, KnownCompletionTokens: g.KnownCompletionTokens, KnownEmbeddingTokens: g.KnownEmbeddingTokens, KnownTotalTokens: g.KnownTotalTokens})
		}
	}
	return result
}
func mapUsageCalls(rows []*aipb.UsageCallInfo) []*types.AiUsageCallInfo {
	result := make([]*types.AiUsageCallInfo, 0, len(rows))
	for _, r := range rows {
		if r == nil {
			continue
		}
		result = append(result, &types.AiUsageCallInfo{Id: r.Id, AttemptId: r.AttemptId, OwnerId: r.OwnerId, Source: r.Source, Operation: r.Operation, RequestId: r.RequestId, SourceId: r.SourceId, StepIndex: r.StepIndex, BatchIndex: r.BatchIndex, Provider: r.Provider, Model: r.Model, QuotaDay: r.QuotaDay, QuotaCharged: r.QuotaCharged, DispatchConfirmed: r.DispatchConfirmed, Status: r.Status, UsageState: r.UsageState, PromptTokens: r.PromptTokens, CompletionTokens: r.CompletionTokens, TotalTokens: r.TotalTokens, HttpStatus: r.HttpStatus, ProviderRequestId: r.ProviderRequestId, ErrorCode: r.ErrorCode, StartedAt: r.StartedAt, DispatchedAt: r.DispatchedAt, CompletedAt: r.CompletedAt})
	}
	return result
}
func mapUsagePolicy(r *aipb.UsagePolicyInfo) *types.AiUsagePolicyInfo {
	if r == nil {
		return nil
	}
	return &types.AiUsagePolicyInfo{OwnerId: r.OwnerId, QuotaDay: r.QuotaDay, DailyRequestLimit: r.DailyRequestLimit, UsedRequests: r.UsedRequests, RemainingRequests: r.RemainingRequests, Unlimited: r.Unlimited, Inherited: r.Inherited, DailyTokenWarning: r.DailyTokenWarning, KnownTokens: r.KnownTokens, WarningReached: r.WarningReached, IncompleteRequests: r.IncompleteRequests, Version: r.Version, GlobalConcurrentLimit: r.GlobalConcurrentLimit, ActiveCalls: r.ActiveCalls, ActiveKnown: r.ActiveKnown}
}
