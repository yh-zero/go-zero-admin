package model

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiKbDocModel = (*customAiKbDocModel)(nil)

// ErrKbDocState 文档状态已变化（如并发重试/已就绪），映射为 400 而非系统错误
var ErrKbDocState = errors.New("kb doc state changed")

// KbDocCommittedError 表示事务已提交，仅缓存失效失败；调用方不能按回滚补偿已提交的数据。
type KbDocCommittedError struct{ Err error }

func (e *KbDocCommittedError) Error() string {
	return "kb doc committed; cache invalidation failed: " + e.Err.Error()
}
func (e *KbDocCommittedError) Unwrap() error { return e.Err }

func KbDocCommitted(err error) bool {
	var committed *KbDocCommittedError
	return errors.As(err, &committed)
}

type (
	// AiKbDocModel 知识库文档数据层；归属经 kb 行校验，越权与不存在同样返回 ErrNotFound
	AiKbDocModel interface {
		aiKbDocModel
		FindOwned(ctx context.Context, kbId, docId, ownerId int64) (*KbDocRow, error)
		FindMetadataOwned(ctx context.Context, kbId, docId, ownerId int64) (*KbDocRow, error)
		ListInKb(ctx context.Context, kbId, ownerId, page, pageSize int64) ([]*KbDocRow, int64, error)
		SearchInKb(ctx context.Context, kbId, ownerId, page, pageSize int64, query, status string) ([]*KbDocRow, int64, error)
		CountsInKb(ctx context.Context, kbId, ownerId int64) (*KbDocCounts, error)
		// InsertWithChunks 在父知识库锁内精确去重并创建分块；created=false 时复用已有文档。
		InsertWithChunks(ctx context.Context, ownerId int64, doc *KbDocRow, chunks []KbChunkRow) (docId int64, chunkIds []uint64, created bool, err error)
		// MarkIngested 摄取成功后置文档 ready、chunk ready；MarkFailed 置文档 failed 并记录安全错误
		MarkIngested(ctx context.Context, docId int64, chunkIds []uint64, embeddingModel string) error
		MarkFailed(ctx context.Context, docId int64, chunkIds []uint64, safeErr string) error
		// DeleteOwned 软删文档并删除其 chunk 行，返回需从向量索引清除的 chunk ID
		DeleteOwned(ctx context.Context, kbId, docId, ownerId int64) (chunkIds []uint64, err error)
		// ReplaceChunksForRetry 原子领取 failed/过期 pending 文档并替换分块，刷新尝试时间。
		ReplaceChunksForRetry(ctx context.Context, kbId, docId, ownerId int64, chunks []KbChunkRow, expectedVersion ...int64) (oldChunkIds, chunkIds []uint64, err error)
		ReplaceDocument(ctx context.Context, ownerId, expectedVersion int64, doc *KbDocRow, chunks []KbChunkRow) (oldChunkIds, chunkIds []uint64, err error)
		MarkIndexing(ctx context.Context, docId int64, chunkIds []uint64) error
		InvalidateCommitted(ctx context.Context, docId int64) error
	}

	customAiKbDocModel struct {
		*defaultAiKbDocModel
		conn sqlx.SqlConn
	}

	// KbDocRow 文档行（时间已转 int64）
	KbDocRow struct {
		Id             int64  `db:"id"`
		KbId           int64  `db:"kb_id"`
		Title          string `db:"title"`
		Content        string `db:"content"`
		Status         string `db:"status"`
		Error          string `db:"error"`
		ChunkCount     int64  `db:"chunk_count"`
		EmbeddingModel string `db:"embedding_model"`
		CreatedAt      int64  `db:"created_at"`
		UpdatedAt      int64  `db:"updated_at"`
		Version        int64  `db:"version"`
		Stage          string `db:"ingest_stage"`
	}
)

type KbDocCounts struct {
	ReadyCount   int64 `db:"ready_count"`
	PendingCount int64 `db:"pending_count"`
	FailedCount  int64 `db:"failed_count"`
}

