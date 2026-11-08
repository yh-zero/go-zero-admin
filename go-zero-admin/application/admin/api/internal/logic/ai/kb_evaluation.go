package ai

import (
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

func mapKbEvaluation(row *aipb.KbEvaluationInfo) *types.AiKbEvaluationInfo {
	if row == nil {
		return nil
	}
	return &types.AiKbEvaluationInfo{Id: row.Id, KbId: row.KbId, RequestId: row.RequestId,
		DatasetKey: row.DatasetKey, DatasetSha256: row.DatasetSha256, GroupKey: row.GroupKey,
		Status: row.Status, CaseCount: row.CaseCount, CompletedCount: row.CompletedCount,
		FailedCount: row.FailedCount, StopRequested: row.StopRequested, CorpusVerified: row.CorpusVerified,
		Error: row.Error, Snapshot: row.Snapshot, Bindings: row.Bindings, Context: row.Context,
		Observations: row.Observations, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		StartedAt: row.StartedAt, CompletedAt: row.CompletedAt}
}

func mapKbEvaluationList(rows []*aipb.KbEvaluationInfo) []*types.AiKbEvaluationInfo {
	list := make([]*types.AiKbEvaluationInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, mapKbEvaluation(row))
	}
	return list
}
