package model

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type appReviewTestCache struct{ cache.Cache }

func (appReviewTestCache) DelCtx(context.Context, ...string) error { return nil }

// 同一连接的临时表隔离测试账号，断开连接后 MySQL 自动清理。
func TestAppReviewPreservesFirstDecision(t *testing.T) {
	dsn := os.Getenv("GOZERO_APP_TEST_DSN")
	if dsn == "" {
		t.Skip("requires GOZERO_APP_TEST_DSN for an isolated MySQL temporary table")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("无法连接测试数据库")
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, "CREATE TEMPORARY TABLE app_user (id BIGINT PRIMARY KEY, account VARCHAR(128), status VARCHAR(16), review_reason VARCHAR(512), updated_at DATETIME, deleted_at DATETIME) ENGINE=InnoDB"); err != nil {
		t.Fatal("无法创建隔离审核表")
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO app_user (id, account, status, review_reason) VALUES (1, 'test@example.test', 'pending', '')"); err != nil {
		t.Fatal(err)
	}
	users := &customAppUserModel{&defaultAppUserModel{CachedConn: sqlc.NewConnWithCache(sqlx.NewSqlConnFromDB(db), appReviewTestCache{})}}
	revocations := 0
	revoke := func(context.Context) error { revocations++; return nil }
	initial := ReviewDecision{Status: "pending"}
	first := ReviewDecision{Status: "rejected", Reason: "缺少申请资料"}
	if err := users.SetReview(ctx, 1, initial, first, revoke); err != nil {
		t.Fatal(err)
	}
	for _, stale := range []ReviewDecision{initial, {Status: first.Status, Reason: "旧说明"}} {
		if err := users.SetReview(ctx, 1, stale, ReviewDecision{Status: "active"}, revoke); !errors.Is(err, ErrAppReviewChanged) {
			t.Fatalf("过期审核应报告冲突，得到 %v", err)
		}
	}
	var saved ReviewDecision
	if err := db.QueryRowContext(ctx, "SELECT status,review_reason FROM app_user WHERE id=1").Scan(&saved.Status, &saved.Reason); err != nil {
		t.Fatal(err)
	}
	if saved != first || revocations != 1 {
		t.Fatalf("冲突改变了先保存的决定或会话：%+v，撤销次数 %d", saved, revocations)
	}
	if err := users.SetReview(ctx, 1, first, ReviewDecision{Status: "active"}, revoke); err != nil || revocations != 2 {
		t.Fatalf("读取最新决定后重新审核：%v，撤销次数 %d", err, revocations)
	}
}
