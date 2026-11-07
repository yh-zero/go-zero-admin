package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func workflowInputSummary(input string) string {
	var object map[string]json.RawMessage
	if json.Unmarshal([]byte(input), &object) != nil {
		return ""
	}
	for _, key := range []string{"question", "text"} {
		var value string
		if json.Unmarshal(object[key], &value) == nil && strings.TrimSpace(value) != "" {
			runes := []rune(strings.TrimSpace(value))
			return string(runes[:min(256, len(runes))])
		}
	}
	return ""
}

func (m *customAiWorkflowModel) CreateDraftTaskIdempotent(ctx context.Context, ownerId int64, name, definition, input, requestId, signature string) (*TaskRow, bool, error) {
	find := func() (*TaskRow, error) {
		row, err := m.FindTaskRequestOwned(ctx, ownerId, requestId)
		if err != nil {
			return nil, err
		}
		if row.WorkflowId != 0 || row.RequestSignature != signature {
			return nil, ErrTaskRequestConflict
		}
		return row, nil
	}
	if existing, err := find(); err == nil {
		return existing, false, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}
	var task *TaskRow
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		summary := workflowInputSummary(input)
		result, err := session.ExecCtx(ctx, "INSERT INTO ai_task (workflow_id,status,input,created_by,workflow_snapshot,request_id,request_signature,step_records,title,workflow_name,input_summary) VALUES (NULL,'running',?,?,?,?,?,'[]',?,?,?)", input, ownerId, definition, requestId, signature, name, name, summary)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		task = &TaskRow{Id: id, Status: "running", Input: input, CreatedBy: ownerId, WorkflowSnapshot: definition, RequestId: requestId, RequestSignature: signature, StepRecords: "[]", Title: name, WorkflowName: name, InputSummary: summary}
		return nil
	})
	if err != nil {
		if existing, readErr := find(); readErr == nil {
			return existing, false, nil
		} else if errors.Is(readErr, ErrTaskRequestConflict) {
			return nil, false, readErr
		}
		return nil, false, err
	}
	if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiTaskIdPrefix, task.Id), fmt.Sprintf("%s%v:%v", cacheGozeroAdminAiTaskCreatedByRequestIdPrefix, ownerId, sql.NullString{String: requestId, Valid: true})); err != nil {
		return task, true, ErrTaskCommitted
	}
	return task, true, nil
}
