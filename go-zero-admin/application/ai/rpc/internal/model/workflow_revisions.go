package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var ErrWorkflowRevisionConflict = errors.New("workflow revision changed")
var ErrWorkflowCommitted = errors.New("workflow committed but cache invalidation failed")

type WorkflowRevisionRow struct {
	Id              int64  `db:"id"`
	WorkflowId      int64  `db:"workflow_id"`
	OwnerId         int64  `db:"owner_id"`
	RevisionSeq     int64  `db:"revision_seq"`
	Name            string `db:"name"`
	Definition      string `db:"definition"`
	IntervalSeconds int64  `db:"interval_seconds"`
	Enabled         int64  `db:"enabled"`
	StepCount       int64  `db:"step_count"`
	CreatedAt       int64  `db:"created_at"`
}

const workflowRevisionFields = "h.id,h.workflow_id,h.owner_id,h.revision_seq,h.name,%s AS definition,h.interval_seconds,h.enabled,COALESCE(JSON_LENGTH(h.definition,'$.steps'),0) AS step_count,UNIX_TIMESTAMP(h.created_at) AS created_at FROM ai_workflow_revision h "
const workflowRevisionOwnerJoin = "JOIN ai_workflow w ON w.id=h.workflow_id AND w.created_by=h.owner_id AND w.deleted_at IS NULL "

var workflowRevisionSelect = "SELECT " + fmt.Sprintf(workflowRevisionFields, "CAST(h.definition AS CHAR)")
var workflowRevisionSummarySelect = "SELECT " + fmt.Sprintf(workflowRevisionFields, "''")

func (m *customAiWorkflowModel) FindRevisionOwned(ctx context.Context, id, ownerId, revision int64) (*WorkflowRevisionRow, error) {
	var row WorkflowRevisionRow
	err := m.QueryRowNoCacheCtx(ctx, &row, workflowRevisionSelect+workflowRevisionOwnerJoin+"WHERE h.workflow_id=? AND h.owner_id=? AND h.revision_seq=?", id, ownerId, revision)
	return &row, err
}

func (m *customAiWorkflowModel) ListRevisionsOwned(ctx context.Context, id, ownerId, page, pageSize int64) ([]*WorkflowRevisionRow, int64, error) {
	rows := make([]*WorkflowRevisionRow, 0)
	var total int64
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var owned int64
		if err := session.QueryRowCtx(ctx, &owned, "SELECT id FROM ai_workflow WHERE id=? AND created_by=? AND deleted_at IS NULL FOR SHARE", id, ownerId); err != nil {
			return err
		}
		if err := session.QueryRowCtx(ctx, &total, "SELECT COUNT(*) FROM ai_workflow_revision WHERE workflow_id=? AND owner_id=?", id, ownerId); err != nil {
			return err
		}
		return session.QueryRowsCtx(ctx, &rows, workflowRevisionSummarySelect+"WHERE h.workflow_id=? AND h.owner_id=? ORDER BY h.revision_seq DESC LIMIT ? OFFSET ?", id, ownerId, pageSize, (page-1)*pageSize)
	})
	return rows, total, err
}

// JSON列读取时会重排键和添加空白，不能把排版差异误归档成一次业务修改；保留数字原精度。
func sameWorkflowDefinition(left, right string) bool {
	canonical := func(raw string) (string, error) {
		if !json.Valid([]byte(raw)) {
			return "", errors.New("invalid workflow definition")
		}
		decoder := json.NewDecoder(strings.NewReader(raw))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return "", err
		}
		encoded, err := json.Marshal(value)
		return string(encoded), err
	}
	a, errA := canonical(left)
	b, errB := canonical(right)
	return errA == nil && errB == nil && a == b
}

func (m *customAiWorkflowModel) UpdateOwned(ctx context.Context, id, ownerId, expectedRevision int64, patch WorkflowPatch) (*WorkflowRow, error) {
	return m.writeRevisionOwned(ctx, id, ownerId, expectedRevision, patch, 0)
}