// NewAiKbDocModel returns a model for the database table.
func NewAiKbDocModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiKbDocModel {
	return &customAiKbDocModel{defaultAiKbDocModel: newAiKbDocModel(conn, c, opts...), conn: conn}
}

const kbDocFields = "d.id,d.kb_id,d.title,%s AS content,d.status,d.error,d.chunk_count,d.embedding_model,d.version,d.ingest_stage," +
	"UNIX_TIMESTAMP(d.created_at) AS created_at,UNIX_TIMESTAMP(d.updated_at) AS updated_at " +
	"FROM ai_kb_doc d JOIN ai_kb k ON d.kb_id=k.id AND k.deleted_at IS NULL "

var kbDocSelect = "SELECT " + fmt.Sprintf(kbDocFields, "d.content")
var kbDocMetadataSelect = "SELECT " + fmt.Sprintf(kbDocFields, "''")

func (m *customAiKbDocModel) FindOwned(ctx context.Context, kbId, docId, ownerId int64) (*KbDocRow, error) {
	var row KbDocRow
	err := m.QueryRowNoCacheCtx(ctx, &row, kbDocSelect+"WHERE d.id=? AND d.kb_id=? AND k.owner_id=? AND d.deleted_at IS NULL", docId, kbId, ownerId)
	return &row, err
}

func (m *customAiKbDocModel) FindMetadataOwned(ctx context.Context, kbId, docId, ownerId int64) (*KbDocRow, error) {
	var row KbDocRow
	err := m.QueryRowNoCacheCtx(ctx, &row, kbDocMetadataSelect+"WHERE d.id=? AND d.kb_id=? AND k.owner_id=? AND d.deleted_at IS NULL", docId, kbId, ownerId)
	return &row, err
}

func (m *customAiKbDocModel) ListInKb(ctx context.Context, kbId, ownerId, page, pageSize int64) ([]*KbDocRow, int64, error) {
	return m.SearchInKb(ctx, kbId, ownerId, page, pageSize, "", "")
}

func (m *customAiKbDocModel) SearchInKb(ctx context.Context, kbId, ownerId, page, pageSize int64, query, status string) ([]*KbDocRow, int64, error) {
	where := "WHERE d.kb_id=? AND k.owner_id=? AND d.deleted_at IS NULL"
	args := []any{kbId, ownerId}
	if query != "" {
		where += " AND d.title LIKE ? ESCAPE '!'"
		args = append(args, "%"+strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(query)+"%")
	}
	if status != "" {
		where += " AND d.status=?"
		args = append(args, status)
	}
	var count struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &count,
		"SELECT COUNT(*) AS total FROM ai_kb_doc d JOIN ai_kb k ON d.kb_id=k.id AND k.deleted_at IS NULL "+
			where, args...); err != nil {
		return nil, 0, err
	}
	rows := make([]*KbDocRow, 0)
	args = append(args, pageSize, (page-1)*pageSize)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, kbDocMetadataSelect+where+" ORDER BY d.id DESC LIMIT ? OFFSET ?", args...)
	return rows, count.Total, err
}

func (m *customAiKbDocModel) CountsInKb(ctx context.Context, kbId, ownerId int64) (*KbDocCounts, error) {
	var row KbDocCounts
	err := m.QueryRowNoCacheCtx(ctx, &row, "SELECT COALESCE(SUM(d.status='ready'),0) AS ready_count,COALESCE(SUM(d.status='pending'),0) AS pending_count,COALESCE(SUM(d.status='failed'),0) AS failed_count FROM ai_kb k LEFT JOIN ai_kb_doc d ON d.kb_id=k.id AND d.deleted_at IS NULL WHERE k.id=? AND k.owner_id=? AND k.deleted_at IS NULL GROUP BY k.id", kbId, ownerId)
	return &row, err
}

