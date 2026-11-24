package model

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type auditTestCache struct{ cache.Cache }

func (auditTestCache) DelCtx(context.Context, ...string) error { return nil }

func TestAuditLogWhereValidationAndEscaping(t *testing.T) {
	where, args, err := auditLogWhere(AuditLogListFilter{Username: "admin", Method: "GET", Path: "100%!_%", StartTime: 10, EndTime: 20, MinDurationMs: 5, MaxDurationMs: 50})
	if err != nil || !strings.Contains(where, "path LIKE ? ESCAPE '!'") {
		t.Fatalf("unexpected where: %s %v", where, err)
	}
	if args[2] != "%"+auditPathEscaper.Replace("100%!_%")+"%" {
		t.Fatalf("path wildcard was not escaped: %v", args)
	}
	for _, filter := range []AuditLogListFilter{{StartTime: 20, EndTime: 10}, {MinDurationMs: 50, MaxDurationMs: 5}, {StartTime: -1}} {
		if _, _, err := auditLogWhere(filter); err != ErrInvalidAuditFilter {
			t.Fatalf("expected invalid filter: %+v", filter)
		}
	}
}

func TestAuditFiltersAndRetention(t *testing.T) {
	dsn := os.Getenv("GOZERO_ADMIN_TEST_DSN")
	if dsn == "" {
		t.Skip("需要隔离的 GOZERO_ADMIN_TEST_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	username := "audit-enhance-" + time.Now().Format("20060102150405.000000000")
	old := time.Now().Add(-72 * time.Hour)
	recent := time.Now().Add(-time.Hour)
	insert := func(method, path, ip string, duration int, created time.Time) {
		_, err := db.ExecContext(t.Context(), "INSERT INTO sys_audit_log (username,module,action,method,path,result,error_msg,ip,user_agent,duration_ms,created_at) VALUES (?,?,?,?,?,1,'',?,'test',?,?)", username, "sys/test", "read", method, path, ip, duration, created)
		if err != nil {
			t.Fatal(err)
		}
	}
	insert("GET", "/v1/sys/test/item", "127.0.0.1", 20, old)
	insert("POST", "/v1/sys/test/item", "127.0.0.2", 200, recent)
	t.Cleanup(func() { _, _ = db.Exec("DELETE FROM sys_audit_log WHERE username=?", username) })
	conn := sqlx.NewSqlConnFromDB(db)
	m := &customSysAuditLogModel{defaultSysAuditLogModel: &defaultSysAuditLogModel{CachedConn: sqlc.NewConnWithCache(conn, auditTestCache{})}}
	rows, total, err := m.ListAuditLogPage(t.Context(), AuditLogListFilter{
		Username: username, Method: "POST", Path: "/test/", IP: "127.0.0.2",
		StartTime: recent.Add(-time.Minute).Unix(), EndTime: recent.Add(time.Minute).Unix(), MinDurationMs: 100, MaxDurationMs: 300,
	}, 1, 20)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Method != "POST" {
		t.Fatalf("unexpected filtered result: rows=%v total=%d err=%v", rows, total, err)
	}
	cutoff := old.Add(time.Minute).Unix()
	_, expectedDeleted, err := m.ListAuditLogPage(t.Context(), AuditLogListFilter{EndTime: cutoff - 1}, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := m.PurgeBefore(t.Context(), cutoff, 10)
	if err != nil || deleted != expectedDeleted {
		t.Fatalf("unexpected retention delete: got=%d want=%d err=%v", deleted, expectedDeleted, err)
	}
	_, total, err = m.ListAuditLogPage(t.Context(), AuditLogListFilter{Username: username}, 1, 20)
	if err != nil || total != 1 {
		t.Fatalf("retention deleted wrong rows: total=%d err=%v", total, err)
	}
}
