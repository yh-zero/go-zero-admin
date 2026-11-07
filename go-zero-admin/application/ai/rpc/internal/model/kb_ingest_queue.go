package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var ErrKbRequestConflict = errors.New("kb request identity conflict")
var ErrKbBatchStopped = errors.New("kb import stopped")
var ErrKbBatchFull = errors.New("kb import has twenty items")
var ErrKbGenerationChanged = fmt.Errorf("kb ingest generation changed: %w", ErrKbDocState)

// KbIngestQueue 的查询绕过缓存；队列与文档代次在同一 MySQL 事务中提交。
type KbIngestQueue struct {
	conn sqlx.SqlConn
	docs AiKbDocModel
}

type KbEnqueueRequest struct {
	KbId, DocId, OwnerId, ExpectedVersion int64
	RequestId, BatchId, Operation         string
	Title, Content, EmbeddingModel        string
}

type KbIngestJobRow struct {
	Id          int64  `db:"id"`
	BatchId     int64  `db:"batch_id"`
	KbId        int64  `db:"kb_id"`
	DocId       int64  `db:"doc_id"`
	OwnerId     int64  `db:"owner_id"`
	DocVersion  int64  `db:"doc_version"`
	RequestId   string `db:"request_id"`
	RequestHash string `db:"request_hash"`
	Operation   string `db:"operation"`
	Status      string `db:"status"`
	Error       string `db:"error"`
	ChunkIds    string `db:"chunk_ids"`
	OldChunkIds string `db:"old_chunk_ids"`
	LeaseToken  string `db:"lease_token"`
	CreatedAt   int64  `db:"created_at"`
	UpdatedAt   int64  `db:"updated_at"`
}

type KbIngestBatchRow struct {
	InternalId int64  `db:"internal_id"`
	Id         string `db:"request_id"`
	KbId       int64  `db:"kb_id"`
	Stopped    bool   `db:"stopped"`
	Total      int64  `db:"total"`
	Queued     int64  `db:"queued"`
	Running    int64  `db:"running"`
	Succeeded  int64  `db:"succeeded"`
	Failed     int64  `db:"failed"`
	Cancelled  int64  `db:"cancelled"`
	Reused     int64  `db:"reused"`
	CreatedAt  int64  `db:"created_at"`
	UpdatedAt  int64  `db:"updated_at"`
}

type KbIngestJobView struct {
	*KbIngestJobRow
	Doc *KbDocRow
}

type KbEnqueueResult struct {
	Job     *KbIngestJobRow
	Doc     *KbDocRow
	BatchId string
}

type KbIngestAttempt struct {
	Job         *KbIngestJobRow
	ChunkIds    []uint64
	OldChunkIds []uint64
	Chunks      []KbChunkRow
}

func NewKbIngestQueue(conn sqlx.SqlConn, docs AiKbDocModel) *KbIngestQueue {
	return &KbIngestQueue{conn: conn, docs: docs}
}

