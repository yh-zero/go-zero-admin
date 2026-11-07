package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"strings"
)

var _ AiWorkflowModel = (*customAiWorkflowModel)(nil)

type (
	// AiWorkflowModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiWorkflowModel.
	AiWorkflowModel interface {
		aiWorkflowModel
		FindOwned(ctx context.Context, id, ownerId int64) (*WorkflowRow, error)
		ListOwned(ctx context.Context, ownerId, page, pageSize int64) ([]*WorkflowRow, int64, error)
		ListSearchOwned(ctx context.Context, ownerId, page, pageSize int64, query string) ([]*WorkflowRow, int64, error)
		UpdateOwned(ctx context.Context, id, ownerId int64, patch WorkflowPatch) error
		DeleteOwned(ctx context.Context, id, ownerId int64) error
		CreateTaskOwned(ctx context.Context, id, ownerId int64, input string) (*TaskRow, error)
		CreateTaskIdempotent(ctx context.Context, id, ownerId int64, input, requestId string) (*TaskRow, bool, error)
		FindTaskRequestOwned(ctx context.Context, ownerId int64, requestId string) (*TaskRow, error)
		CreateDraftTaskIdempotent(ctx context.Context, ownerId int64, name, definition, input, requestId, signature string) (*TaskRow, bool, error)
	}

	customAiWorkflowModel struct {
		*defaultAiWorkflowModel
		conn sqlx.SqlConn
	}
)

// NewAiWorkflowModel returns a model for the database table.
func NewAiWorkflowModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiWorkflowModel {
	return &customAiWorkflowModel{
		defaultAiWorkflowModel: newAiWorkflowModel(conn, c, opts...),
		conn:                   conn,
	}
}

var ErrWorkflowDisabled = errors.New("workflow disabled")
var ErrTaskRequestConflict = errors.New("workflow request payload mismatch")

func (m *customAiWorkflowModel) FindTaskRequestOwned(ctx context.Context, ownerId int64, requestId string) (*TaskRow, error) {
	var row TaskRow
	err := m.QueryRowNoCacheCtx(ctx, &row, taskSelect+"WHERE created_by=? AND request_id=?", ownerId, requestId)
	return &row, err
}

// CreateTaskOwned 与修改/停用/删除使用同一行锁，只有锁内仍启用的快照可创建任务。
func (m *customAiWorkflowModel) CreateTaskOwned(ctx context.Context, id, ownerId int64, input string) (*TaskRow, error) {
	task, _, err := m.CreateTaskIdempotent(ctx, id, ownerId, input, uuid.NewString())
	return task, err
}

