package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiTaskModel = (*customAiTaskModel)(nil)

type (
	// AiTaskModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiTaskModel.
	AiTaskModel interface {
		aiTaskModel
		FindOwned(ctx context.Context, id, ownerId int64) (*TaskRow, error)
		ListOwned(ctx context.Context, ownerId, page, pageSize int64, status string, workflowId int64) ([]*TaskRow, int64, error)
		ClaimApproval(ctx context.Context, id, ownerId, expectedStep int64) error
		RejectOwned(ctx context.Context, id, ownerId, expectedStep int64, reason string) error
		Advance(ctx context.Context, id, ownerId, expectedStep, nextStep int64, status, result, errorMessage string) error
	}

	customAiTaskModel struct {
		*defaultAiTaskModel
	}
)

// NewAiTaskModel returns a model for the database table.
func NewAiTaskModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiTaskModel {
	return &customAiTaskModel{
		defaultAiTaskModel: newAiTaskModel(conn, c, opts...),
	}
}

var ErrTaskState = errors.New("task state does not permit this operation")

type TaskRow struct {
	Id               int64  `db:"id"`
	WorkflowId       int64  `db:"workflow_id"`
	ConversationId   int64  `db:"conversation_id"`
	Status           string `db:"status"`
	Input            string `db:"input"`
	Result           string `db:"result"`
	Error            string `db:"error"`
	ApprovedBy       int64  `db:"approved_by"`
	CreatedAt        int64  `db:"created_at"`
	CompletedAt      int64  `db:"completed_at"`
	CreatedBy        int64  `db:"created_by"`
	NextStep         int64  `db:"next_step"`
	WorkflowSnapshot string `db:"workflow_snapshot"`
}

const taskSelect = "SELECT id,COALESCE(workflow_id,0) AS workflow_id,COALESCE(conversation_id,0) AS conversation_id,status,COALESCE(input,'') AS input,COALESCE(result,'') AS result,error,COALESCE(approved_by,0) AS approved_by,UNIX_TIMESTAMP(created_at) AS created_at,COALESCE(UNIX_TIMESTAMP(completed_at),0) AS completed_at,created_by,next_step,COALESCE(CAST(workflow_snapshot AS CHAR),'') AS workflow_snapshot FROM ai_task "

func (m *customAiTaskModel) FindOwned(ctx context.Context, id, ownerId int64) (*TaskRow, error) {
	var row TaskRow
	err := m.QueryRowNoCacheCtx(ctx, &row, taskSelect+"WHERE id=? AND created_by=?", id, ownerId)
	return &row, err
}

func (m *customAiTaskModel) ListOwned(ctx context.Context, ownerId, page, pageSize int64, status string, workflowId int64) ([]*TaskRow, int64, error) {
	where := "created_by=?"
	args := []any{ownerId}
	if status != "" {
		where += " AND status=?"
		args = append(args, status)
	}
	if workflowId > 0 {
		where += " AND workflow_id=?"
		args = append(args, workflowId)
	}
	var count struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM ai_task WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	rows := make([]*TaskRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, taskSelect+"WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, pageSize, (page-1)*pageSize)...)
	return rows, count.Total, err
}

func (m *customAiTaskModel) ClaimApproval(ctx context.Context, id, ownerId, expectedStep int64) error {
	return m.changeState(ctx, id, "UPDATE ai_task SET status='running',approved_by=? WHERE id=? AND created_by=? AND status='pending_approval' AND next_step=? AND workflow_snapshot IS NOT NULL", ownerId, id, ownerId, expectedStep)
}

func (m *customAiTaskModel) RejectOwned(ctx context.Context, id, ownerId, expectedStep int64, reason string) error {
	return m.changeState(ctx, id, "UPDATE ai_task SET status='rejected',approved_by=?,error=?,completed_at=NOW() WHERE id=? AND created_by=? AND status='pending_approval' AND next_step=?", ownerId, reason, id, ownerId, expectedStep)
}

func (m *customAiTaskModel) Advance(ctx context.Context, id, ownerId, expectedStep, nextStep int64, status, result, errorMessage string) error {
	completed := "NULL"
	if status == "succeeded" || status == "failed" {
		completed = "NOW()"
	}
	return m.changeState(ctx, id, "UPDATE ai_task SET status=?,next_step=?,result=?,error=?,completed_at="+completed+" WHERE id=? AND created_by=? AND status='running' AND next_step=?", status, nextStep, result, errorMessage, id, ownerId, expectedStep)
}

func (m *customAiTaskModel) changeState(ctx context.Context, id int64, query string, args ...any) error {
	result, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, query, args...)
	}, fmt.Sprintf("%s%d", cacheGozeroAdminAiTaskIdPrefix, id))
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrTaskState
	}
	return nil
}
