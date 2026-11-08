package ai

import (
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

func mapAiWorkflowRevision(row *aipb.WorkflowRevisionInfo) *types.AiWorkflowRevisionInfo {
	if row == nil {
		return nil
	}
	return &types.AiWorkflowRevisionInfo{Id: row.Id, WorkflowId: row.WorkflowId, Revision: row.Revision, Name: row.Name, Definition: row.Definition, IntervalSeconds: row.IntervalSeconds, Enabled: row.Enabled, StepCount: row.StepCount, CreatedAt: row.CreatedAt}
}