func kbIngestRequestHash(r KbEnqueueRequest) string {
	// 重试只绑定公开请求，不能因服务器读取到新状态而改变同一 request_id 的含义。
	if r.Operation == "retry" {
		r.Title, r.Content, r.ExpectedVersion = "", "", 0
	}
	body, _ := json.Marshal([]any{r.KbId, r.DocId, r.Operation, r.BatchId, r.ExpectedVersion, r.Title, r.Content})
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

const kbJobFields = "id,batch_id,kb_id,doc_id,owner_id,doc_version,request_id,request_hash,operation,status,error,chunk_ids,old_chunk_ids,lease_token,UNIX_TIMESTAMP(created_at) AS created_at,UNIX_TIMESTAMP(updated_at) AS updated_at"

func (q *KbIngestQueue) FindRequest(ctx context.Context, r KbEnqueueRequest) (*KbEnqueueResult, error) {
	if r.BatchId == "" {
		r.BatchId = r.RequestId
	}
	var job KbIngestJobRow
	if err := q.conn.QueryRowCtx(ctx, &job, "SELECT "+kbJobFields+" FROM ai_kb_ingest_job WHERE owner_id=? AND request_id=?", r.OwnerId, r.RequestId); err != nil {
		return nil, err
	}
	if job.KbId != r.KbId || job.RequestHash != kbIngestRequestHash(r) {
		return nil, ErrKbRequestConflict
	}
	doc, err := q.docs.FindMetadataOwned(ctx, r.KbId, job.DocId, r.OwnerId)
	if err != nil {
		return nil, err
	}
	return &KbEnqueueResult{Job: &job, Doc: doc, BatchId: r.BatchId}, nil
}

func (q *KbIngestQueue) Enqueue(ctx context.Context, r KbEnqueueRequest, chunks []KbChunkRow) (*KbEnqueueResult, error) {
	if r.BatchId == "" {
		r.BatchId = r.RequestId
	}
	originalRequest := r
	out := &KbEnqueueResult{BatchId: r.BatchId}
	var changed bool
	err := q.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if err := lockKb(ctx, s, r.KbId, r.OwnerId); err != nil {
			return err
		}
		var prior KbIngestJobRow
		err := s.QueryRowCtx(ctx, &prior, "SELECT "+kbJobFields+" FROM ai_kb_ingest_job WHERE owner_id=? AND request_id=?", r.OwnerId, r.RequestId)
		if err == nil {
			if prior.KbId != r.KbId || prior.RequestHash != kbIngestRequestHash(r) {
				return ErrKbRequestConflict
			}
			out.Job = &prior
			return nil
		}
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		if _, err := s.ExecCtx(ctx, "INSERT INTO ai_kb_ingest_batch(kb_id,owner_id,request_id) VALUES (?,?,?) ON DUPLICATE KEY UPDATE id=id", r.KbId, r.OwnerId, r.BatchId); err != nil {
			return err
		}
		var batch struct {
			Id      int64 `db:"id"`
			Stopped bool  `db:"stopped"`
		}
		if err := s.QueryRowCtx(ctx, &batch, "SELECT id,stopped FROM ai_kb_ingest_batch WHERE kb_id=? AND owner_id=? AND request_id=? FOR UPDATE", r.KbId, r.OwnerId, r.BatchId); err != nil {
			return err
		}
		if batch.Stopped {
			return ErrKbBatchStopped
		}
		var count int64
		if err := s.QueryRowCtx(ctx, &count, "SELECT COUNT(*) FROM ai_kb_ingest_job WHERE batch_id=?", batch.Id); err != nil {
			return err
		}
		if count >= 20 {
			return ErrKbBatchFull
		}
		var oldIds, ids []uint64
		var version int64
		status := "queued"
		switch r.Operation {
		case "create":
			err := s.QueryRowCtx(ctx, &r.DocId, "SELECT id FROM ai_kb_doc WHERE kb_id=? AND deleted_at IS NULL AND BINARY title=BINARY ? AND BINARY content=BINARY ? AND BINARY embedding_model=BINARY ? ORDER BY id LIMIT 1 FOR UPDATE", r.KbId, r.Title, r.Content, r.EmbeddingModel)
			if err == nil {
				status = "reused"
				if err := s.QueryRowCtx(ctx, &version, "SELECT version FROM ai_kb_doc WHERE id=?", r.DocId); err != nil {
					return err
				}
			} else {
				if !errors.Is(err, ErrNotFound) {
					return err
				}
				result, err := s.ExecCtx(ctx, "INSERT INTO ai_kb_doc(kb_id,title,content,status,ingest_stage,chunk_count,embedding_model) VALUES (?,?,?,'pending','queued',?,?)", r.KbId, r.Title, r.Content, len(chunks), r.EmbeddingModel)
				if err != nil {
					return err
				}
				r.DocId, err = result.LastInsertId()
				if err != nil {
					return err
				}
				version = 1
				ids, err = insertKbChunks(ctx, s, r.DocId, r.KbId, chunks)
				if err != nil {
					return err
				}
				changed = true
			}
		case "update", "retry":
			var doc struct {
				Version int64  `db:"version"`
				Status  string `db:"status"`
				Active  bool   `db:"active"`
			}
			if err := s.QueryRowCtx(ctx, &doc, "SELECT version,status,(updated_at>DATE_SUB(NOW(),INTERVAL 120 SECOND)) AS active FROM ai_kb_doc WHERE id=? AND kb_id=? AND deleted_at IS NULL FOR UPDATE", r.DocId, r.KbId); err != nil {
				return err
			}
			if doc.Version != r.ExpectedVersion {
				return ErrKbDocState
			}
			if r.Operation == "retry" {
				var active int64
				if err := s.QueryRowCtx(ctx, &active, "SELECT COUNT(*) FROM ai_kb_ingest_job WHERE doc_id=? AND status IN ('queued','running')", r.DocId); err != nil {
					return err
				}
				if active > 0 || (doc.Status != "failed" && (doc.Status != "pending" || doc.Active)) {
					return ErrKbDocState
				}
			}
			if err := s.QueryRowsCtx(ctx, &oldIds, "SELECT id FROM ai_kb_chunk WHERE doc_id=? ORDER BY id", r.DocId); err != nil {
				return err
			}
			// 后续修订接管尚未执行任务的旧索引清理，避免队列取消丢失补偿范围。
			var previous []struct {
				OldIds string `db:"old_chunk_ids"`
			}
			if err := s.QueryRowsCtx(ctx, &previous, "SELECT old_chunk_ids FROM ai_kb_ingest_job WHERE doc_id=? AND status IN ('queued','running')", r.DocId); err != nil {
				return err
			}
			for _, p := range previous {
				var more []uint64
				if err := json.Unmarshal([]byte(p.OldIds), &more); err != nil {
					return err
				}
				oldIds = append(oldIds, more...)
			}
			if _, err := s.ExecCtx(ctx, "UPDATE ai_kb_ingest_job SET status='failed',error='文档已有新修订',completed_at=NOW() WHERE doc_id=? AND status IN ('queued','running')", r.DocId); err != nil {
				return err
			}
			if _, err := s.ExecCtx(ctx, "DELETE FROM ai_kb_chunk WHERE doc_id=?", r.DocId); err != nil {
				return err
			}
			var err error
			ids, err = insertKbChunks(ctx, s, r.DocId, r.KbId, chunks)
			if err != nil {
				return err
			}
			version = doc.Version
			if r.Operation == "update" {
				version++
				_, err = s.ExecCtx(ctx, "UPDATE ai_kb_doc SET title=?,content=?,version=?,status='pending',ingest_stage='queued',error='',chunk_count=?,embedding_model=?,updated_at=NOW() WHERE id=?", r.Title, r.Content, version, len(chunks), r.EmbeddingModel, r.DocId)
			} else {
				_, err = s.ExecCtx(ctx, "UPDATE ai_kb_doc SET status='pending',ingest_stage='queued',error='',chunk_count=?,embedding_model=?,updated_at=NOW() WHERE id=?", len(chunks), r.EmbeddingModel, r.DocId)
			}
			if err != nil {
				return err
			}
			changed = true
		default:
			return ErrKbDocState
		}
		// create 的文档 ID 是事务结果，不属于客户端请求哈希。
		hashRequest := r
		if r.Operation == "create" {
			hashRequest.DocId = 0
		}
		idsJSON, _ := json.Marshal(nonNilIds(ids))
		oldJSON, _ := json.Marshal(nonNilIds(oldIds))
		result, err := s.ExecCtx(ctx, "INSERT INTO ai_kb_ingest_job(batch_id,kb_id,doc_id,owner_id,doc_version,request_id,request_hash,operation,status,chunk_ids,old_chunk_ids,completed_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,IF(?='reused',NOW(),NULL))", batch.Id, r.KbId, r.DocId, r.OwnerId, version, r.RequestId, kbIngestRequestHash(hashRequest), r.Operation, status, string(idsJSON), string(oldJSON), status)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		out.Job = &KbIngestJobRow{}
		if err := s.QueryRowCtx(ctx, out.Job, "SELECT "+kbJobFields+" FROM ai_kb_ingest_job WHERE id=?", id); err != nil {
			return err
		}
		_, err = s.ExecCtx(ctx, "UPDATE ai_kb_ingest_batch SET updated_at=NOW() WHERE id=?", batch.Id)
		return err
	})
	if err != nil {
		var duplicate *mysql.MySQLError
		if errors.As(err, &duplicate) && duplicate.Number == 1062 {
			// 跨库事务锁不同父行，唯一键竞争后按原始载荷读取已提交请求。
			if accepted, readErr := q.FindRequest(ctx, originalRequest); readErr == nil || errors.Is(readErr, ErrKbRequestConflict) {
				return accepted, readErr
			}
		}
		return nil, err
	}
	var errRead error
	out.Doc, errRead = q.docs.FindMetadataOwned(ctx, r.KbId, out.Job.DocId, r.OwnerId)
	if errRead != nil {
		return out, errRead
	}
	if changed {
		return out, q.docs.InvalidateCommitted(ctx, out.Job.DocId)
	}
	return out, nil
}