func (m *customAiWorkflowModel) RestoreRevisionOwned(ctx context.Context, id, ownerId, revision, expectedRevision int64) (*WorkflowRow, error) {
	if revision < 1 {
		return nil, ErrNotFound
	}
	return m.writeRevisionOwned(ctx, id, ownerId, expectedRevision, WorkflowPatch{}, revision)
}

func (m *customAiWorkflowModel) writeRevisionOwned(ctx context.Context, id, ownerId, expectedRevision int64, patch WorkflowPatch, restoreRevision int64) (*WorkflowRow, error) {
	var result *WorkflowRow
	changed := false
	var archivedId, archivedRevision int64
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var current WorkflowRow
		if err := session.QueryRowCtx(ctx, &current, workflowSelect+"WHERE id=? AND created_by=? AND deleted_at IS NULL FOR UPDATE", id, ownerId); err != nil {
			return err
		}
		if expectedRevision < 1 || current.RevisionSeq != expectedRevision {
			return ErrWorkflowRevisionConflict
		}
		if restoreRevision > 0 {
			var old WorkflowRevisionRow
			if err := session.QueryRowCtx(ctx, &old, workflowRevisionSelect+"WHERE h.workflow_id=? AND h.owner_id=? AND h.revision_seq=?", id, ownerId, restoreRevision); err != nil {
				return err
			}
			patch = WorkflowPatch{Name: &old.Name, Definition: &old.Definition, IntervalSeconds: &old.IntervalSeconds, Enabled: &old.Enabled}
		}
		next := current
		if patch.Name != nil {
			next.Name = *patch.Name
		}
		if patch.Definition != nil {
			next.Definition = *patch.Definition
		}
		if patch.IntervalSeconds != nil {
			next.IntervalSeconds = *patch.IntervalSeconds
		}
		if patch.Enabled != nil {
			next.Enabled = *patch.Enabled
		}
		result = &current
		if restoreRevision == 0 && next.Name == current.Name && next.Enabled == current.Enabled && next.IntervalSeconds == current.IntervalSeconds && sameWorkflowDefinition(next.Definition, current.Definition) {
			return nil
		}
		if current.RevisionSeq >= math.MaxInt64 {
			return errors.New("workflow revision exhausted")
		}
		archive, err := session.ExecCtx(ctx, "INSERT INTO ai_workflow_revision (workflow_id,owner_id,revision_seq,name,definition,interval_seconds,enabled) VALUES (?,?,?,?,?,?,?)", id, ownerId, current.RevisionSeq, current.Name, current.Definition, current.IntervalSeconds, current.Enabled)
		if err != nil {
			return err
		}
		archivedId, err = archive.LastInsertId()
		if err != nil {
			return err
		}
		archivedRevision = current.RevisionSeq
		schedule := ""
		if patch.IntervalSeconds != nil || restoreRevision > 0 {
			schedule = ",next_run_at=NULL"
		}
		updated, err := session.ExecCtx(ctx, "UPDATE ai_workflow SET name=?,definition=?,interval_seconds=?,enabled=?,revision_seq=revision_seq+1"+schedule+" WHERE id=? AND created_by=? AND revision_seq=? AND deleted_at IS NULL", next.Name, next.Definition, next.IntervalSeconds, next.Enabled, id, ownerId, expectedRevision)
		if err != nil {
			return err
		}
		affected, err := updated.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return ErrWorkflowRevisionConflict
		}
		var committed WorkflowRow
		if err := session.QueryRowCtx(ctx, &committed, workflowSelect+"WHERE id=? AND created_by=? AND deleted_at IS NULL", id, ownerId); err != nil {
			return err
		}
		result, changed = &committed, true
		return nil
	})
	if err != nil {
		return nil, err
	}
	if changed {
		if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiWorkflowIdPrefix, id),
			fmt.Sprintf("%s%d", cacheGozeroAdminAiWorkflowRevisionIdPrefix, archivedId),
			fmt.Sprintf("%s%d:%d", cacheGozeroAdminAiWorkflowRevisionWorkflowIdRevisionSeqPrefix, id, archivedRevision)); err != nil {
			return result, ErrWorkflowCommitted
		}
	}
	return result, nil
}
