package model

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

const evaluationFields = "e.id,e.owner_id,e.kb_id,e.request_id,e.request_signature,e.dataset_key,e.dataset_sha256,e.group_key,e.status,e.case_count,e.completed_count,e.failed_count,e.stop_requested,e.corpus_verified,e.error,%s AS dataset_snapshot,%s AS bindings,%s AS run_context,%s AS observations,e.lease_token,COALESCE(UNIX_TIMESTAMP(e.lease_expires_at),0) AS lease_expires_at,COALESCE(UNIX_TIMESTAMP(e.started_at),0) AS started_at,COALESCE(UNIX_TIMESTAMP(e.completed_at),0) AS completed_at,UNIX_TIMESTAMP(e.created_at) AS created_at,UNIX_TIMESTAMP(e.updated_at) AS updated_at,UNIX_TIMESTAMP(NOW()) AS database_time "

var evaluationSelect = "SELECT " + fmt.Sprintf(evaluationFields, "e.dataset_snapshot", "CAST(e.bindings AS CHAR)", "CAST(e.run_context AS CHAR)", "CAST(e.observations AS CHAR)") + "FROM ai_kb_evaluation e "
var evaluationSummarySelect = "SELECT " + fmt.Sprintf(evaluationFields, "''", "''", "''", "''") + "FROM ai_kb_evaluation e "

const evaluationKbJoin = "JOIN ai_kb k ON k.id=e.kb_id AND k.owner_id=e.owner_id AND k.deleted_at IS NULL "

func (m *customAiKbEvaluationModel) FindRequestOwned(ctx context.Context, ownerId int64, requestId string) (*KbEvaluationRow, error) {
	var row KbEvaluationRow
	err := m.QueryRowNoCacheCtx(ctx, &row, evaluationSelect+"WHERE e.owner_id=? AND e.request_id=?", ownerId, requestId)
	if err == nil {
		err = compactEvaluationRow(&row)
	}
	return &row, err
}
func (m *customAiKbEvaluationModel) FindOwned(ctx context.Context, ownerId, kbId, id int64) (*KbEvaluationRow, error) {
	var row KbEvaluationRow
	err := m.QueryRowNoCacheCtx(ctx, &row, evaluationSelect+evaluationKbJoin+"WHERE e.id=? AND e.owner_id=? AND e.kb_id=?", id, ownerId, kbId)
	if err == nil {
		err = compactEvaluationRow(&row)
	}
	return &row, err
}