func nonNilIds(ids []uint64) []uint64 {
	if ids == nil {
		return []uint64{}
	}
	return ids
}

const kbBatchSelect = "SELECT b.id AS internal_id,b.request_id,b.kb_id,b.stopped,COUNT(j.id) AS total,COALESCE(SUM(j.status='queued'),0) AS queued,COALESCE(SUM(j.status='running'),0) AS running,COALESCE(SUM(j.status='succeeded'),0) AS succeeded,COALESCE(SUM(j.status='failed'),0) AS failed,COALESCE(SUM(j.status='cancelled'),0) AS cancelled,COALESCE(SUM(j.status='reused'),0) AS reused,UNIX_TIMESTAMP(b.created_at) AS created_at,UNIX_TIMESTAMP(GREATEST(b.updated_at,COALESCE(MAX(j.updated_at),b.updated_at))) AS updated_at FROM ai_kb_ingest_batch b JOIN ai_kb k ON k.id=b.kb_id AND k.deleted_at IS NULL LEFT JOIN ai_kb_ingest_job j ON j.batch_id=b.id "

func (q *KbIngestQueue) ListOwned(ctx context.Context, kbId, ownerId, page, size int64) ([]*KbIngestBatchRow, int64, error) {
	var owned int64
	if err := q.conn.QueryRowCtx(ctx, &owned, "SELECT id FROM ai_kb WHERE id=? AND owner_id=? AND deleted_at IS NULL", kbId, ownerId); err != nil {
		return nil, 0, err
	}
	var total int64
	if err := q.conn.QueryRowCtx(ctx, &total, "SELECT COUNT(*) FROM ai_kb_ingest_batch b JOIN ai_kb k ON k.id=b.kb_id AND k.deleted_at IS NULL WHERE b.kb_id=? AND b.owner_id=? AND k.owner_id=?", kbId, ownerId, ownerId); err != nil {
		return nil, 0, err
	}
	rows := make([]*KbIngestBatchRow, 0)
	err := q.conn.QueryRowsCtx(ctx, &rows, kbBatchSelect+"WHERE b.kb_id=? AND b.owner_id=? AND k.owner_id=? GROUP BY b.id ORDER BY b.id DESC LIMIT ? OFFSET ?", kbId, ownerId, ownerId, size, (page-1)*size)
	return rows, total, err
}

