package model

import (
	"context"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"strings"
	"time"
)

var _ AiTaskModel = (*customAiTaskModel)(nil)

type (
	// AiTaskModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiTaskModel.
	AiTaskModel interface {
		aiTaskModel
		FindOwned(ctx context.Context, id, ownerId int64) (*TaskRow, error)
		FindSummaryOwned(ctx context.Context, id, ownerId int64) (*TaskRow, error)
		ListOwned(ctx context.Context, ownerId, page, pageSize int64, status string, workflowId int64) ([]*TaskRow, int64, error)
		ListSearchOwned(ctx context.Context, ownerId, page, pageSize int64, status string, workflowId int64, query string) ([]*TaskRow, int64, error)
		ClaimApproval(ctx context.Context, id, ownerId, expectedStep int64) error
		RejectOwned(ctx context.Context, id, ownerId, expectedStep int64, reason string) error
		Advance(ctx context.Context, id, ownerId, expectedStep, nextStep int64, status, result, errorMessage string) error
		AdvanceRecorded(ctx context.Context, id, ownerId, expectedStep, nextStep int64, status, result, errorMessage string, records []WorkflowStepRecord) error
		CancelOwned(ctx context.Context, id, ownerId int64) error
		FailStaleOwned(ctx context.Context, id, ownerId int64) error
		CheckRunningOwned(ctx context.Context, id, ownerId int64) (bool, error)
	}

	customAiTaskModel struct {
		*defaultAiTaskModel
		conn sqlx.SqlConn
	}
)

// NewAiTaskModel returns a model for the database table.
func NewAiTaskModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiTaskModel {
	return &customAiTaskModel{
		defaultAiTaskModel: newAiTaskModel(conn, c, opts...),
		conn:               conn,
	}
}

var ErrTaskState = errors.New("task state does not permit this operation")
var ErrTaskCommitted = errors.New("task SQL committed; cache refresh failed")

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
	RequestId        string `db:"request_id"`
	StepRecords      string `db:"step_records"`
	UpdatedAt        int64  `db:"updated_at"`
	RequestSignature string `db:"request_signature"`
	Title            string `db:"title"`
	WorkflowName     string `db:"workflow_name"`
	InputSummary     string `db:"input_summary"`
}

const taskSelect = "SELECT id,COALESCE(workflow_id,0) AS workflow_id,COALESCE(conversation_id,0) AS conversation_id,status,COALESCE(input,'') AS input,COALESCE(result,'') AS result,error,COALESCE(approved_by,0) AS approved_by,UNIX_TIMESTAMP(created_at) AS created_at,COALESCE(UNIX_TIMESTAMP(completed_at),0) AS completed_at,created_by,next_step,COALESCE(CAST(workflow_snapshot AS CHAR),'') AS workflow_snapshot,COALESCE(request_id,'') AS request_id,COALESCE(CAST(step_records AS CHAR),'[]') AS step_records,UNIX_TIMESTAMP(updated_at) AS updated_at,request_signature,title,workflow_name,input_summary FROM ai_task "

// 列表不读取大字段；100 条最大分页仍保持有界，执行内容仅在本人详情中读取。
const taskSummarySelect = "SELECT id,COALESCE(workflow_id,0) AS workflow_id,COALESCE(conversation_id,0) AS conversation_id,status,'' AS input,'' AS result,error,COALESCE(approved_by,0) AS approved_by,UNIX_TIMESTAMP(created_at) AS created_at,COALESCE(UNIX_TIMESTAMP(completed_at),0) AS completed_at,created_by,next_step,'' AS workflow_snapshot,COALESCE(request_id,'') AS request_id,'[]' AS step_records,UNIX_TIMESTAMP(updated_at) AS updated_at,request_signature,title,workflow_name,input_summary FROM ai_task "

func (m *customAiTaskModel) FindOwned(ctx context.Context, id, ownerId int64) (*TaskRow, error) {
	var row TaskRow
	err := m.QueryRowNoCacheCtx(ctx, &row, taskSelect+"WHERE id=? AND created_by=?", id, ownerId)
	return &row, err
}

func (m *customAiTaskModel) FindSummaryOwned(ctx context.Context, id, ownerId int64) (*TaskRow, error) {
	var row TaskRow
	err := m.QueryRowNoCacheCtx(ctx, &row, taskSummarySelect+"WHERE id=? AND created_by=?", id, ownerId)
	return &row, err
}

func (m *customAiTaskModel) CheckRunningOwned(ctx context.Context, id, ownerId int64) (bool, error) {
	var status string
	err := m.QueryRowNoCacheCtx(ctx, &status, "SELECT status FROM ai_task WHERE id=? AND created_by=?", id, ownerId)
	return status == "running", err
}

func (m *customAiTaskModel) ListOwned(ctx context.Context, ownerId, page, pageSize int64, status string, workflowId int64) ([]*TaskRow, int64, error) {
	return m.ListSearchOwned(ctx, ownerId, page, pageSize, status, workflowId, "")
}

func (m *customAiTaskModel) ListSearchOwned(ctx context.Context, ownerId, page, pageSize int64, status string, workflowId int64, query string) ([]*TaskRow, int64, error) {
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
	if query != "" {
		where += " AND (title LIKE ? ESCAPE '!' OR workflow_name LIKE ? ESCAPE '!' OR input_summary LIKE ? ESCAPE '!')"
		pattern := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(query) + "%"
		args = append(args, pattern, pattern, pattern)
	}
	var count struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM ai_task WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	rows := make([]*TaskRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, taskSummarySelect+"WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, pageSize, (page-1)*pageSize)...)
	return rows, count.Total, err
}

func (m *customAiTaskModel) ClaimApproval(ctx context.Context, id, ownerId, expectedStep int64) error {
	return m.changeState(ctx, id, "UPDATE ai_task SET status='running',approved_by=? WHERE id=? AND created_by=? AND status='pending_approval' AND next_step=? AND workflow_snapshot IS NOT NULL", ownerId, id, ownerId, expectedStep)
}

func (m *customAiTaskModel) RejectOwned(ctx context.Context, id, ownerId, expectedStep int64, reason string) error {
	return m.changeState(ctx, id, "UPDATE ai_task SET status='rejected',approved_by=?,error=?,completed_at=NOW(),step_records=CASE WHEN step_records IS NOT NULL AND JSON_LENGTH(step_records)>next_step THEN JSON_SET(step_records,CONCAT('$[',next_step,'].status'),'cancelled',CONCAT('$[',next_step,'].error'),?,CONCAT('$[',next_step,'].completed_at'),?) ELSE step_records END WHERE id=? AND created_by=? AND status='pending_approval' AND next_step=?", ownerId, reason, reason, time.Now().UnixMilli(), id, ownerId, expectedStep)
}

func (m *customAiTaskModel) Advance(ctx context.Context, id, ownerId, expectedStep, nextStep int64, status, result, errorMessage string) error {
	completed := "NULL"
	if status == "succeeded" || status == "failed" {
		completed = "NOW()"
	}
	return m.changeState(ctx, id, "UPDATE ai_task SET status=?,next_step=?,result=?,error=?,completed_at="+completed+" WHERE id=? AND created_by=? AND status='running' AND next_step=?", status, nextStep, result, errorMessage, id, ownerId, expectedStep)
}

func (m *customAiTaskModel) changeState(ctx context.Context, id int64, query string, args ...any) error {
	result, err := m.conn.ExecCtx(ctx, query, args...)
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
	if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiTaskIdPrefix, id)); err != nil {
		return ErrTaskCommitted
	}
	return nil
}
