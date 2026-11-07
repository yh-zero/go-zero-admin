package model

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// 仅替换故障缓存边界，事务和最终状态读取都使用真实隔离 MySQL。
type failedKbCache struct{ cache.Cache }

func (*failedKbCache) DelCtx(context.Context, ...string) error {
	return errors.New("isolated cache invalidation failure")
}

func TestKbDocCommittedCacheFailureReturnsCommittedData(t *testing.T) {
	dsn := os.Getenv("GOZERO_AI_TEST_DSN")
	if dsn == "" {
		t.Skip("requires isolated GOZERO_AI_TEST_DSN")
	}
	sqlx.DisableLog()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	conn := sqlx.NewSqlConnFromDB(db)
	m := &customAiKbDocModel{conn: conn, defaultAiKbDocModel: &defaultAiKbDocModel{CachedConn: sqlc.NewConnWithCache(conn, &failedKbCache{})}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	owner := time.Now().UnixNano()
	result, err := db.ExecContext(ctx, "INSERT INTO ai_kb(owner_id,name) VALUES (?,'缓存故障隔离')", owner)
	if err != nil {
		t.Fatal(err)
	}
	kb, _ := result.LastInsertId()
	t.Logf("事务缓存故障回归 owner_id=%d kb_id=%d；缓存为本地故障桩", owner, kb)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		for _, q := range []string{"DELETE FROM ai_kb_chunk WHERE kb_id=?", "DELETE FROM ai_kb_doc WHERE kb_id=?", "DELETE FROM ai_kb WHERE id=?"} {
			if _, err := db.ExecContext(cleanupCtx, q, kb); err != nil {
				t.Error(err)
			}
		}
		for _, q := range []string{"SELECT COUNT(*) FROM ai_kb_chunk WHERE kb_id=?", "SELECT COUNT(*) FROM ai_kb_doc WHERE kb_id=?", "SELECT COUNT(*) FROM ai_kb WHERE id=?"} {
			var remaining int
			if err := db.QueryRowContext(cleanupCtx, q, kb).Scan(&remaining); err != nil || remaining != 0 {
				t.Error("事务故障回归数据清理后仍存在或不可核对")
			}
		}
	})
	id, ids, created, err := m.InsertWithChunks(ctx, owner, &KbDocRow{KbId: kb, Title: "文档", Content: "正文", EmbeddingModel: "local-model"}, []KbChunkRow{{Content: "正文"}})
	if err != nil || !created {
		t.Fatalf("fixture insert created=%v err=%v", created, err)
	}
	if err := m.MarkFailed(ctx, id, ids, "本地失败"); !KbDocCommitted(err) {
		t.Fatalf("failed transition must report committed cache error: %v", err)
	}
	old, next, err := m.ReplaceChunksForRetry(ctx, kb, id, owner, []KbChunkRow{{Content: "新正文"}})
	if !KbDocCommitted(err) || len(old) != 1 || old[0] != ids[0] || len(next) != 1 || next[0] == ids[0] {
		t.Fatalf("retry lost committed IDs: old=%v new=%v err=%v", old, next, err)
	}
	if err := m.MarkIngested(ctx, id, next, "local-model"); !KbDocCommitted(err) {
		t.Fatalf("ready transition must report committed cache error: %v", err)
	}
	var state string
	if err := db.QueryRowContext(ctx, "SELECT status FROM ai_kb_doc WHERE id=?", id).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "ready" {
		t.Errorf("ready transaction rolled back: state=%s", state)
	}
	deleted, err := m.DeleteOwned(ctx, kb, id, owner)
	if !KbDocCommitted(err) || len(deleted) != 1 || deleted[0] != next[0] {
		t.Errorf("delete lost committed vector IDs: ids=%v err=%v", deleted, err)
	}
}
