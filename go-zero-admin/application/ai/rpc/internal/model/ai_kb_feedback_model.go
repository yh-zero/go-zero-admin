package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiKbFeedbackModel = (*customAiKbFeedbackModel)(nil)

type (
	// AiKbFeedbackModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiKbFeedbackModel.
	AiKbFeedbackModel interface {
		aiKbFeedbackModel
		SetOwned(ctx context.Context, ownerId, expectedRevision int64, row *KbFeedbackRow, step WorkflowStepRecord) (*KbFeedbackRow, error)
		FindOwned(ctx context.Context, ownerId, kbId, id int64) (*KbFeedbackRow, error)
		ListTaskOwned(ctx context.Context, ownerId, taskId int64) ([]*KbFeedbackRow, error)
		ListOwned(ctx context.Context, ownerId, kbId, page, pageSize int64, status string) ([]*KbFeedbackRow, int64, error)
		ReviewOwned(ctx context.Context, ownerId, kbId, id, expectedRevision int64, status, review string) (*KbFeedbackRow, error)
		FindReadySourcesOwned(ctx context.Context, ownerId, kbId int64, chunkIds []int64) ([]*FeedbackSourceRow, error)
	}

	customAiKbFeedbackModel struct {
		*defaultAiKbFeedbackModel
		conn sqlx.SqlConn
	}
)

// NewAiKbFeedbackModel returns a model for the database table.
func NewAiKbFeedbackModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiKbFeedbackModel {
	return &customAiKbFeedbackModel{
		defaultAiKbFeedbackModel: newAiKbFeedbackModel(conn, c, opts...),
		conn:                     conn,
	}
}

var ErrFeedbackConflict = errors.New("feedback revision changed")
var ErrFeedbackCommitted = errors.New("feedback committed; cache invalidation failed")

type FeedbackSourceRow struct {
	ChunkId      int64  `db:"chunk_id"`
	DocId        int64  `db:"doc_id"`
	KbId         int64  `db:"kb_id"`
	Title        string `db:"title"`
	Version      int64  `db:"version"`
	Content      string `db:"content"`
	ChunkContent string `db:"chunk_content"`
	Seq          int64  `db:"seq"`
}

// 同一条查询采集分块和正文版本，避免两次读取跨过一次文档替换。
func (m *customAiKbFeedbackModel) FindReadySourcesOwned(ctx context.Context, ownerId, kbId int64, chunkIds []int64) ([]*FeedbackSourceRow, error) {
	rows := make([]*FeedbackSourceRow, 0)
	if len(chunkIds) == 0 {
		return rows, nil
	}
	if len(chunkIds) > 8 {
		return nil, ErrFeedbackConflict
	}
	args := []any{ownerId, kbId}
	for _, id := range chunkIds {
		args = append(args, id)
	}
	err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT c.id AS chunk_id,c.doc_id,c.kb_id,d.title,d.version,d.content,c.content AS chunk_content,c.seq FROM ai_kb_chunk c JOIN ai_kb_doc d ON d.id=c.doc_id AND d.kb_id=c.kb_id AND d.status='ready' AND d.deleted_at IS NULL JOIN ai_kb k ON k.id=c.kb_id AND k.owner_id=? AND k.deleted_at IS NULL WHERE c.kb_id=? AND c.status='ready' AND c.id IN ("+strings.TrimSuffix(strings.Repeat("?,", len(chunkIds)), ",")+")", args...)
	return rows, err
}

type KbFeedbackRow struct {
	Id         int64  `db:"id"`
	OwnerId    int64  `db:"owner_id"`
	KbId       int64  `db:"kb_id"`
	TaskId     int64  `db:"task_id"`
	StepIndex  int64  `db:"step_index"`
	Feedback   string `db:"feedback"`
	Note       string `db:"note"`
	Status     string `db:"status"`
	Revision   int64  `db:"revision"`
	Question   string `db:"question"`
	Answer     string `db:"answer"`
	Snapshot   string `db:"snapshot"`
	Review     string `db:"review"`
	ReviewedAt int64  `db:"reviewed_at"`
	CreatedAt  int64  `db:"created_at"`
	UpdatedAt  int64  `db:"updated_at"`
}

