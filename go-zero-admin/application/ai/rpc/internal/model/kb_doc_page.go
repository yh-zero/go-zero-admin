package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

type KbDocPage struct {
	Rows                          []*KbDocRow
	Total, Ready, Pending, Failed int64
	Signature                     string
}

// 两种投影共用同一有序页。仅散列可见元数据，排除正文，防止同秒替换或不同资料的相同计数掩盖变化。
const kbDocJSON = "CAST(JSON_OBJECT('Id',d.id,'KbId',d.kb_id,'Title',d.title,'Status',d.status,'Error',d.error," +
	"'ChunkCount',d.chunk_count,'EmbeddingModel',d.embedding_model,'Version',d.version,'Stage',d.ingest_stage," +
	"'CreatedAt',UNIX_TIMESTAMP(d.created_at),'UpdatedAt',UNIX_TIMESTAMP(d.updated_at)) AS CHAR)"

func (m *customAiKbDocModel) ReadPage(ctx context.Context, kbId, ownerId, page, size int64, query, status string, metaOnly bool) (*KbDocPage, error) {
	filter := "d.id IS NOT NULL"
	var filters []any
	if query != "" {
		filter += " AND d.title LIKE ? ESCAPE '!'"
		filters = append(filters, "%"+strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(query)+"%")
	}
	if status != "" {
		filter += " AND d.status=?"
		filters = append(filters, status)
	}
	// 聚合只扫描本库，分页查询独立使用资料索引。两者来自同一 SQL 快照，空页仍有统计，越权库没有记录。
	stats := "SELECT COUNT(CASE WHEN " + filter + " THEN 1 END) AS total," +
		"COUNT(CASE WHEN d.status='ready' THEN 1 END) AS ready,COUNT(CASE WHEN d.status='pending' THEN 1 END) AS pending,COUNT(CASE WHEN d.status='failed' THEN 1 END) AS failed " +
		"FROM ai_kb k LEFT JOIN ai_kb_doc d ON d.kb_id=k.id AND d.deleted_at IS NULL WHERE k.id=? AND k.owner_id=? AND k.deleted_at IS NULL GROUP BY k.id"
	pageSQL := "SELECT d.id," + kbDocJSON + " AS doc FROM ai_kb_doc d JOIN ai_kb k ON k.id=d.kb_id AND k.deleted_at IS NULL WHERE k.id=? AND k.owner_id=? AND d.deleted_at IS NULL AND " + filter + " ORDER BY d.id DESC LIMIT ? OFFSET ?"
	projection := "p.doc"
	if metaOnly {
		projection = "SHA2(p.doc,256)"
	}
	stmt := "WITH stats AS (" + stats + "),doc_page AS (" + pageSQL + ") SELECT stats.*,COALESCE(" + projection + ",'') AS payload FROM stats LEFT JOIN doc_page p ON TRUE ORDER BY p.id DESC"
	args := append([]any{}, filters...)
	args = append(args, kbId, ownerId, kbId, ownerId)
	args = append(args, filters...)
	args = append(args, size, (page-1)*size)
	var rows []struct {
		Total   int64  `db:"total"`
		Ready   int64  `db:"ready"`
		Pending int64  `db:"pending"`
		Failed  int64  `db:"failed"`
		Payload string `db:"payload"`
	}
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, stmt, args...); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrNotFound
	}
	result := &KbDocPage{Total: rows[0].Total, Ready: rows[0].Ready, Pending: rows[0].Pending, Failed: rows[0].Failed}
	hash := sha256.New()
	for _, r := range rows {
		if r.Payload == "" {
			continue
		}
		rowSHA := r.Payload
		if !metaOnly {
			var doc KbDocRow
			if err := json.Unmarshal([]byte(r.Payload), &doc); err != nil {
				return nil, err
			}
			result.Rows = append(result.Rows, &doc)
			digest := sha256.Sum256([]byte(r.Payload))
			rowSHA = hex.EncodeToString(digest[:])
		}
		_, _ = hash.Write([]byte(rowSHA))
	}
	result.Signature = hex.EncodeToString(hash.Sum(nil))
	return result, nil
}
