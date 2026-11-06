package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiKbDocModel = (*customAiKbDocModel)(nil)

// ErrKbDocState 文档状态已变化（如并发重试/已就绪），映射为 400 而非系统错误
var ErrKbDocState = errors.New("kb doc state changed")

type (
	// AiKbDocModel 知识库文档数据层；归属经 kb 行校验，越权与不存在同样返回 ErrNotFound
	AiKbDocModel interface {
		aiKbDocModel
		FindOwned(ctx context.Context, kbId, docId, ownerId int64) (*KbDocRow, error)
		ListInKb(ctx context.Context, kbId, ownerId, page, pageSize int64) ([]*KbDocRow, int64, error)
		// InsertWithChunks 事务内创建文档与分块（初始 status=pending），返回文档 ID 与 chunk ID 列表
		InsertWithChunks(ctx context.Context, doc *KbDocRow, chunks []KbChunkRow) (int64, []uint64, error)
		// MarkIngested 摄取成功后置文档 ready、chunk ready；MarkFailed 置文档 failed 并记录安全错误
		MarkIngested(ctx context.Context, docId int64, chunkIds []uint64, embeddingModel string) error
		MarkFailed(ctx context.Context, docId int64, safeErr string) error
		// DeleteOwned 软删文档并删除其 chunk 行，返回需从向量索引清除的 chunk ID
		DeleteOwned(ctx context.Context, kbId, docId, ownerId int64) (chunkIds []uint64, err error)
		// ResetForRetry 仅 failed/pending 可重试：软删状态不变，清旧 chunk 行并将文档置回 pending，返回旧 chunk ID
		ResetForRetry(ctx context.Context, kbId, docId, ownerId int64) (oldChunkIds []uint64, err error)
		// ReinsertChunks 重试路径：事务内插入新分块并刷新 chunk_count
		ReinsertChunks(ctx context.Context, docId, kbId int64, chunks []KbChunkRow) ([]uint64, error)
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
	}
)

// NewAiKbDocModel returns a model for the database table.
func NewAiKbDocModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiKbDocModel {
	return &customAiKbDocModel{defaultAiKbDocModel: newAiKbDocModel(conn, c, opts...), conn: conn}
}

const kbDocSelect = "SELECT d.id,d.kb_id,d.title,d.content,d.status,d.error,d.chunk_count,d.embedding_model," +
	"UNIX_TIMESTAMP(d.created_at) AS created_at,UNIX_TIMESTAMP(d.updated_at) AS updated_at " +
	"FROM ai_kb_doc d JOIN ai_kb k ON d.kb_id=k.id AND k.deleted_at IS NULL "

func (m *customAiKbDocModel) FindOwned(ctx context.Context, kbId, docId, ownerId int64) (*KbDocRow, error) {
	var row KbDocRow
	err := m.QueryRowNoCacheCtx(ctx, &row, kbDocSelect+"WHERE d.id=? AND d.kb_id=? AND k.owner_id=? AND d.deleted_at IS NULL", docId, kbId, ownerId)
	return &row, err
}

func (m *customAiKbDocModel) ListInKb(ctx context.Context, kbId, ownerId, page, pageSize int64) ([]*KbDocRow, int64, error) {
	var count struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &count,
		"SELECT COUNT(*) AS total FROM ai_kb_doc d JOIN ai_kb k ON d.kb_id=k.id AND k.deleted_at IS NULL "+
			"WHERE d.kb_id=? AND k.owner_id=? AND d.deleted_at IS NULL", kbId, ownerId); err != nil {
		return nil, 0, err
	}
	rows := make([]*KbDocRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, kbDocSelect+
		"WHERE d.kb_id=? AND k.owner_id=? AND d.deleted_at IS NULL ORDER BY d.id DESC LIMIT ? OFFSET ?",
		kbId, ownerId, pageSize, (page-1)*pageSize)
	return rows, count.Total, err
}

