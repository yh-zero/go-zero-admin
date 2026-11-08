package model

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// StoreTerminalStepUsage 只补回同一次已终止执行的计量，不恢复任务、推进断点或保存迟到正文。
func (m *customAiTaskModel) StoreTerminalStepUsage(ctx context.Context, id, owner int64, expected WorkflowStepRecord, usage WorkflowStepUsage) (string, error) {
	if expected.Index < 0 || expected.StartedAt <= 0 || expected.Id == "" || expected.Tool == "" || usage.Scope != "chat" || usage.PromptTokens < 0 || usage.CompletionTokens < 0 || usage.TotalTokens < 0 || usage.PromptTokens > 10_000_000 || usage.CompletionTokens > 10_000_000 || usage.TotalTokens > 10_000_000 || usage.Complete {
		return "", ErrTaskState
	}
	var status string
	changed := false
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var row TaskRow
		if err := session.QueryRowCtx(ctx, &row, taskSelect+"WHERE id=? AND created_by=? FOR UPDATE", id, owner); err != nil {
			return err
		}
		if (row.Status != "cancelled" && row.Status != "failed") || row.NextStep != expected.Index {
			return ErrTaskState
		}
		records, err := ReadWorkflowStepRecords(row.StepRecords)
		if err != nil || expected.Index >= int64(len(records)) {
			return ErrTaskState
		}
		record := &records[expected.Index]
		if record.Index != expected.Index || record.Id != expected.Id || record.Tool != expected.Tool || record.StartedAt != expected.StartedAt || record.Status != row.Status {
			return ErrTaskState
		}
		status = row.Status
		if record.Usage != nil {
			if *record.Usage != usage {
				return ErrTaskState
			}
			return nil
		}
		record.Usage = &usage
		raw, err := json.Marshal(records)
		if err != nil || len(raw) > 128<<10 {
			return ErrTaskState
		}
		result, err := session.ExecCtx(ctx, "UPDATE ai_task SET step_records=?,updated_at=updated_at WHERE id=? AND created_by=? AND status=? AND next_step=?", string(raw), id, owner, row.Status, expected.Index)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrTaskState
		}
		changed = true
		return nil
	})
	if err != nil {
		return "", err
	}
	if changed {
		if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiTaskIdPrefix, id)); err != nil {
			return status, ErrTaskCommitted
		}
	}
	return status, nil
}