const feedbackFields = "f.id,f.owner_id,f.kb_id,f.task_id,f.step_index,f.feedback,f.note,f.status,f.revision,f.question,%s AS answer,%s AS snapshot,%s AS review,COALESCE(UNIX_TIMESTAMP(f.reviewed_at),0) AS reviewed_at,UNIX_TIMESTAMP(f.created_at) AS created_at,UNIX_TIMESTAMP(f.updated_at) AS updated_at "

var feedbackSelect = "SELECT " + fmt.Sprintf(feedbackFields, "f.answer", "CAST(f.snapshot AS CHAR)", "COALESCE(CAST(f.review AS CHAR),'')") + "FROM ai_kb_feedback f "
var feedbackSummarySelect = "SELECT " + fmt.Sprintf(feedbackFields, "''", "''", "''") + "FROM ai_kb_feedback f "

const feedbackOwnedJoin = "JOIN ai_kb k ON k.id=f.kb_id AND k.owner_id=f.owner_id AND k.deleted_at IS NULL "

func (m *customAiKbFeedbackModel) FindOwned(ctx context.Context, ownerId, kbId, id int64) (*KbFeedbackRow, error) {
	var row KbFeedbackRow
	err := m.QueryRowNoCacheCtx(ctx, &row, feedbackSelect+feedbackOwnedJoin+"WHERE f.id=? AND f.owner_id=? AND f.kb_id=?", id, ownerId, kbId)
	return &row, err
}

func (m *customAiKbFeedbackModel) ListTaskOwned(ctx context.Context, ownerId, taskId int64) ([]*KbFeedbackRow, error) {
	rows := make([]*KbFeedbackRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, feedbackSelect+feedbackOwnedJoin+"WHERE f.owner_id=? AND f.task_id=? ORDER BY f.step_index LIMIT 32", ownerId, taskId)
	return rows, err
}

func (m *customAiKbFeedbackModel) ListOwned(ctx context.Context, ownerId, kbId, page, pageSize int64, status string) ([]*KbFeedbackRow, int64, error) {
	where := "WHERE f.owner_id=? AND f.kb_id=?"
	args := []any{ownerId, kbId}
	if status != "all" {
		where += " AND f.status=?"
		args = append(args, status)
	}
	var total int64
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) FROM ai_kb_feedback f "+feedbackOwnedJoin+where, args...); err != nil {
		return nil, 0, err
	}
	rows := make([]*KbFeedbackRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, feedbackSummarySelect+feedbackOwnedJoin+where+" ORDER BY f.id DESC LIMIT ? OFFSET ?", append(args, pageSize, (page-1)*pageSize)...)
	return rows, total, err
}

