package model

import (
	"context"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AiKbChunkModel = (*customAiKbChunkModel)(nil)

type (
	// AiKbChunkModel 知识库分块数据层；向量索引命中后回表复核用（status=ready + 归属 + 未删除）
	AiKbChunkModel interface {
		aiKbChunkModel
		// FindReadyByIDs 复核 owner、指定库（0 为全部）、文档/分块 ready、库一致与软删；不可用 ID 静默跳过。
		FindReadyByIDs(ctx context.Context, ids []uint64, ownerId, kbId int64) ([]*KbChunkRow, error)
		// FindRetainedIndexIDs 对账时保留未软删且归属一致的 pending/ready 分块，避免清理正在写入的索引。
		FindRetainedIndexIDs(ctx context.Context, ids []uint64) (map[uint64]struct{}, error)
		// ListReadyIDs 只统计 ready 分块，不能作为在线索引删除依据。
		ListReadyIDs(ctx context.Context) (map[uint64]struct{}, error)
	}

	customAiKbChunkModel struct {
		*defaultAiKbChunkModel
		conn sqlx.SqlConn
	}

	// KbChunkRow 分块行
	KbChunkRow struct {
		Id         int64  `db:"id"`
		DocId      int64  `db:"doc_id"`
		KbId       int64  `db:"kb_id"`
		Seq        int64  `db:"seq"`
		Content    string `db:"content"`
		TokenCount int64  `db:"token_count"`
		DocTitle   string `db:"doc_title"`
	}
)

// NewAiKbChunkModel returns a model for the database table.
func NewAiKbChunkModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiKbChunkModel {
	return &customAiKbChunkModel{defaultAiKbChunkModel: newAiKbChunkModel(conn, c, opts...), conn: conn}
}

// 任一批SQL或取消失败都丢弃部分结果，调用方不得据不完整事实源删除向量。
func (m *customAiKbChunkModel) FindRetainedIndexIDs(ctx context.Context, ids []uint64) (map[uint64]struct{}, error) {
	retained := make(map[uint64]struct{}, len(ids))
	for start := 0; start < len(ids); start += 500 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		batch := ids[start:min(start+500, len(ids))]
		args := make([]any, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		rows := make([]struct {
			Id uint64 `db:"id"`
		}, 0, len(batch))
		placeholders := strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")
		if err := m.QueryRowsNoCacheCtx(ctx, &rows,
			"SELECT c.id FROM ai_kb_chunk c "+
				"JOIN ai_kb_doc d ON c.doc_id=d.id AND d.kb_id=c.kb_id AND d.deleted_at IS NULL AND d.status IN ('pending','ready') "+
				"JOIN ai_kb k ON c.kb_id=k.id AND k.deleted_at IS NULL "+
				"WHERE c.status IN ('pending','ready') AND c.id IN ("+placeholders+")", args...); err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, row := range rows {
			retained[row.Id] = struct{}{}
		}
	}
	return retained, nil
}

func (m *customAiKbChunkModel) FindReadyByIDs(ctx context.Context, ids []uint64, ownerId, kbId int64) ([]*KbChunkRow, error) {
	rows := make([]*KbChunkRow, 0)
	if len(ids) == 0 {
		return rows, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+2)
	args = append(args, ownerId)
	kbFilter := ""
	if kbId > 0 {
		kbFilter = " AND c.kb_id=?"
		args = append(args, kbId)
	}
	for _, id := range ids {
		args = append(args, id)
	}
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT c.id,c.doc_id,c.kb_id,c.seq,c.content,c.token_count,d.title AS doc_title "+
			"FROM ai_kb_chunk c "+
			"JOIN ai_kb_doc d ON c.doc_id=d.id AND d.kb_id=c.kb_id AND d.status='ready' AND d.deleted_at IS NULL "+
			"JOIN ai_kb k ON c.kb_id=k.id AND k.deleted_at IS NULL "+
			"WHERE k.owner_id=?"+kbFilter+" AND c.status='ready' AND c.id IN ("+placeholders+")", args...)
	return rows, err
}

func (m *customAiKbChunkModel) ListReadyIDs(ctx context.Context) (map[uint64]struct{}, error) {
	rows := make([]struct {
		Id uint64 `db:"id"`
	}, 0)
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT id FROM ai_kb_chunk WHERE status='ready'"); err != nil {
		return nil, err
	}
	out := make(map[uint64]struct{}, len(rows))
	for _, r := range rows {
		out[r.Id] = struct{}{}
	}
	return out, nil
}