func (m *customAiKbDocModel) InsertWithChunks(ctx context.Context, doc *KbDocRow, chunks []KbChunkRow) (int64, []uint64, error) {
	var docId int64
	chunkIds := make([]uint64, 0, len(chunks))
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
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
		for _, c := range chunks {
			r, err := session.ExecCtx(ctx,
				"INSERT INTO ai_kb_chunk (doc_id,kb_id,seq,content,token_count,status) VALUES (?,?,?,?,?,'pending')",
				docId, doc.KbId, c.Seq, c.Content, c.TokenCount)
			if err != nil {
				return err
			}
			id, err := r.LastInsertId()
			if err != nil {
				return err
			}
			chunkIds = append(chunkIds, uint64(id))
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	return docId, chunkIds, nil
}

func (m *customAiKbDocModel) MarkIngested(ctx context.Context, docId int64, chunkIds []uint64, embeddingModel string) error {
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		for _, id := range chunkIds {
			if _, err := session.ExecCtx(ctx, "UPDATE ai_kb_chunk SET status='ready' WHERE id=?", id); err != nil {
				return err
			}
		}
		_, err := session.ExecCtx(ctx,
			"UPDATE ai_kb_doc SET status='ready',error='',embedding_model=? WHERE id=?", embeddingModel, docId)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiKbDocIdPrefix, docId))
}

func (m *customAiKbDocModel) MarkFailed(ctx context.Context, docId int64, safeErr string) error {
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if _, err := session.ExecCtx(ctx, "UPDATE ai_kb_chunk SET status='failed' WHERE doc_id=?", docId); err != nil {
			return err
		}
		_, err := session.ExecCtx(ctx, "UPDATE ai_kb_doc SET status='failed',error=? WHERE id=?", safeErr, docId)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiKbDocIdPrefix, docId))
}

func (m *customAiKbDocModel) DeleteOwned(ctx context.Context, kbId, docId, ownerId int64) ([]uint64, error) {
	chunkIds := make([]uint64, 0)
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var owned struct {
			Id int64 `db:"id"`
		}
		if err := session.QueryRowCtx(ctx, &owned,
			"SELECT d.id FROM ai_kb_doc d JOIN ai_kb k ON d.kb_id=k.id AND k.deleted_at IS NULL "+
				"WHERE d.id=? AND d.kb_id=? AND k.owner_id=? AND d.deleted_at IS NULL FOR UPDATE", docId, kbId, ownerId); err != nil {
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
		if _, err := session.ExecCtx(ctx, "DELETE FROM ai_kb_chunk WHERE doc_id=?", docId); err != nil {
			return err
		}
		_, err := session.ExecCtx(ctx, "UPDATE ai_kb_doc SET deleted_at=NOW() WHERE id=?", docId)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiKbDocIdPrefix, docId)); err != nil {
		return nil, err
	}
	return chunkIds, nil
}

func (m *customAiKbDocModel) ReinsertChunks(ctx context.Context, docId, kbId int64, chunks []KbChunkRow) ([]uint64, error) {
	chunkIds := make([]uint64, 0, len(chunks))
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		for _, c := range chunks {
			r, err := session.ExecCtx(ctx,
				"INSERT INTO ai_kb_chunk (doc_id,kb_id,seq,content,token_count,status) VALUES (?,?,?,?,?,'pending')",
				docId, kbId, c.Seq, c.Content, c.TokenCount)
			if err != nil {
				return err
			}
			id, err := r.LastInsertId()
			if err != nil {
				return err
			}
			chunkIds = append(chunkIds, uint64(id))
		}
		_, err := session.ExecCtx(ctx, "UPDATE ai_kb_doc SET chunk_count=? WHERE id=?", len(chunks), docId)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiKbDocIdPrefix, docId)); err != nil {
		return nil, err
	}
	return chunkIds, nil
}

func (m *customAiKbDocModel) ResetForRetry(ctx context.Context, kbId, docId, ownerId int64) ([]uint64, error) {
	oldIds := make([]uint64, 0)
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var doc struct {
			Id     int64  `db:"id"`
			Status string `db:"status"`
		}
		if err := session.QueryRowCtx(ctx, &doc,
			"SELECT d.id,d.status FROM ai_kb_doc d JOIN ai_kb k ON d.kb_id=k.id AND k.deleted_at IS NULL "+
				"WHERE d.id=? AND d.kb_id=? AND k.owner_id=? AND d.deleted_at IS NULL FOR UPDATE", docId, kbId, ownerId); err != nil {
			return err
		}
		if doc.Status != "failed" && doc.Status != "pending" {
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
		_, err := session.ExecCtx(ctx, "UPDATE ai_kb_doc SET status='pending',error='' WHERE id=?", docId)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiKbDocIdPrefix, docId)); err != nil {
		return nil, err
	}
	return oldIds, nil
}