func (q *KbIngestQueue) GetOwned(ctx context.Context, kbId, ownerId int64, batchId string) (*KbIngestBatchRow, []*KbIngestJobView, error) {
	var batch KbIngestBatchRow
	if err := q.conn.QueryRowCtx(ctx, &batch, kbBatchSelect+"WHERE b.kb_id=? AND b.owner_id=? AND k.owner_id=? AND b.request_id=? GROUP BY b.id", kbId, ownerId, ownerId, batchId); err != nil {
		return nil, nil, err
	}
	var jobs []*KbIngestJobRow
	if err := q.conn.QueryRowsCtx(ctx, &jobs, "SELECT "+kbJobFields+" FROM ai_kb_ingest_job WHERE batch_id=? ORDER BY id", batch.InternalId); err != nil {
		return nil, nil, err
	}
	views := make([]*KbIngestJobView, 0, len(jobs))
	var documents []*KbDocRow
	if err := q.conn.QueryRowsCtx(ctx, &documents, kbDocMetadataSelect+"WHERE d.kb_id=? AND k.owner_id=? AND d.deleted_at IS NULL AND d.id IN (SELECT doc_id FROM ai_kb_ingest_job WHERE batch_id=?)", kbId, ownerId, batch.InternalId); err != nil {
		return nil, nil, err
	}
	byId := make(map[int64]*KbDocRow, len(documents))
	for _, doc := range documents {
		byId[doc.Id] = doc
	}
	for _, job := range jobs {
		views = append(views, &KbIngestJobView{KbIngestJobRow: job, Doc: byId[job.DocId]})
	}
	return &batch, views, nil
}