// MySQL JSON的文本输出会添加空白；统一压紧，避免读回额外空白突破观测预算。
func compactEvaluationRow(row *KbEvaluationRow) error {
	for _, field := range []*string{&row.Bindings, &row.Context, &row.Observations} {
		if *field == "" {
			continue
		}
		var compact bytes.Buffer
		if json.Compact(&compact, []byte(*field)) != nil {
			return ErrEvaluationState
		}
		*field = compact.String()
	}
	if len(row.Observations) > 1<<20 {
		return ErrEvaluationState
	}
	return nil
}
func (m *customAiKbEvaluationModel) ListOwned(ctx context.Context, ownerId, kbId, page, size int64, status, requestId string) ([]*KbEvaluationRow, int64, error) {
	where := "WHERE e.owner_id=? AND e.kb_id=?"
	args := []any{ownerId, kbId}
	if status != "all" {
		where += " AND e.status=?"
		args = append(args, status)
	}
	if requestId != "" {
		where += " AND e.request_id=?"
		args = append(args, requestId)
	}
	var total int64
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) FROM ai_kb_evaluation e "+evaluationKbJoin+where, args...); err != nil {
		return nil, 0, err
	}
	rows := make([]*KbEvaluationRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, evaluationSummarySelect+evaluationKbJoin+where+" ORDER BY e.id DESC LIMIT ? OFFSET ?", append(args, size, (page-1)*size)...)
	return rows, total, err
}
func (m *customAiKbEvaluationModel) invalidateEvaluation(ctx context.Context, row *KbEvaluationRow) error {
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiKbEvaluationIdPrefix, row.Id), fmt.Sprintf("%s%d:%s", cacheGozeroAdminAiKbEvaluationOwnerIdRequestIdPrefix, row.OwnerId, row.RequestId))
}
func (m *customAiKbEvaluationModel) CreateOwned(ctx context.Context, draft *KbEvaluationRow) (*KbEvaluationRow, bool, error) {
	if draft == nil || draft.OwnerId <= 0 || draft.KbId <= 0 || draft.CaseCount < 1 || draft.CaseCount > 30 || len(draft.Snapshot) > 256<<10 || !json.Valid([]byte(draft.Snapshot)) || !json.Valid([]byte(draft.Bindings)) || !json.Valid([]byte(draft.Context)) {
		return nil, false, ErrEvaluationState
	}
	observations, err := decodeEvaluationObservations(draft.Observations)
	if err != nil || int64(len(observations)) != draft.CaseCount {
		return nil, false, ErrEvaluationState
	}
	for _, item := range observations {
		if item.Status != "waiting" {
			return nil, false, ErrEvaluationState
		}
	}
	existing := func() (*KbEvaluationRow, error) {
		row, err := m.FindRequestOwned(ctx, draft.OwnerId, draft.RequestId)
		if err != nil {
			return nil, err
		}
		if row.KbId != draft.KbId || row.RequestSignature != draft.RequestSignature {
			return nil, ErrEvaluationConflict
		}
		return row, nil
	}
	if row, err := existing(); err == nil {
		return row, false, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, false, err
	}
	row := *draft
	err = m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := lockKb(ctx, session, row.KbId, row.OwnerId); err != nil {
			return err
		}
		result, err := session.ExecCtx(ctx, "INSERT INTO ai_kb_evaluation (owner_id,kb_id,request_id,request_signature,dataset_key,dataset_sha256,group_key,case_count,dataset_snapshot,bindings,run_context,observations) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)", row.OwnerId, row.KbId, row.RequestId, row.RequestSignature, row.DatasetKey, row.DatasetSHA, row.GroupKey, row.CaseCount, row.Snapshot, row.Bindings, row.Context, row.Observations)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		row.Id = id
		return err
	})
	if err != nil {
		if old, readErr := existing(); readErr == nil {
			return old, false, nil
		} else if errors.Is(readErr, ErrEvaluationConflict) {
			return nil, false, readErr
		}
		return nil, false, err
	}
	fresh, err := m.FindOwned(ctx, row.OwnerId, row.KbId, row.Id)
	if err != nil {
		return &row, true, ErrEvaluationCommitted
	}
	if m.invalidateEvaluation(ctx, fresh) != nil {
		return fresh, true, ErrEvaluationCommitted
	}
	return fresh, true, nil
}
func evaluationLeaseValid(row *KbEvaluationRow, token string) bool {
	return row.Status == "running" && token != "" && row.LeaseToken == token && row.LeaseExpiresAt > row.DatabaseTime
}

