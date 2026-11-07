package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiKbModel = (*customAiKbModel)(nil)

type (
	// AiKbModel 知识库数据层；归属校验以参数传入，越权与不存在同样返回 ErrNotFound
	AiKbModel interface {
		aiKbModel
		FindOwned(ctx context.Context, id, ownerId int64) (*KbRow, error)
		ListOwned(ctx context.Context, ownerId, page, pageSize int64) ([]*KbRow, int64, error)
		SearchOwned(ctx context.Context, ownerId, page, pageSize int64, query string) ([]*KbRow, int64, error)
		UpdateOwned(ctx context.Context, id, ownerId int64, patch KbPatch) error
		// DeleteOwned 事务内软删知识库及其文档、删除 chunk 行，返回需从向量索引清除的 chunk ID
		DeleteOwned(ctx context.Context, id, ownerId int64) (chunkIds []uint64, err error)
	}

	customAiKbModel struct {
		*defaultAiKbModel
		conn sqlx.SqlConn
	}

	// KbRow 知识库行（时间已转 int64，统计列随查询聚合）
	KbRow struct {
		Id           int64  `db:"id"`
		OwnerId      int64  `db:"owner_id"`
		Name         string `db:"name"`
		Description  string `db:"description"`
		DocCount     int64  `db:"doc_count"`
		ChunkCount   int64  `db:"chunk_count"`
		ReadyCount   int64  `db:"ready_count"`
		PendingCount int64  `db:"pending_count"`
		FailedCount  int64  `db:"failed_count"`
		CreatedAt    int64  `db:"created_at"`
		UpdatedAt    int64  `db:"updated_at"`
	}

	// KbPatch 部分更新；nil 表示不修改
	KbPatch struct {
		Name, Description *string
	}
)

// NewAiKbModel returns a model for the database table.
func NewAiKbModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiKbModel {
	return &customAiKbModel{defaultAiKbModel: newAiKbModel(conn, c, opts...), conn: conn}
}

// 统计随主查询一次聚合（避免列表 N+1）：文档按软删过滤，chunk 经未删除文档连接
const kbSelect = "SELECT k.id,k.owner_id,k.name,k.description," +
	"(SELECT COUNT(*) FROM ai_kb_doc d WHERE d.kb_id=k.id AND d.deleted_at IS NULL) AS doc_count," +
	"(SELECT COUNT(*) FROM ai_kb_doc d WHERE d.kb_id=k.id AND d.deleted_at IS NULL AND d.status='ready') AS ready_count," +
	"(SELECT COUNT(*) FROM ai_kb_doc d WHERE d.kb_id=k.id AND d.deleted_at IS NULL AND d.status='pending') AS pending_count," +
	"(SELECT COUNT(*) FROM ai_kb_doc d WHERE d.kb_id=k.id AND d.deleted_at IS NULL AND d.status='failed') AS failed_count," +
	"(SELECT COUNT(*) FROM ai_kb_chunk c JOIN ai_kb_doc d2 ON c.doc_id=d2.id AND d2.deleted_at IS NULL WHERE c.kb_id=k.id) AS chunk_count," +
	"UNIX_TIMESTAMP(k.created_at) AS created_at,UNIX_TIMESTAMP(k.updated_at) AS updated_at FROM ai_kb k "

func (m *customAiKbModel) FindOwned(ctx context.Context, id, ownerId int64) (*KbRow, error) {
	var row KbRow
	err := m.QueryRowNoCacheCtx(ctx, &row, kbSelect+"WHERE k.id=? AND k.owner_id=? AND k.deleted_at IS NULL", id, ownerId)
	return &row, err
}

func (m *customAiKbModel) ListOwned(ctx context.Context, ownerId, page, pageSize int64) ([]*KbRow, int64, error) {
	return m.SearchOwned(ctx, ownerId, page, pageSize, "")
}

