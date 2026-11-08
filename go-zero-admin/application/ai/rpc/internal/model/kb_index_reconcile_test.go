package model

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type retainedIndexConn struct {
	sqlx.SqlConn
	batches []int
	failAt  int
	after   func()
}

func (c *retainedIndexConn) QueryRowsCtx(ctx context.Context, v any, query string, args ...any) error {
	c.batches = append(c.batches, len(args))
	if c.failAt == len(c.batches) {
		return errors.New("isolated retained lookup failure")
	}
	err := c.SqlConn.QueryRowsCtx(ctx, v, query, args...)
	if c.after != nil {
		c.after()
	}
	return err
}

func retainedIndexFixture(t *testing.T) (*customAiKbChunkModel, *retainedIndexConn, *sql.DB, []int64) {
	t.Helper()
	dsn := os.Getenv("GOZERO_AI_TEST_DSN")
	if dsn == "" {
		t.Skip("未设置隔离 GOZERO_AI_TEST_DSN，跳过保留索引事实源回归")
	}
	sqlx.DisableLog()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("隔离数据库初始化失败")
	}
	t.Cleanup(func() { _ = db.Close() })
	kbs := make([]int64, 0, 3)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for _, id := range kbs {
			for _, query := range []string{"DELETE FROM ai_kb_chunk WHERE kb_id=?", "DELETE FROM ai_kb_doc WHERE kb_id=?", "DELETE FROM ai_kb WHERE id=?"} {
				if _, err := db.ExecContext(ctx, query, id); err != nil {
					t.Error("隔离索引事实源清理失败")
				}
			}
			var count int
			if err := db.QueryRowContext(ctx, "SELECT (SELECT COUNT(*) FROM ai_kb_chunk WHERE kb_id=?)+(SELECT COUNT(*) FROM ai_kb_doc WHERE kb_id=?)+(SELECT COUNT(*) FROM ai_kb WHERE id=?)", id, id, id).Scan(&count); err != nil || count != 0 {
				t.Error("隔离索引事实源清理核对失败")
			}
		}
	})
	for range 3 {
		result, err := db.ExecContext(context.Background(), "INSERT INTO ai_kb(owner_id,name) VALUES (?,'索引对账事实源回归')", time.Now().UnixNano())
		if err != nil {
			t.Fatal("隔离知识库创建失败")
		}
		id, _ := result.LastInsertId()
		kbs = append(kbs, id)
	}
	conn := &retainedIndexConn{SqlConn: sqlx.NewSqlConnFromDB(db)}
	m := &customAiKbChunkModel{conn: conn, defaultAiKbChunkModel: &defaultAiKbChunkModel{CachedConn: sqlc.NewConnWithCache(conn, nil)}}
	return m, conn, db, kbs
}

func addRetainedIndexChunk(t *testing.T, db *sql.DB, docKB, chunkKB int64, docStatus, chunkStatus string, deleted bool) uint64 {
	t.Helper()
	result, err := db.ExecContext(context.Background(), "INSERT INTO ai_kb_doc(kb_id,title,content,status,ingest_stage) VALUES (?,'索引事实源','本地测试',?,'indexing')", docKB, docStatus)
	if err != nil {
		t.Fatal("隔离文档创建失败")
	}
	doc, _ := result.LastInsertId()
	if deleted {
		if _, err := db.ExecContext(context.Background(), "UPDATE ai_kb_doc SET deleted_at=NOW() WHERE id=?", doc); err != nil {
			t.Fatal("隔离文档软删失败")
		}
	}
	result, err = db.ExecContext(context.Background(), "INSERT INTO ai_kb_chunk(doc_id,kb_id,seq,content,status) VALUES (?,?,0,'本地测试',?)", doc, chunkKB, chunkStatus)
	if err != nil {
		t.Fatal("隔离分块创建失败")
	}
	id, _ := result.LastInsertId()
	return uint64(id)
}

