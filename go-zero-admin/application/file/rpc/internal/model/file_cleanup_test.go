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

type cleanupTestCache struct{ cache.Cache }

func (cleanupTestCache) DelCtx(context.Context, ...string) error { return nil }

func TestFileCleanupListingSurvivesRefresh(t *testing.T) {
	dsn := os.Getenv("FILE_CLEANUP_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("需要隔离 MySQL 的 FILE_CLEANUP_TEST_MYSQL_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := t.Context()
	owner := time.Now().UnixNano()
	res, err := db.ExecContext(ctx, "INSERT INTO sys_file (bucket,object_key,uploader_id) VALUES ('private','cleanup-regression',?)", owner)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := db.Exec("DELETE FROM sys_file WHERE id=? AND uploader_id=?", id, owner); err != nil {
			t.Error(err)
		}
	})
	conn := sqlx.NewSqlConnFromDB(db)
	m := &customSysFileModel{conn: conn, defaultSysFileModel: &defaultSysFileModel{CachedConn: sqlc.NewConnWithCache(conn, cleanupTestCache{})}}
	check := func(viewer int64, wantCount int, pending bool) {
		t.Helper()
		rows, total, err := m.ListFilePage(ctx, FileListFilter{ViewerId: viewer, UploaderId: owner}, 1, 20)
		if err != nil || total != int64(wantCount) || len(rows) != wantCount {
			t.Fatalf("viewer=%d total=%d rows=%d err=%v", viewer, total, len(rows), err)
		}
		if wantCount > 0 && (rows[0].Id != id || rows[0].CleanupPending != pending) {
			t.Fatalf("unexpected row: %+v", rows[0])
		}
	}
	check(owner, 1, false)
	if _, err := m.MarkDeletedIfUnused(ctx, id, owner); err != nil {
		t.Fatal(err)
	}
	if _, err := m.FindLive(ctx, id); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("已删除文件仍可读取: %v", err)
	}
	check(owner, 1, true)
	check(owner+1, 0, false)
	check(0, 0, false)
	if _, err := m.MarkDeletedIfUnused(ctx, id, owner+1); !errors.Is(err, ErrFileForbidden) {
		t.Fatalf("其他用户不能重试: %v", err)
	}
	if _, err := m.MarkDeletedIfUnused(ctx, id, owner); err != nil {
		t.Fatal(err)
	}
	if err := m.ConfirmObjectCleanup(ctx, id); err != nil {
		t.Fatal(err)
	}
	check(owner, 0, false)
	row, err := m.MarkDeletedIfUnused(ctx, id, owner)
	if err != nil || !row.ObjectCleanedAt.Valid {
		t.Fatalf("完成回执未保留: %v", err)
	}
}
