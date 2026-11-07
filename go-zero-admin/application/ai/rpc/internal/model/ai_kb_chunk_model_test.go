package model

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestFindReadyChunksRechecksFactSource(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	execID := func(query string, args ...any) int64 {
		t.Helper()
		result, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			t.Fatal(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	owner := time.Now().UnixNano()
	kbs := []int64{}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, kb := range kbs {
			for _, q := range []string{"DELETE FROM ai_kb_chunk WHERE kb_id=?", "DELETE FROM ai_kb_doc WHERE kb_id=?", "DELETE FROM ai_kb WHERE id=?"} {
				if _, err := db.ExecContext(cleanup, q, kb); err != nil {
					t.Error(err)
				}
			}
			for _, q := range []string{"SELECT COUNT(*) FROM ai_kb_chunk WHERE kb_id=?", "SELECT COUNT(*) FROM ai_kb_doc WHERE kb_id=?", "SELECT COUNT(*) FROM ai_kb WHERE id=?"} {
				var count int
				if err := db.QueryRowContext(cleanup, q, kb).Scan(&count); err != nil || count != 0 {
					t.Error("fixture cleanup failed")
				}
			}
		}
	})
	for _, uid := range []int64{owner, owner, owner + 1, owner} {
		kbs = append(kbs, execID("INSERT INTO ai_kb(owner_id,name) VALUES (?,'召回事实源隔离')", uid))
	}
	if _, err := db.ExecContext(ctx, "UPDATE ai_kb SET deleted_at=NOW() WHERE id=?", kbs[3]); err != nil {
		t.Fatal(err)
	}
	ids := []uint64{}
	add := func(docKB, chunkKB int64, docStatus, chunkStatus string, deleted bool) uint64 {
		doc := execID("INSERT INTO ai_kb_doc(kb_id,title,content,status) VALUES (?,'文档','正文',?)", docKB, docStatus)
		if deleted {
			if _, err := db.ExecContext(ctx, "UPDATE ai_kb_doc SET deleted_at=NOW() WHERE id=?", doc); err != nil {
				t.Fatal(err)
			}
		}
		chunk := uint64(execID("INSERT INTO ai_kb_chunk(doc_id,kb_id,seq,content,status) VALUES (?,?,0,'正文',?)", doc, chunkKB, chunkStatus))
		ids = append(ids, chunk)
		return chunk
	}
	good := add(kbs[0], kbs[0], "ready", "ready", false)
	otherKB := add(kbs[1], kbs[1], "ready", "ready", false)
	add(kbs[0], kbs[0], "pending", "ready", false)
	add(kbs[0], kbs[0], "failed", "ready", false)
	add(kbs[0], kbs[0], "ready", "ready", true)
	add(kbs[0], kbs[0], "ready", "pending", false)
	add(kbs[0], kbs[0], "ready", "failed", false)
	add(kbs[1], kbs[0], "ready", "ready", false)
	add(kbs[2], kbs[2], "ready", "ready", false)
	add(kbs[3], kbs[3], "ready", "ready", false)
	conn := sqlx.NewSqlConnFromDB(db)
	m := &customAiKbChunkModel{conn: conn, defaultAiKbChunkModel: &defaultAiKbChunkModel{CachedConn: sqlc.NewConnWithCache(conn, nil)}}
	rows, err := m.FindReadyByIDs(ctx, ids, owner, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := []uint64{}
	for _, row := range rows {
		got = append(got, uint64(row.Id))
	}
	slices.Sort(got)
	if !slices.Equal(got, []uint64{good, otherKB}) {
		t.Fatalf("fact source filter IDs=%v want=%v", got, []uint64{good, otherKB})
	}
	rows, err = m.FindReadyByIDs(ctx, ids, owner, kbs[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Id != int64(good) {
		t.Fatalf("specified KB leaked other KB chunks: %+v", rows)
	}
	rows, err = m.FindReadyByIDs(ctx, ids, owner+2, 0)
	if err != nil || len(rows) != 0 {
		t.Fatalf("foreign owner result=%+v err=%v", rows, err)
	}
	rows, err = m.FindReadyByIDs(ctx, nil, owner, 0)
	if err != nil || rows == nil || len(rows) != 0 {
		t.Fatalf("empty input result=%+v err=%v", rows, err)
	}
	t.Logf("隔离回表 fixture kb_ids=%v；清理逐ID核对", kbs)
}