func (m *customAiKbDocModel) InsertWithChunks(ctx context.Context, ownerId int64, doc *KbDocRow, chunks []KbChunkRow) (int64, []uint64, bool, error) {
	var docId int64
	var created bool
	chunkIds := make([]uint64, 0, len(chunks))
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := lockKb(ctx, session, doc.KbId, ownerId); err != nil {
			return err
		}
		// 父行串行化同库创建；BINARY 避免数据库默认忽略大小写/重音造成错误合并。
		err := session.QueryRowCtx(ctx, &docId, "SELECT id FROM ai_kb_doc WHERE kb_id=? AND deleted_at IS NULL AND BINARY title=BINARY ? AND BINARY content=BINARY ? AND BINARY embedding_model=BINARY ? ORDER BY id LIMIT 1 FOR UPDATE", doc.KbId, doc.Title, doc.Content, doc.EmbeddingModel)
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		result, err := session.ExecCtx(ctx,
			"INSERT INTO ai_kb_doc (kb_id,title,content,status,chunk_count,embedding_model) VALUES (?,?,?,'pending',?,?)",
			doc.KbId, doc.Title, doc.Content, len(chunks), doc.EmbeddingModel)
		if err != nil {
			return err
		}
		docId, err = result.LastInsertId()
		if err != nil {
			return err
		}
		chunkIds, err = insertKbChunks(ctx, session, docId, doc.KbId, chunks)
		created = err == nil
		return err
	})
	if err != nil {
		return 0, nil, false, err
	}
	return docId, chunkIds, created, nil
}

func (m *customAiKbDocModel) MarkIngested(ctx context.Context, docId int64, chunkIds []uint64, embeddingModel string) error {
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := lockKbAttempt(ctx, session, docId, chunkIds); err != nil {
			return err
		}
		for _, id := range chunkIds {
			if _, err := session.ExecCtx(ctx, "UPDATE ai_kb_chunk SET status='ready' WHERE id=?", id); err != nil {
				return err
			}
		}
		_, err := session.ExecCtx(ctx,
			"UPDATE ai_kb_doc SET status='ready',ingest_stage='ready',error='',embedding_model=? WHERE id=?", embeddingModel, docId)
		return err
	})
	if err != nil {
		return err
	}
	return m.invalidateCommitted(ctx, docId)
}

func (m *customAiKbDocModel) MarkFailed(ctx context.Context, docId int64, chunkIds []uint64, safeErr string) error {
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := lockKbAttempt(ctx, session, docId, chunkIds); err != nil {
			return err
		}
		if _, err := session.ExecCtx(ctx, "UPDATE ai_kb_chunk SET status='failed' WHERE doc_id=?", docId); err != nil {
			return err
		}
		_, err := session.ExecCtx(ctx, "UPDATE ai_kb_doc SET status='failed',ingest_stage='failed',error=? WHERE id=?", safeErr, docId)
		return err
	})
	if err != nil {
		return err
	}
	return m.invalidateCommitted(ctx, docId)
}

func (m *customAiKbDocModel) DeleteOwned(ctx context.Context, kbId, docId, ownerId int64) ([]uint64, error) {
	chunkIds := make([]uint64, 0)
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := lockKb(ctx, session, kbId, ownerId); err != nil {
			return err
		}
		var owned struct {
			Id int64 `db:"id"`
		}
		if err := session.QueryRowCtx(ctx, &owned,
			"SELECT id FROM ai_kb_doc WHERE id=? AND kb_id=? AND deleted_at IS NULL FOR UPDATE", docId, kbId); err != nil {
			return err
		}
		rows := make([]struct {
			Id uint64 `db:"id"`
		}, 0)
		if err := session.QueryRowsCtx(ctx, &rows, "SELECT id FROM ai_kb_chunk WHERE doc_id=?", docId); err != nil {
			return err
		}
		for _, r := range rows {
			chunkIds = append(chunkIds, r.Id)
		}
		oldIds, err := pendingKbJobVectorIDs(ctx, session, kbId, docId)
		if err != nil {
			return err
		}
		chunkIds = append(chunkIds, oldIds...)
		if _, err := session.ExecCtx(ctx, "DELETE FROM ai_kb_chunk WHERE doc_id=?", docId); err != nil {
			return err
		}
		if _, err := session.ExecCtx(ctx, "UPDATE ai_kb_doc SET deleted_at=NOW() WHERE id=?", docId); err != nil {
			return err
		}
		_, err = session.ExecCtx(ctx, "UPDATE ai_kb_ingest_job SET status='cancelled',error='文档已删除',completed_at=NOW(),lease_until=NULL WHERE doc_id=? AND status IN ('queued','running')", docId)
		return err
	})
	if err != nil {
		return nil, err
	}
	return chunkIds, m.invalidateCommitted(ctx, docId)
}