func (m *customAiKbModel) SearchOwned(ctx context.Context, ownerId, page, pageSize int64, query string) ([]*KbRow, int64, error) {
	where := "WHERE k.owner_id=? AND k.deleted_at IS NULL"
	args := []any{ownerId}
	if query != "" {
		pattern := "%" + strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(query) + "%"
		where += " AND (k.name LIKE ? ESCAPE '!' OR k.description LIKE ? ESCAPE '!')"
		args = append(args, pattern, pattern)
	}
	var count struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM ai_kb k "+where, args...); err != nil {
		return nil, 0, err
	}
	rows := make([]*KbRow, 0)
	args = append(args, pageSize, (page-1)*pageSize)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, kbSelect+where+" ORDER BY k.id DESC LIMIT ? OFFSET ?", args...)
	return rows, count.Total, err
}

func (m *customAiKbModel) UpdateOwned(ctx context.Context, id, ownerId int64, patch KbPatch) error {
	sets := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if patch.Name != nil {
		sets = append(sets, "name=?")
		args = append(args, *patch.Name)
	}
	if patch.Description != nil {
		sets = append(sets, "description=?")
		args = append(args, *patch.Description)
	}
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var owner struct {
			Id int64 `db:"id"`
		}
		if err := session.QueryRowCtx(ctx, &owner, "SELECT id FROM ai_kb WHERE id=? AND owner_id=? AND deleted_at IS NULL FOR UPDATE", id, ownerId); err != nil {
			return err
		}
		if len(sets) == 0 {
			return nil
		}
		_, err := session.ExecCtx(ctx, "UPDATE ai_kb SET "+strings.Join(sets, ",")+" WHERE id=? AND owner_id=?", append(args, id, ownerId)...)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiKbIdPrefix, id))
}

func (m *customAiKbModel) DeleteOwned(ctx context.Context, id, ownerId int64) ([]uint64, error) {
	chunkIds := make([]uint64, 0)
	var docIds []int64
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var owner struct {
			Id int64 `db:"id"`
		}
		if err := session.QueryRowCtx(ctx, &owner, "SELECT id FROM ai_kb WHERE id=? AND owner_id=? AND deleted_at IS NULL FOR UPDATE", id, ownerId); err != nil {
			return err
		}
		// 与创建、重试和摄取终态统一按 KB→doc→chunk 加锁，避免删除与摄取交叉。
		if err := session.QueryRowsCtx(ctx, &docIds, "SELECT id FROM ai_kb_doc WHERE kb_id=? ORDER BY id FOR UPDATE", id); err != nil {
			return err
		}
		rows := make([]struct {
			Id uint64 `db:"id"`
		}, 0)
		if err := session.QueryRowsCtx(ctx, &rows, "SELECT id FROM ai_kb_chunk WHERE kb_id=?", id); err != nil {
			return err
		}
		for _, r := range rows {
			chunkIds = append(chunkIds, r.Id)
		}
		oldIds, err := pendingKbJobVectorIDs(ctx, session, id, 0)
		if err != nil {
			return err
		}
		chunkIds = append(chunkIds, oldIds...)
		if _, err := session.ExecCtx(ctx, "DELETE FROM ai_kb_chunk WHERE kb_id=?", id); err != nil {
			return err
		}
		if _, err := session.ExecCtx(ctx, "UPDATE ai_kb_doc SET deleted_at=NOW() WHERE kb_id=? AND deleted_at IS NULL", id); err != nil {
			return err
		}
		if _, err := session.ExecCtx(ctx, "UPDATE ai_kb SET deleted_at=NOW() WHERE id=?", id); err != nil {
			return err
		}
		_, err = session.ExecCtx(ctx, "UPDATE ai_kb_ingest_job SET status='cancelled',error='知识库已删除',completed_at=NOW(),lease_until=NULL WHERE kb_id=? AND status IN ('queued','running')", id)
		return err
	})
	if err != nil {
		return nil, err
	}
	keys := []string{fmt.Sprintf("%s%d", cacheGozeroAdminAiKbIdPrefix, id)}
	for _, docId := range docIds {
		keys = append(keys, fmt.Sprintf("%s%d", cacheGozeroAdminAiKbDocIdPrefix, docId))
	}
	if err := m.DelCacheCtx(ctx, keys...); err != nil {
		return chunkIds, &KbDocCommittedError{Err: err}
	}
	return chunkIds, nil
}
