package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"time"
)

// WorkflowStepRecord 保留本次任务真正使用的输入；审批不重新读取可变的工作流。
type WorkflowStepRecord struct {
	Index       int64  `json:"index"`
	Id          string `json:"id"`
	Name        string `json:"name"`
	Tool        string `json:"tool"`
	Input       string `json:"input"`
	Status      string `json:"status"`
	Output      string `json:"output"`
	Error       string `json:"error"`
	StartedAt   int64  `json:"started_at"`
	CompletedAt int64  `json:"completed_at"`
}

func ReadWorkflowStepRecords(raw string) ([]WorkflowStepRecord, error) {
	if raw == "" {
		return []WorkflowStepRecord{}, nil
	}
	var records []WorkflowStepRecord
	if len(raw) > 128<<10 || json.Unmarshal([]byte(raw), &records) != nil || len(records) > 32 {
		return nil, errors.New("invalid workflow step records")
	}
	for i, record := range records {
		if record.Index != int64(i) {
			return nil, errors.New("invalid workflow record index")
		}
	}
	return records, nil
}

func (m *customAiTaskModel) AdvanceRecorded(ctx context.Context, id, ownerId, expectedStep, nextStep int64, status, result, errorMessage string, records []WorkflowStepRecord) error {
	raw, err := json.Marshal(records)
	if err != nil || len(raw) > 128<<10 || len(result) > 60000 {
		return errors.New("workflow records exceed size limit")
	}
	completed := "NULL"
	if status == "succeeded" || status == "failed" {
		completed = "NOW()"
	}
	return m.changeState(ctx, id, "UPDATE ai_task SET status=?,next_step=?,result=?,error=?,step_records=?,completed_at="+completed+",updated_at=NOW() WHERE id=? AND created_by=? AND status='running' AND next_step=?", status, nextStep, result, errorMessage, string(raw), id, ownerId, expectedStep)
}

func (m *customAiTaskModel) CancelOwned(ctx context.Context, id, ownerId int64) error {
	changed := false
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var row TaskRow
		if err := session.QueryRowCtx(ctx, &row, taskSelect+"WHERE id=? AND created_by=? FOR UPDATE", id, ownerId); err != nil {
			return err
		}
		if row.Status == "cancelled" {
			return nil
		}
		if row.Status != "running" && row.Status != "pending" && row.Status != "pending_approval" {
			return ErrTaskState
		}
		records, err := ReadWorkflowStepRecords(row.StepRecords)
		if err != nil {
			return err
		}
		if row.NextStep < int64(len(records)) {
			record := &records[row.NextStep]
			record.Status = "cancelled"
			record.CompletedAt = time.Now().UnixMilli()
			record.Error = "已请求取消，当前工具可能已产生外部作用，请核对结果"
		}
		raw, _ := json.Marshal(records)
		result, err := session.ExecCtx(ctx, "UPDATE ai_task SET status='cancelled',error=?,step_records=?,completed_at=NOW(),updated_at=NOW() WHERE id=? AND created_by=? AND status=? AND next_step=?", "用户取消；已发生的外部操作不会撤回", string(raw), id, ownerId, row.Status, row.NextStep)
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
		return err
	}
	if changed {
		if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiTaskIdPrefix, id)); err != nil {
			return ErrTaskCommitted
		}
	}
	return nil
}

// 超过执行上限的 running 只置未知失败，不领取或重放工具。
func (m *customAiTaskModel) FailStaleOwned(ctx context.Context, id, ownerId int64) error {
	var state struct {
		Status    string `db:"status"`
		UpdatedAt int64  `db:"updated_at"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &state, "SELECT status,UNIX_TIMESTAMP(updated_at) AS updated_at FROM ai_task WHERE id=? AND created_by=?", id, ownerId); err != nil {
		return err
	}
	if state.Status != "running" || state.UpdatedAt > time.Now().Add(-100*time.Second).Unix() {
		return nil
	}
	row, err := m.FindOwned(ctx, id, ownerId)
	if err != nil {
		return err
	}
	records, err := ReadWorkflowStepRecords(row.StepRecords)
	if err != nil {
		return err
	}
	message := "任务执行中断或超时，结果可能未知，请核对后另行运行"
	if row.NextStep < int64(len(records)) {
		r := &records[row.NextStep]
		r.Status = "failed"
		r.Error = message
		r.CompletedAt = time.Now().UnixMilli()
	}
	raw, _ := json.Marshal(records)
	err = m.changeState(ctx, id, "UPDATE ai_task SET status='failed',error=?,step_records=?,completed_at=NOW(),updated_at=NOW() WHERE id=? AND created_by=? AND status='running' AND next_step=? AND updated_at < DATE_SUB(NOW(), INTERVAL 100 SECOND)", message, string(raw), id, ownerId, row.NextStep)
	if errors.Is(err, ErrTaskState) {
		return nil
	}
	return err
}