func insertKbChunks(ctx context.Context, session sqlx.Session, docId, kbId int64, chunks []KbChunkRow) ([]uint64, error) {
	chunkIds := make([]uint64, 0, len(chunks))
	for _, c := range chunks {
		r, err := session.ExecCtx(ctx,
			"INSERT INTO ai_kb_chunk (doc_id,kb_id,seq,content,token_count,status) VALUES (?,?,?,?,?,'pending')",
			docId, kbId, c.Seq, c.Content, c.TokenCount)
		if err != nil {
			return nil, err
		}
		id, err := r.LastInsertId()
		if err != nil {
			return nil, err
		}
		chunkIds = append(chunkIds, uint64(id))
	}
	return chunkIds, nil
}

func (m *customAiKbDocModel) ReplaceChunksForRetry(ctx context.Context, kbId, docId, ownerId int64, chunks []KbChunkRow, expectedVersion ...int64) ([]uint64, []uint64, error) {
	oldIds := make([]uint64, 0)
	var chunkIds []uint64
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := lockKb(ctx, session, kbId, ownerId); err != nil {
			return err
		}
		var doc struct {
			Id      int64  `db:"id"`
			Status  string `db:"status"`
			Active  bool   `db:"active"`
			Version int64  `db:"version"`
		}
		if err := session.QueryRowCtx(ctx, &doc,
			"SELECT id,status,version,(updated_at > DATE_SUB(NOW(), INTERVAL 120 SECOND)) AS active FROM ai_kb_doc WHERE id=? AND kb_id=? AND deleted_at IS NULL FOR UPDATE", docId, kbId); err != nil {
			return err
		}
		if doc.Status != "failed" && (doc.Status != "pending" || doc.Active) {
			return ErrKbDocState
		}
		// 读取正文与领取重试之间可能发生编辑，旧正文不能替换新修订的分块。
		if len(expectedVersion) > 0 && doc.Version != expectedVersion[0] {
			return ErrKbDocState
		}
		rows := make([]struct {
			Id uint64 `db:"id"`
		}, 0)
		if err := session.QueryRowsCtx(ctx, &rows, "SELECT id FROM ai_kb_chunk WHERE doc_id=?", docId); err != nil {
			return err
		}
		for _, r := range rows {
			oldIds = append(oldIds, r.Id)
		}
		if _, err := session.ExecCtx(ctx, "DELETE FROM ai_kb_chunk WHERE doc_id=?", docId); err != nil {
			return err
		}
		var err error
		chunkIds, err = insertKbChunks(ctx, session, docId, kbId, chunks)
		if err != nil {
			return err
		}
		_, err = session.ExecCtx(ctx, "UPDATE ai_kb_doc SET status='pending',ingest_stage='embedding',error='',chunk_count=?,updated_at=NOW() WHERE id=?", len(chunks), docId)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return oldIds, chunkIds, m.invalidateCommitted(ctx, docId)
}

