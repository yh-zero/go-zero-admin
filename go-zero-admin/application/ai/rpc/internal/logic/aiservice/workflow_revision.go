package aiservicelogic

import (
	"errors"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/xerr"
)

func workflowModelError(err error) error {
	if errors.Is(err, model.ErrWorkflowRevisionConflict) {
		return xerr.NewErrCodeMsg(409, "工作流版本已变化，请核对最新内容后再保存")
	}
	if errors.Is(err, model.ErrWorkflowCommitted) {
		return xerr.NewErrCodeMsg(503, "工作流已提交，缓存刷新失败，请核对最新版本")
	}
	return aiModelError(err)
}

func workflowRevisionProto(row *model.WorkflowRevisionRow) *aipb.WorkflowRevisionInfo {
	return &aipb.WorkflowRevisionInfo{Id: row.Id, WorkflowId: row.WorkflowId, Revision: row.RevisionSeq, Name: row.Name, Definition: row.Definition, IntervalSeconds: row.IntervalSeconds, Enabled: row.Enabled, StepCount: row.StepCount, CreatedAt: row.CreatedAt}
}