func (m *customAiWorkflowModel) CreateTaskIdempotent(ctx context.Context, id, ownerId int64, input, requestId string) (*TaskRow, bool, error) {
	find := func() (*TaskRow, error) {
		var row TaskRow
		err := m.QueryRowNoCacheCtx(ctx, &row, taskSelect+"WHERE created_by=? AND request_id=?", ownerId, requestId)
		if err != nil {
			return nil, err
		}
		if row.WorkflowId != id || row.Input != input {
			return nil, ErrTaskRequestConflict
		}
		return &row, nil
	}
	if existing, err := find(); err == nil {
		return existing, false, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}
	var task *TaskRow
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var workflow WorkflowRow
		if err := session.QueryRowCtx(ctx, &workflow, workflowSelect+"WHERE id=? AND created_by=? AND deleted_at IS NULL FOR UPDATE", id, ownerId); err != nil {
			return err
		}
		if workflow.Enabled != 1 {
			return ErrWorkflowDisabled
		}
		summary := workflowInputSummary(input)
		result, err := session.ExecCtx(ctx, "INSERT INTO ai_task (workflow_id,status,input,created_by,workflow_snapshot,request_id,step_records,title,workflow_name,input_summary) VALUES (?,'running',?,?,?,?,'[]',?,?,?)", id, input, ownerId, workflow.Definition, requestId, workflow.Name, workflow.Name, summary)
		if err != nil {
			return err
		}
		taskId, err := result.LastInsertId()
		if err != nil {
			return err
		}
		task = &TaskRow{Id: taskId, WorkflowId: id, Status: "running", Input: input, CreatedBy: ownerId, WorkflowSnapshot: workflow.Definition, RequestId: requestId, StepRecords: "[]", Title: workflow.Name, WorkflowName: workflow.Name, InputSummary: summary}
		return nil
	})
	if err != nil {
		// 同一请求可能在另一个实例先提交；唯一键失败只读回原任务，不重复执行。
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

type WorkflowRow struct {
	Id              int64  `db:"id"`
	Name            string `db:"name"`
	Definition      string `db:"definition"`
	IntervalSeconds int64  `db:"interval_seconds"`
	Enabled         int64  `db:"enabled"`
	NextRunAt       int64  `db:"next_run_at"`
	CreatedBy       int64  `db:"created_by"`
	CreatedAt       int64  `db:"created_at"`
	UpdatedAt       int64  `db:"updated_at"`
	StepCount       int64  `db:"step_count"`
}

type WorkflowPatch struct {
	Name, Definition         *string
	IntervalSeconds, Enabled *int64
}

const workflowSelect = "SELECT id,name,CAST(definition AS CHAR) AS definition,interval_seconds,enabled,COALESCE(UNIX_TIMESTAMP(next_run_at),0) AS next_run_at,created_by,UNIX_TIMESTAMP(created_at) AS created_at,UNIX_TIMESTAMP(updated_at) AS updated_at,COALESCE(JSON_LENGTH(definition,'$.steps'),0) AS step_count FROM ai_workflow "
const workflowSummarySelect = "SELECT id,name,'' AS definition,interval_seconds,enabled,COALESCE(UNIX_TIMESTAMP(next_run_at),0) AS next_run_at,created_by,UNIX_TIMESTAMP(created_at) AS created_at,UNIX_TIMESTAMP(updated_at) AS updated_at,COALESCE(JSON_LENGTH(definition,'$.steps'),0) AS step_count FROM ai_workflow "

func (m *customAiWorkflowModel) FindOwned(ctx context.Context, id, ownerId int64) (*WorkflowRow, error) {
	var row WorkflowRow
	err := m.QueryRowNoCacheCtx(ctx, &row, workflowSelect+"WHERE id=? AND created_by=? AND deleted_at IS NULL", id, ownerId)
	return &row, err
}

func (m *customAiWorkflowModel) ListOwned(ctx context.Context, ownerId, page, pageSize int64) ([]*WorkflowRow, int64, error) {
	return m.ListSearchOwned(ctx, ownerId, page, pageSize, "")
}

func (m *customAiWorkflowModel) ListSearchOwned(ctx context.Context, ownerId, page, pageSize int64, query string) ([]*WorkflowRow, int64, error) {
	where := "created_by=? AND deleted_at IS NULL"
	args := []any{ownerId}
	if query != "" {
		where += " AND name LIKE ? ESCAPE '!'"
		args = append(args, "%"+strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(query)+"%")
	}
	var count struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM ai_workflow WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	rows := make([]*WorkflowRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, workflowSummarySelect+"WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, pageSize, (page-1)*pageSize)...)
	return rows, count.Total, err
}

func (m *customAiWorkflowModel) UpdateOwned(ctx context.Context, id, ownerId int64, patch WorkflowPatch) error {
	sets := make([]string, 0, 4)
	args := make([]any, 0, 6)
	if patch.Name != nil {
		sets = append(sets, "name=?")
		args = append(args, *patch.Name)
	}
	if patch.Definition != nil {
		sets = append(sets, "definition=?")
		args = append(args, *patch.Definition)
	}
	if patch.IntervalSeconds != nil {
		sets = append(sets, "interval_seconds=?", "next_run_at=NULL")
		args = append(args, *patch.IntervalSeconds)
	}
	if patch.Enabled != nil {
		sets = append(sets, "enabled=?")
		args = append(args, *patch.Enabled)
	}
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var owner struct {
			Id int64 `db:"id"`
		}
		if err := session.QueryRowCtx(ctx, &owner, "SELECT id FROM ai_workflow WHERE id=? AND created_by=? AND deleted_at IS NULL FOR UPDATE", id, ownerId); err != nil {
			return err
		}
		if len(sets) == 0 {
			return nil
		}
		_, err := session.ExecCtx(ctx, "UPDATE ai_workflow SET "+strings.Join(sets, ",")+" WHERE id=? AND created_by=?", append(args, id, ownerId)...)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiWorkflowIdPrefix, id))
}

func (m *customAiWorkflowModel) DeleteOwned(ctx context.Context, id, ownerId int64) error {
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var owner struct {
			Id int64 `db:"id"`
		}
		if err := session.QueryRowCtx(ctx, &owner, "SELECT id FROM ai_workflow WHERE id=? AND created_by=? AND deleted_at IS NULL FOR UPDATE", id, ownerId); err != nil {
			return err
		}
		_, err := session.ExecCtx(ctx, "UPDATE ai_workflow SET deleted_at=NOW(),enabled=0,next_run_at=NULL WHERE id=?", id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiWorkflowIdPrefix, id))
}