// ReplaceDocument 在同一事务中替换正文及整个分块代次，旧索引即使未删除也无法通过事实源复核。
func (m *customAiKbDocModel) ReplaceDocument(ctx context.Context, ownerId, expectedVersion int64, doc *KbDocRow, chunks []KbChunkRow) ([]uint64, []uint64, error) {
	oldIds := make([]uint64, 0)
	var nextIds []uint64
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := lockKb(ctx, session, doc.KbId, ownerId); err != nil {
			return err
		}
		var version int64
		if err := session.QueryRowCtx(ctx, &version, "SELECT version FROM ai_kb_doc WHERE id=? AND kb_id=? AND deleted_at IS NULL FOR UPDATE", doc.Id, doc.KbId); err != nil {
			return err
		}
		if version != expectedVersion {
			return ErrKbDocState
		}
		if err := session.QueryRowsCtx(ctx, &oldIds, "SELECT id FROM ai_kb_chunk WHERE doc_id=? ORDER BY id", doc.Id); err != nil {
			return err
		}
		if _, err := session.ExecCtx(ctx, "DELETE FROM ai_kb_chunk WHERE doc_id=?", doc.Id); err != nil {
			return err
		}
		var err error
		nextIds, err = insertKbChunks(ctx, session, doc.Id, doc.KbId, chunks)
		if err != nil {
			return err
		}
		_, err = session.ExecCtx(ctx, "UPDATE ai_kb_doc SET title=?,content=?,version=version+1,status='pending',ingest_stage='embedding',error='',chunk_count=?,embedding_model=?,updated_at=NOW() WHERE id=?", doc.Title, doc.Content, len(chunks), doc.EmbeddingModel, doc.Id)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return oldIds, nextIds, m.invalidateCommitted(ctx, doc.Id)
}

func (m *customAiKbDocModel) MarkIndexing(ctx context.Context, docId int64, chunkIds []uint64) error {
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := lockKbAttempt(ctx, session, docId, chunkIds); err != nil {
			return err
		}
		_, err := session.ExecCtx(ctx, "UPDATE ai_kb_doc SET ingest_stage='indexing' WHERE id=?", docId)
		return err
	})
	if err != nil {
		return err
	}
	return m.invalidateCommitted(ctx, docId)
}

func (m *customAiKbDocModel) invalidateCommitted(ctx context.Context, docId int64) error {
	if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiKbDocIdPrefix, docId)); err != nil {
		return &KbDocCommittedError{Err: err}
	}
	return nil
}

func (m *customAiKbDocModel) InvalidateCommitted(ctx context.Context, docId int64) error {
	return m.invalidateCommitted(ctx, docId)
}

// 全部文档写入按父 KB→doc→chunks 顺序加锁，与 KB 级联删除一致。
func lockKb(ctx context.Context, session sqlx.Session, kbId, ownerId int64) error {
	var id int64
	return session.QueryRowCtx(ctx, &id, "SELECT id FROM ai_kb WHERE id=? AND owner_id=? AND deleted_at IS NULL FOR UPDATE", kbId, ownerId)
}

func lockKbAttempt(ctx context.Context, session sqlx.Session, docId int64, expected []uint64) error {
	var kbId int64
	if err := session.QueryRowCtx(ctx, &kbId, "SELECT kb_id FROM ai_kb_doc WHERE id=?", docId); err != nil {
		return err
	}
	var liveKbId int64
	if err := session.QueryRowCtx(ctx, &liveKbId, "SELECT id FROM ai_kb WHERE id=? AND deleted_at IS NULL FOR UPDATE", kbId); err != nil {
		return err
	}
	var status string
	if err := session.QueryRowCtx(ctx, &status, "SELECT status FROM ai_kb_doc WHERE id=? AND deleted_at IS NULL FOR UPDATE", docId); err != nil {
		return err
	}
	if status != "pending" || len(expected) == 0 {
		return ErrKbDocState
	}
	var current []uint64
	if err := session.QueryRowsCtx(ctx, &current, "SELECT id FROM ai_kb_chunk WHERE doc_id=? ORDER BY id FOR UPDATE", docId); err != nil {
		return err
	}
	want := slices.Clone(expected)
	slices.Sort(want)
	if !slices.Equal(current, want) {
		return ErrKbDocState
	}
	return nil
}