func (q *KbIngestQueue) StopOwned(ctx context.Context, kbId, ownerId int64, batchId string) ([]uint64, error) {
	var changed []int64
	var cleanupIds []uint64
	err := q.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if err := lockKb(ctx, s, kbId, ownerId); err != nil {
			return err
		}
		var id int64
		if err := s.QueryRowCtx(ctx, &id, "SELECT id FROM ai_kb_ingest_batch WHERE kb_id=? AND owner_id=? AND request_id=? FOR UPDATE", kbId, ownerId, batchId); err != nil {
			return err
		}
		if _, err := s.ExecCtx(ctx, "UPDATE ai_kb_ingest_batch SET stopped=1,updated_at=NOW() WHERE id=?", id); err != nil {
			return err
		}
		var jobs []*KbIngestJobRow
		if err := s.QueryRowsCtx(ctx, &jobs, "SELECT "+kbJobFields+" FROM ai_kb_ingest_job WHERE batch_id=? AND status='queued' ORDER BY id", id); err != nil {
			return err
		}
		for _, job := range jobs {
			attempt, err := decodeKbAttempt(job)
			if err != nil {
				return err
			}
			cleanupIds = append(cleanupIds, attempt.OldChunkIds...)
			if err := lockKbAttempt(ctx, s, job.DocId, attempt.ChunkIds); err == nil {
				if _, err := s.ExecCtx(ctx, "UPDATE ai_kb_doc SET status='failed',ingest_stage='failed',error='批次已停止，未开始摄取' WHERE id=?", job.DocId); err != nil {
					return err
				}
				if _, err := s.ExecCtx(ctx, "UPDATE ai_kb_chunk SET status='failed' WHERE doc_id=?", job.DocId); err != nil {
					return err
				}
				changed = append(changed, job.DocId)
			} else if !errors.Is(err, ErrNotFound) && !errors.Is(err, ErrKbDocState) {
				return err
			}
			if _, err := s.ExecCtx(ctx, "UPDATE ai_kb_ingest_job SET status='cancelled',error='批次已停止，未开始摄取',completed_at=NOW() WHERE id=? AND status='queued'", job.Id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, id := range changed {
		if err := q.docs.InvalidateCommitted(ctx, id); err != nil {
			return cleanupIds, err
		}
	}
	return cleanupIds, nil
}

func decodeKbAttempt(job *KbIngestJobRow) (*KbIngestAttempt, error) {
	a := &KbIngestAttempt{Job: job}
	if err := json.Unmarshal([]byte(job.ChunkIds), &a.ChunkIds); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(job.OldChunkIds), &a.OldChunkIds); err != nil {
		return nil, err
	}
	return a, nil
}

func (q *KbIngestQueue) Claim(ctx context.Context, token string) (*KbIngestAttempt, error) {
	var job KbIngestJobRow
	if err := q.conn.QueryRowCtx(ctx, &job, "SELECT "+kbJobFields+" FROM ai_kb_ingest_job WHERE status='queued' ORDER BY id LIMIT 1"); err != nil {
		return nil, err
	}
	a, err := decodeKbAttempt(&job)
	if err != nil {
		return nil, err
	}
	var rejected bool
	err = q.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if err := lockKb(ctx, s, job.KbId, job.OwnerId); err != nil {
			if !errors.Is(err, ErrNotFound) {
				return err
			}
			rejected = true
			_, err = s.ExecCtx(ctx, "UPDATE ai_kb_ingest_job SET status='cancelled',error='知识库已删除',completed_at=NOW() WHERE id=? AND status='queued'", job.Id)
			return err
		}
		var stopped bool
		if err := s.QueryRowCtx(ctx, &stopped, "SELECT stopped FROM ai_kb_ingest_batch WHERE id=? FOR UPDATE", job.BatchId); err != nil {
			return err
		}
		var state string
		if err := s.QueryRowCtx(ctx, &state, "SELECT status FROM ai_kb_ingest_job WHERE id=? FOR UPDATE", job.Id); err != nil {
			return err
		}
		if state != "queued" {
			return ErrNotFound
		}
		attemptErr := lockKbAttempt(ctx, s, job.DocId, a.ChunkIds)
		if stopped || errors.Is(attemptErr, ErrNotFound) || errors.Is(attemptErr, ErrKbDocState) {
			rejected = true
			_, err := s.ExecCtx(ctx, "UPDATE ai_kb_ingest_job SET status='cancelled',error='文档已变化或批次已停止',completed_at=NOW() WHERE id=?", job.Id)
			return err
		}
		if attemptErr != nil {
			return attemptErr
		}
		if err := s.QueryRowsCtx(ctx, &a.Chunks, "SELECT id,doc_id,kb_id,seq,content,token_count,'' AS doc_title FROM ai_kb_chunk WHERE doc_id=? ORDER BY seq", job.DocId); err != nil {
			return err
		}
		if _, err := s.ExecCtx(ctx, "UPDATE ai_kb_ingest_job SET status='running',lease_token=?,lease_until=DATE_ADD(NOW(),INTERVAL 120 SECOND),started_at=NOW() WHERE id=?", token, job.Id); err != nil {
			return err
		}
		_, err := s.ExecCtx(ctx, "UPDATE ai_kb_doc SET ingest_stage='embedding' WHERE id=?", job.DocId)
		return err
	})
	if err != nil {
		return nil, err
	}
	if rejected {
		return nil, ErrNotFound
	}
	job.LeaseToken = token
	job.Status = "running"
	if err := q.docs.InvalidateCommitted(ctx, job.DocId); err != nil && !KbDocCommitted(err) {
		return nil, err
	}
	return a, nil
}

