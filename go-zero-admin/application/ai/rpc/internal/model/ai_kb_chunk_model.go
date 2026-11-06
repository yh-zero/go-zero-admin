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
		// FindReadyByIDs 按 ID 批量取可展示的 chunk（JOIN 复核归属与软删），不存在/不可用的 ID 静默跳过
		FindReadyByIDs(ctx context.Context, ids []uint64, ownerId int64) ([]*KbChunkRow, error)
		// ListReadyIDs 全量 ready chunk ID（对账用）
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

func (m *customAiKbChunkModel) FindReadyByIDs(ctx context.Context, ids []uint64, ownerId int64) ([]*KbChunkRow, error) {
	rows := make([]*KbChunkRow, 0)
	if len(ids) == 0 {
		return rows, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids)+1)
	args = append(args, ownerId)
	for _, id := range ids {
		args = append(args, id)
	}
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT c.id,c.doc_id,c.kb_id,c.seq,c.content,c.token_count,d.title AS doc_title "+
			"FROM ai_kb_chunk c "+
			"JOIN ai_kb_doc d ON c.doc_id=d.id AND d.deleted_at IS NULL "+
			"JOIN ai_kb k ON c.kb_id=k.id AND k.deleted_at IS NULL "+
			"WHERE k.owner_id=? AND c.status='ready' AND c.id IN ("+placeholders+")", args...)
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
