package svc

import (
	"context"
	"database/sql"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/ai/rpc/internal/model"
)

func TestKbIndexReconcileActualSQLKeepsIndexingWindow(t *testing.T) {
	dsn := os.Getenv("GOZERO_AI_TEST_DSN")
	redisAddress := os.Getenv("GOZERO_AI_TEST_REDIS")
	if dsn == "" || redisAddress == "" {
		t.Skip("未设置本轮隔离MySQL/Redis，未执行对账事实源联调")
	}
	sqlx.DisableLog()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("本轮隔离MySQL初始化失败")
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	execID := func(query string, args ...any) int64 {
		t.Helper()
		result, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			t.Fatal("本轮对账夹具写入失败")
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	kb := execID("INSERT INTO ai_kb(owner_id,name) VALUES (?,'索引对账窗口回归')", time.Now().UnixNano())
	t.Logf("本轮对账夹具kb_id=%d；结束逐ID清理并核对归零", kb)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, query := range []string{"DELETE FROM ai_kb_chunk WHERE kb_id=?", "DELETE FROM ai_kb_doc WHERE kb_id=?", "DELETE FROM ai_kb WHERE id=?"} {
			if _, err := db.ExecContext(cleanup, query, kb); err != nil {
				t.Error("按本轮知识库ID清理对账夹具失败")
			}
		}
		for _, query := range []string{"SELECT COUNT(*) FROM ai_kb_chunk WHERE kb_id=?", "SELECT COUNT(*) FROM ai_kb_doc WHERE kb_id=?", "SELECT COUNT(*) FROM ai_kb WHERE id=?"} {
			var count int
			if err := db.QueryRowContext(cleanup, query, kb).Scan(&count); err != nil || count != 0 {
				t.Error("本轮对账夹具未清理归零")
			}
		}
	})
	indexingDoc := execID("INSERT INTO ai_kb_doc(kb_id,title,content,status,ingest_stage) VALUES (?,'正在索引','正文','pending','indexing')", kb)
	pending := uint64(execID("INSERT INTO ai_kb_chunk(doc_id,kb_id,seq,content,status) VALUES (?,?,0,'正文','pending')", indexingDoc, kb))
	ready := uint64(execID("INSERT INTO ai_kb_chunk(doc_id,kb_id,seq,content,status) VALUES (?,?,1,'正文','ready')", indexingDoc, kb))
	failedDoc := execID("INSERT INTO ai_kb_doc(kb_id,title,content,status,ingest_stage) VALUES (?,'失败旧代次','正文','failed','failed')", kb)
	failed := uint64(execID("INSERT INTO ai_kb_chunk(doc_id,kb_id,seq,content,status) VALUES (?,?,0,'正文','failed')", failedDoc, kb))
	const absent uint64 = 1 << 62
	store := &reconcileVectorStore{ids: map[uint64]struct{}{pending: {}, ready: {}, failed: {}, absent: {}}}
	cacheConf := cache.CacheConf{{RedisConf: redis.RedisConf{Host: redisAddress, Type: "node"}, Weight: 100}}
	stats, err := reconcileKbIndex(ctx, store, model.NewAiKbChunkModel(sqlx.NewSqlConnFromDB(db), cacheConf))
	if err != nil || stats.Scanned != 4 || stats.Deleted != 2 || len(store.ids) != 2 || len(store.deletes) != 1 {
		t.Fatalf("实际SQL只能删除failed/不存在项: %+v err=%v", stats, err)
	}
	deleted := append([]uint64(nil), store.deletes[0]...)
	slices.Sort(deleted)
	if !slices.Equal(deleted, []uint64{failed, absent}) {
		t.Fatal("Upsert后尚未ready的实际分块被误删")
	}
}