// 同一父知识库锁和行锁保护停止、租约与观测；事务内不调用模型或RPC。
func (m *customAiKbEvaluationModel) mutateEvaluation(ctx context.Context, ownerId, kbId, id int64, leaseToken string, change func(*KbEvaluationRow) (bool, error)) (*KbEvaluationRow, error) {
	var row KbEvaluationRow
	changed := false
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := lockKb(ctx, session, kbId, ownerId); err != nil {
			return err
		}
		if err := session.QueryRowCtx(ctx, &row, evaluationSelect+"WHERE e.id=? AND e.owner_id=? AND e.kb_id=? FOR UPDATE", id, ownerId, kbId); err != nil {
			return err
		}
		if err := compactEvaluationRow(&row); err != nil {
			return err
		}
		var err error
		changed, err = change(&row)
		if err != nil || !changed {
			return err
		}
		if len(row.Error) > 500 || !utf8.ValidString(row.Error) {
			return ErrEvaluationState
		}
		query := "UPDATE ai_kb_evaluation SET status=?,completed_count=?,failed_count=?,stop_requested=?,corpus_verified=?,error=?,observations=?,lease_token=?,lease_expires_at=FROM_UNIXTIME(NULLIF(?,0)),started_at=FROM_UNIXTIME(NULLIF(?,0)),completed_at=FROM_UNIXTIME(NULLIF(?,0)),updated_at=NOW() WHERE id=? AND owner_id=? AND kb_id=?"
		args := []any{row.Status, row.CompletedCount, row.FailedCount, row.StopRequested, row.CorpusVerified, row.Error, row.Observations, row.LeaseToken, row.LeaseExpiresAt, row.StartedAt, row.CompletedAt, id, ownerId, kbId}
		if leaseToken != "" {
			query += " AND status='running' AND lease_token=? AND lease_expires_at>NOW()"
			args = append(args, leaseToken)
		}
		result, err := session.ExecCtx(ctx, query, args...)
		if err != nil {
			return err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if leaseToken != "" && count != 1 {
			return ErrEvaluationState
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if changed && m.invalidateEvaluation(ctx, &row) != nil {
		return &row, ErrEvaluationCommitted
	}
	fresh, readErr := m.FindOwned(ctx, ownerId, kbId, id)
	if changed && readErr != nil {
		return &row, ErrEvaluationCommitted
	}
	return fresh, readErr
}
func (m *customAiKbEvaluationModel) ActivateOwned(ctx context.Context, owner, kb, id int64, token string) (*KbEvaluationRow, error) {
	return m.mutateEvaluation(ctx, owner, kb, id, "", func(row *KbEvaluationRow) (bool, error) {
		if row.Status != "pending" || row.StopRequested || token == "" {
			return false, ErrEvaluationState
		}
		row.Status = "running"
		row.LeaseToken = token
		row.LeaseExpiresAt = row.DatabaseTime + 120
		row.StartedAt = row.DatabaseTime
		return true, nil
	})
}
func (m *customAiKbEvaluationModel) HeartbeatOwned(ctx context.Context, owner, kb, id int64, token string) error {
	_, err := m.mutateEvaluation(ctx, owner, kb, id, token, func(row *KbEvaluationRow) (bool, error) {
		if !evaluationLeaseValid(row, token) {
			return false, ErrEvaluationState
		}
		row.LeaseExpiresAt = row.DatabaseTime + 120
		return true, nil
	})
	return err
}
func (m *customAiKbEvaluationModel) TryStartCase(ctx context.Context, owner, kb, id, index int64, token string) (*KbEvaluationRow, error) {
	return m.mutateEvaluation(ctx, owner, kb, id, token, func(row *KbEvaluationRow) (bool, error) {
		if !evaluationLeaseValid(row, token) {
			return false, ErrEvaluationState
		}
		if row.StopRequested {
			return false, ErrEvaluationStopped
		}
		items, err := decodeEvaluationObservations(row.Observations)
		if err != nil || index < 0 || index >= int64(len(items)) {
			return false, ErrEvaluationState
		}
		next := items[index]
		next.Status = "running"
		next.Stage = "search"
		raw, completed, failed, err := replaceEvaluationObservation(row.Observations, index, next, true)
		if err != nil {
			return false, err
		}
		row.Observations = raw
		row.CompletedCount = completed
		row.FailedCount = failed
		return true, nil
	})
}
func (m *customAiKbEvaluationModel) SaveCase(ctx context.Context, owner, kb, id, index int64, token string, next EvaluationObservation) (*KbEvaluationRow, error) {
	return m.mutateEvaluation(ctx, owner, kb, id, token, func(row *KbEvaluationRow) (bool, error) {
		if !evaluationLeaseValid(row, token) {
			return false, ErrEvaluationState
		}
		raw, completed, failed, err := replaceEvaluationObservation(row.Observations, index, next, false)
		if err != nil {
			return false, err
		}
		row.Observations = raw
		row.CompletedCount = completed
		row.FailedCount = failed
		return true, nil
	})
}
func (m *customAiKbEvaluationModel) FinishOwned(ctx context.Context, owner, kb, id int64, token, status string, verified bool, message string) (*KbEvaluationRow, error) {
	return m.mutateEvaluation(ctx, owner, kb, id, token, func(row *KbEvaluationRow) (bool, error) {
		if !evaluationLeaseValid(row, token) {
			return false, ErrEvaluationState
		}
		switch status {
		case "succeeded":
			if row.CompletedCount != row.CaseCount || row.FailedCount != 0 || !verified {
				return false, ErrEvaluationState
			}
		case "cancelled", "failed", "unknown":
		default:
			return false, ErrEvaluationState
		}
		items, err := decodeEvaluationObservations(row.Observations)
		if err != nil {
			return false, err
		}
		for index, item := range items {
			if item.Status == "running" {
				if status != "unknown" && status != "failed" {
					return false, ErrEvaluationState
				}
				item.Status = "unknown"
				item.Stage = "unknown"
				item.Error = "执行过程未确认，请按原请求标识核对"
				raw, completed, failed, err := replaceEvaluationObservation(row.Observations, int64(index), item, false)
				if err != nil {
					return false, err
				}
				row.Observations = raw
				row.CompletedCount = completed
				row.FailedCount = failed
			}
		}
		row.Status = status
		row.CorpusVerified = verified
		row.Error = message
		row.CompletedAt = row.DatabaseTime
		row.LeaseToken = ""
		row.LeaseExpiresAt = 0
		return true, nil
	})
}
func (m *customAiKbEvaluationModel) StopOwned(ctx context.Context, owner, kb, id int64) (*KbEvaluationRow, error) {
	return m.mutateEvaluation(ctx, owner, kb, id, "", func(row *KbEvaluationRow) (bool, error) {
		if row.Status != "pending" && row.Status != "running" {
			return false, nil
		}
		if row.StopRequested {
			return false, nil
		}
		row.StopRequested = true
		if row.Status == "pending" {
			row.Status = "cancelled"
			row.CompletedAt = row.DatabaseTime
			row.Error = "已停止，未开始的题目不会执行"
		}
		return true, nil
	})
}
func (m *customAiKbEvaluationModel) ExpireOwned(ctx context.Context, owner, kb, id int64) (*KbEvaluationRow, error) {
	return m.mutateEvaluation(ctx, owner, kb, id, "", func(row *KbEvaluationRow) (bool, error) {
		if row.Status != "running" && row.Status != "pending" {
			return false, nil
		}
		if row.Status == "running" && row.LeaseExpiresAt > row.DatabaseTime {
			return false, nil
		}
		if row.Status == "pending" && row.UpdatedAt > row.DatabaseTime-120 {
			return false, nil
		}
		items, err := decodeEvaluationObservations(row.Observations)
		if err != nil {
			return false, err
		}
		for index, item := range items {
			if item.Status == "running" {
				item.Status = "unknown"
				item.Stage = "unknown"
				item.Error = "执行租约过期，结果可能未知，请按原请求标识核对"
				raw, completed, failed, err := replaceEvaluationObservation(row.Observations, int64(index), item, false)
				if err != nil {
					return false, err
				}
				row.Observations = raw
				row.CompletedCount = completed
				row.FailedCount = failed
			}
		}
		row.Status = "unknown"
		row.CorpusVerified = false
		row.Error = "评测执行中断或租约过期，不会自动重放"
		row.CompletedAt = row.DatabaseTime
		row.LeaseToken = ""
		row.LeaseExpiresAt = 0
		return true, nil
	})
}
func (m *customAiKbEvaluationModel) ReadCorpusOwned(ctx context.Context, owner, kb int64) ([]EvaluationCorpusDocument, error) {
	rows := make([]EvaluationCorpusDocument, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT d.id,d.kb_id,d.title,d.version,d.status,d.embedding_model,SHA2(d.content,256) AS content_sha256 FROM ai_kb_doc d JOIN ai_kb k ON k.id=d.kb_id AND k.owner_id=? AND k.deleted_at IS NULL WHERE d.kb_id=? AND d.deleted_at IS NULL ORDER BY d.id", owner, kb)
	return rows, err
}
