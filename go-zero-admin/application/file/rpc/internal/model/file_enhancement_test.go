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

type fileEnhanceTestCache struct{ cache.Cache }

func (fileEnhanceTestCache) DelCtx(context.Context, ...string) error { return nil }

func TestFileEnhancementFiltersStatsAndReferences(t *testing.T) {
	dsn := os.Getenv("FILE_CLEANUP_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("需要隔离 MySQL 的 FILE_CLEANUP_TEST_MYSQL_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := t.Context()
	username := "file-enhance-" + time.Now().Format("20060102150405.000000000")
	result, err := db.ExecContext(ctx, "INSERT INTO sys_user (username,password_hash,status) VALUES (?,'x',1)", username)
	if err != nil {
		t.Fatal(err)
	}
	uploader, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM sys_file_reference WHERE created_by=?", uploader)
		_, _ = db.Exec("DELETE FROM sys_file WHERE uploader_id=?", uploader)
		_, _ = db.Exec("DELETE FROM sys_user WHERE id=? AND username=?", uploader, username)
	})
	insert := func(name, mime, visibility, sha string, size int) int64 {
		res, err := db.ExecContext(ctx, "INSERT INTO sys_file (bucket,object_key,original_name,ext,mime,size,sha256,visibility,uploader_id) VALUES (?,?,?,?,?,?,?,?,?)", visibility, "test/"+name, name, ".txt", mime, size, sha, visibility, uploader)
		if err != nil {
			t.Fatal(err)
		}
		id, err := res.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	publicID := insert("enhance-public.txt", "text/plain", "public", "a", 100)
	privateID := insert("enhance-private.txt", "image/png", "private", "b", 200)
	conn := sqlx.NewSqlConnFromDB(db)
	m := &customSysFileModel{conn: conn, defaultSysFileModel: &defaultSysFileModel{CachedConn: sqlc.NewConnWithCache(conn, fileEnhanceTestCache{}), table: "`sys_file`"}}
	now := time.Now()
	rows, total, err := m.ListFilePage(ctx, FileListFilter{
		UploaderId: uploader, Name: "enhance", Mime: "text/plain", MinSize: 50, MaxSize: 150,
		StartTime: now.Add(-time.Hour).Unix(), EndTime: now.Add(time.Hour).Unix(),
	}, 1, 20)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Id != publicID || rows[0].UploaderName != username || rows[0].SHA256 != "a" {
		t.Fatalf("unexpected file filter result: rows=%v total=%d err=%v", rows, total, err)
	}
	if _, err := m.MarkDeletedIfUnused(ctx, privateID, uploader); err != nil {
		t.Fatal(err)
	}
	rows, total, err = m.ListFilePage(ctx, FileListFilter{ViewerId: uploader, UploaderId: uploader, CleanupPendingOnly: true}, 1, 20)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Id != privateID || !rows[0].CleanupPending {
		t.Fatalf("unexpected cleanup view: rows=%v total=%d err=%v", rows, total, err)
	}
	stats, err := m.GetFileStats(ctx, uploader)
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalFiles != 1 || stats.TotalBytes != 100 || stats.PublicFiles != 1 || stats.PrivateFiles != 0 || stats.CleanupPendingFiles != 1 || stats.CleanupPendingBytes != 200 {
		t.Fatalf("unexpected file stats: %+v", stats)
	}
	refModel := &customSysFileReferenceModel{defaultSysFileReferenceModel: &defaultSysFileReferenceModel{CachedConn: sqlc.NewConnWithCache(conn, fileEnhanceTestCache{}), table: "`sys_file_reference`"}, conn: conn}
	if _, err := db.ExecContext(ctx, "INSERT INTO sys_file_reference (file_id,ref_type,ref_id,created_by) VALUES (?,?,?,?)", publicID, "admin_avatar", uploader, uploader); err != nil {
		t.Fatal(err)
	}
	refs, err := refModel.ListFileReferences(ctx, publicID)
	if err != nil || len(refs) != 1 || refs[0].RefType != "admin_avatar" || refs[0].RefId != uploader {
		t.Fatalf("unexpected references: %+v err=%v", refs, err)
	}
	if _, err := m.MarkDeletedIfUnused(ctx, publicID, uploader); !errors.Is(err, ErrFileReferenced) {
		t.Fatalf("referenced file must not delete: %v", err)
	}
}