func TestFindRetainedIndexIDsKeepsPendingIndexingAndRechecksOwnership(t *testing.T) {
	m, conn, db, kbs := retainedIndexFixture(t)
	var ids, want []uint64
	for _, state := range [][2]string{{"pending", "pending"}, {"pending", "ready"}, {"ready", "pending"}, {"ready", "ready"}} {
		id := addRetainedIndexChunk(t, db, kbs[0], kbs[0], state[0], state[1], false)
		ids, want = append(ids, id), append(want, id)
	}
	ids = append(ids,
		addRetainedIndexChunk(t, db, kbs[0], kbs[0], "failed", "ready", false),
		addRetainedIndexChunk(t, db, kbs[0], kbs[0], "ready", "failed", false),
		addRetainedIndexChunk(t, db, kbs[0], kbs[0], "ready", "ready", true),
		addRetainedIndexChunk(t, db, kbs[1], kbs[0], "ready", "ready", false),
		addRetainedIndexChunk(t, db, kbs[2], kbs[2], "ready", "ready", false),
		1<<62,
	)
	if _, err := db.ExecContext(context.Background(), "UPDATE ai_kb SET deleted_at=NOW() WHERE id=?", kbs[2]); err != nil {
		t.Fatal("隔离知识库软删失败")
	}
	retained, err := m.FindRetainedIndexIDs(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]uint64, 0, len(retained))
	for id := range retained {
		got = append(got, id)
	}
	slices.Sort(got)
	if !slices.Equal(got, want) || !slices.Equal(conn.batches, []int{10}) {
		t.Fatalf("indexing窗口必须保留，失败/软删/跨库须排除: got=%v want=%v batches=%v", got, want, conn.batches)
	}
}

func TestFindRetainedIndexIDsBoundsBatchesAndDiscardsPartialFailure(t *testing.T) {
	m, conn, db, kbs := retainedIndexFixture(t)
	good := addRetainedIndexChunk(t, db, kbs[0], kbs[0], "pending", "pending", false)
	ids := make([]uint64, 1001)
	for i := range ids {
		ids[i] = 1<<62 + uint64(i)
	}
	ids[0] = good
	retained, err := m.FindRetainedIndexIDs(context.Background(), ids)
	if err != nil || len(retained) != 1 || !slices.Equal(conn.batches, []int{500, 500, 1}) {
		t.Fatalf("1001项必须有界分成500/500/1，且保留真实项: retained=%v batches=%v err=%v", retained, conn.batches, err)
	}
	conn.batches, conn.failAt = nil, 2
	retained, err = m.FindRetainedIndexIDs(context.Background(), ids)
	if err == nil || retained != nil || !slices.Equal(conn.batches, []int{500, 500}) || !strings.Contains(err.Error(), "isolated retained lookup failure") {
		t.Fatalf("第二批错误不能把第一批或空结果当完整事实源: retained=%v batches=%v err=%v", retained, conn.batches, err)
	}
	conn.batches, conn.failAt = nil, 0
	ctx, cancel := context.WithCancel(context.Background())
	conn.after = cancel
	retained, err = m.FindRetainedIndexIDs(ctx, ids)
	if !errors.Is(err, context.Canceled) || retained != nil || !slices.Equal(conn.batches, []int{500}) {
		t.Fatalf("取消必须停止后续SQL且不返回部分结果: retained=%v batches=%v err=%v", retained, conn.batches, err)
	}
	conn.batches = nil
	ctx, cancel = context.WithCancel(context.Background())
	conn.after = cancel
	retained, err = m.FindRetainedIndexIDs(ctx, ids[:1])
	if !errors.Is(err, context.Canceled) || retained != nil || !slices.Equal(conn.batches, []int{1}) {
		t.Fatalf("最后一批返回时已取消，也不能交付可用于删除的成功结果: retained=%v batches=%v err=%v", retained, conn.batches, err)
	}
}

func TestFindRetainedIndexIDsEmptyAndCancelledSkipSQL(t *testing.T) {
	m := &customAiKbChunkModel{}
	retained, err := m.FindRetainedIndexIDs(context.Background(), nil)
	if err != nil || retained == nil || len(retained) != 0 {
		t.Fatalf("空输入应返回空集合且不访问SQL: retained=%v err=%v", retained, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	retained, err = m.FindRetainedIndexIDs(ctx, []uint64{1})
	if !errors.Is(err, context.Canceled) || retained != nil {
		t.Fatalf("已取消输入不能访问SQL或返回成功结果: retained=%v err=%v", retained, err)
	}
}