// Transition 在租约和代次复核后的同一事务中提交索引阶段或最终状态。
func (q *KbIngestQueue) Transition(ctx context.Context, a *KbIngestAttempt, state, safeErr, embeddingModel string, expiredOnly bool) error {
	var stateErr error
	var changed bool
	err := q.conn.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		kbErr := lockKb(ctx, s, a.Job.KbId, a.Job.OwnerId)
		if kbErr != nil && !errors.Is(kbErr, ErrNotFound) {
			return kbErr
		}
		var batchId int64
		if err := s.QueryRowCtx(ctx, &batchId, "SELECT id FROM ai_kb_ingest_batch WHERE id=? FOR UPDATE", a.Job.BatchId); err != nil {
			return err
		}
		var lease struct {
			Status string `db:"status"`
			Token  string `db:"lease_token"`
			Active bool   `db:"active"`
		}
		if err := s.QueryRowCtx(ctx, &lease, "SELECT status,lease_token,COALESCE(lease_until>NOW(),0) AS active FROM ai_kb_ingest_job WHERE id=? FOR UPDATE", a.Job.Id); err != nil {
			return err
		}
		if lease.Status != "running" || lease.Token != a.Job.LeaseToken || (expiredOnly && lease.Active) {
			return ErrKbDocState
		}
		if state != "failed" && !lease.Active {
			return ErrKbDocState
		}
		attemptErr := kbErr
		if attemptErr == nil {
			attemptErr = lockKbAttempt(ctx, s, a.Job.DocId, a.ChunkIds)
		}
		if attemptErr != nil && !errors.Is(attemptErr, ErrNotFound) && !errors.Is(attemptErr, ErrKbDocState) {
			return attemptErr
		}
		if attemptErr != nil {
			stateErr = ErrKbGenerationChanged
			state = "failed"
			safeErr = "文档已变化或已删除"
		} else {
			switch state {
			case "indexing":
				_, err := s.ExecCtx(ctx, "UPDATE ai_kb_doc SET ingest_stage='indexing' WHERE id=?", a.Job.DocId)
				changed = true
				return err
			case "succeeded":
				if _, err := s.ExecCtx(ctx, "UPDATE ai_kb_chunk SET status='ready' WHERE doc_id=?", a.Job.DocId); err != nil {
					return err
				}
				if _, err := s.ExecCtx(ctx, "UPDATE ai_kb_doc SET status='ready',ingest_stage='ready',error='',embedding_model=? WHERE id=?", embeddingModel, a.Job.DocId); err != nil {
					return err
				}
			case "failed":
				if _, err := s.ExecCtx(ctx, "UPDATE ai_kb_chunk SET status='failed' WHERE doc_id=?", a.Job.DocId); err != nil {
					return err
				}
				if _, err := s.ExecCtx(ctx, "UPDATE ai_kb_doc SET status='failed',ingest_stage='failed',error=? WHERE id=?", safeErr, a.Job.DocId); err != nil {
					return err
				}
			default:
				return ErrKbDocState
			}
			changed = true
		}
		_, err := s.ExecCtx(ctx, "UPDATE ai_kb_ingest_job SET status=?,error=?,completed_at=NOW(),lease_until=NULL WHERE id=?", state, safeErr, a.Job.Id)
		return err
	})
	if err != nil {
		return err
	}
	if changed {
		if err := q.docs.InvalidateCommitted(ctx, a.Job.DocId); err != nil {
			return err
		}
	}
	return stateErr
}