func (m *customAiKbFeedbackModel) SetOwned(ctx context.Context, ownerId, expectedRevision int64, draft *KbFeedbackRow, step WorkflowStepRecord) (*KbFeedbackRow, error) {
	var row KbFeedbackRow
	changed := false
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := lockKb(ctx, session, draft.KbId, ownerId); err != nil {
			return err
		}
		var records string
		if err := session.QueryRowCtx(ctx, &records, "SELECT COALESCE(CAST(step_records AS CHAR),'[]') FROM ai_task WHERE id=? AND created_by=? FOR UPDATE", draft.TaskId, ownerId); err != nil {
			return err
		}
		actual, err := ReadWorkflowStepRecords(records)
		if err != nil || draft.StepIndex < 0 || draft.StepIndex >= int64(len(actual)) {
			return ErrFeedbackConflict
		}
		current := actual[draft.StepIndex]
		if current.Status != "succeeded" || current.Tool != "kb_analyze" || current.Input != step.Input || current.Output != step.Output {
			return ErrFeedbackConflict
		}
		err = session.QueryRowCtx(ctx, &row, feedbackSelect+"WHERE f.owner_id=? AND f.task_id=? AND f.step_index=? FOR UPDATE", ownerId, draft.TaskId, draft.StepIndex)
		switch {
		case errors.Is(err, ErrNotFound):
			if expectedRevision != 0 {
				return ErrFeedbackConflict
			}
			_, err = session.ExecCtx(ctx, "INSERT INTO ai_kb_feedback (owner_id,kb_id,task_id,step_index,feedback,note,question,answer,snapshot) VALUES (?,?,?,?,?,?,?,?,?)", ownerId, draft.KbId, draft.TaskId, draft.StepIndex, draft.Feedback, draft.Note, draft.Question, draft.Answer, draft.Snapshot)
		case err != nil:
			return err
		case row.Feedback == draft.Feedback && row.Note == draft.Note:
			return nil
		default:
			if row.Revision != expectedRevision {
				return ErrFeedbackConflict
			}
			_, err = session.ExecCtx(ctx, "UPDATE ai_kb_feedback SET feedback=?,note=?,status='pending',review=NULL,reviewed_at=NULL,revision=revision+1,updated_at=NOW() WHERE id=? AND revision=?", draft.Feedback, draft.Note, row.Id, expectedRevision)
		}
		if err != nil {
			return err
		}
		changed = true
		return session.QueryRowCtx(ctx, &row, feedbackSelect+"WHERE f.owner_id=? AND f.task_id=? AND f.step_index=?", ownerId, draft.TaskId, draft.StepIndex)
	})
	if err != nil {
		return nil, err
	}
	if changed && m.invalidateFeedback(ctx, &row) != nil {
		return nil, ErrFeedbackCommitted
	}
	return &row, nil
}

func (m *customAiKbFeedbackModel) ReviewOwned(ctx context.Context, ownerId, kbId, id, expectedRevision int64, status, review string) (*KbFeedbackRow, error) {
	var row KbFeedbackRow
	changed := false
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := lockKb(ctx, session, kbId, ownerId); err != nil {
			return err
		}
		if err := session.QueryRowCtx(ctx, &row, feedbackSelect+"WHERE f.id=? AND f.kb_id=? AND f.owner_id=? FOR UPDATE", id, kbId, ownerId); err != nil {
			return err
		}
		if row.Status == status && feedbackJSONEqual(row.Review, review) {
			return nil
		}
		if row.Revision != expectedRevision {
			return ErrFeedbackConflict
		}
		if _, err := session.ExecCtx(ctx, "UPDATE ai_kb_feedback SET status=?,review=?,reviewed_at=NOW(),revision=revision+1,updated_at=NOW() WHERE id=? AND revision=?", status, review, id, expectedRevision); err != nil {
			return err
		}
		changed = true
		return session.QueryRowCtx(ctx, &row, feedbackSelect+"WHERE f.id=?", id)
	})
	if err != nil {
		return nil, err
	}
	if changed && m.invalidateFeedback(ctx, &row) != nil {
		return nil, ErrFeedbackCommitted
	}
	return &row, nil
}

func feedbackJSONEqual(a, b string) bool {
	var left, right any
	return json.Unmarshal([]byte(a), &left) == nil && json.Unmarshal([]byte(b), &right) == nil && reflect.DeepEqual(left, right)
}

func (m *customAiKbFeedbackModel) invalidateFeedback(ctx context.Context, row *KbFeedbackRow) error {
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiKbFeedbackIdPrefix, row.Id), fmt.Sprintf("%s%d:%d:%d", cacheGozeroAdminAiKbFeedbackOwnerIdTaskIdStepIndexPrefix, row.OwnerId, row.TaskId, row.StepIndex))
}