func (q *KbIngestQueue) Expired(ctx context.Context) ([]*KbIngestAttempt, error) {
	var jobs []*KbIngestJobRow
	if err := q.conn.QueryRowsCtx(ctx, &jobs, "SELECT "+kbJobFields+" FROM ai_kb_ingest_job WHERE status='running' AND lease_until<=NOW() ORDER BY id LIMIT 20"); err != nil {
		return nil, err
	}
	result := make([]*KbIngestAttempt, 0, len(jobs))
	for _, job := range jobs {
		a, err := decodeKbAttempt(job)
		if err != nil {
			return nil, err
		}
		err = q.Transition(ctx, a, "failed", "摄取进程中断或超时，请核对后手动重试", "", true)
		if err == nil || KbDocCommitted(err) || errors.Is(err, ErrNotFound) || errors.Is(err, ErrKbGenerationChanged) {
			result = append(result, a)
		} else if !errors.Is(err, ErrKbDocState) {
			return result, err
		}
	}
	return result, nil
}

func pendingKbJobVectorIDs(ctx context.Context, s sqlx.Session, kbId, docId int64) ([]uint64, error) {
	where := "kb_id=? AND status IN ('queued','running')"
	args := []any{kbId}
	if docId > 0 {
		where += " AND doc_id=?"
		args = append(args, docId)
	}
	var jobs []struct {
		Ids string `db:"old_chunk_ids"`
	}
	if err := s.QueryRowsCtx(ctx, &jobs, "SELECT old_chunk_ids FROM ai_kb_ingest_job WHERE "+where, args...); err != nil {
		return nil, err
	}
	var ids []uint64
	for _, job := range jobs {
		var previous []uint64
		if err := json.Unmarshal([]byte(job.Ids), &previous); err != nil {
			return nil, err
		}
		ids = append(ids, previous...)
	}
	return ids, nil
}
