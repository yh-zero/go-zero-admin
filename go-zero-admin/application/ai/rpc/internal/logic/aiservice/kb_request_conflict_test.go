package aiservicelogic

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/ai/rpc/internal/model"
)

// 两个事务都完成原请求查询后才继续，使跨知识库唯一键竞争可重复。
type kbRequestBarrierConn struct {
	sqlx.SqlConn
	arrived chan<- struct{}
	release <-chan struct{}
}

func (c *kbRequestBarrierConn) TransactCtx(ctx context.Context, fn func(context.Context, sqlx.Session) error) error {
	return c.SqlConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return fn(ctx, &kbRequestBarrierSession{Session: session, arrived: c.arrived, release: c.release})
	})
}

type kbRequestBarrierSession struct {
	sqlx.Session
	arrived chan<- struct{}
	release <-chan struct{}
}

func (s *kbRequestBarrierSession) QueryRowCtx(ctx context.Context, value any, query string, args ...any) error {
	err := s.Session.QueryRowCtx(ctx, value, query, args...)
	if errors.Is(err, model.ErrNotFound) && strings.Contains(query, "FROM ai_kb_ingest_job WHERE owner_id=? AND request_id=?") {
		select {
		case s.arrived <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case <-s.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

// 隐藏事务内原请求读取以重现唯一键兜底，并可在作业写入处验证失败回滚。
type kbRequestFaultConn struct {
	sqlx.SqlConn
	insertError error
	rechecks    int
}

func (c *kbRequestFaultConn) TransactCtx(ctx context.Context, fn func(context.Context, sqlx.Session) error) error {
	return c.SqlConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return fn(ctx, &kbRequestFaultSession{Session: session, insertError: c.insertError})
	})
}

func (c *kbRequestFaultConn) QueryRowCtx(ctx context.Context, value any, query string, args ...any) error {
	if strings.Contains(query, "FROM ai_kb_ingest_job WHERE owner_id=? AND request_id=?") {
		c.rechecks++
	}
	return c.SqlConn.QueryRowCtx(ctx, value, query, args...)
}

type kbRequestFaultSession struct {
	sqlx.Session
	insertError error
}

func (s *kbRequestFaultSession) QueryRowCtx(ctx context.Context, value any, query string, args ...any) error {
	if strings.Contains(query, "FROM ai_kb_ingest_job WHERE owner_id=? AND request_id=?") {
		return model.ErrNotFound
	}
	return s.Session.QueryRowCtx(ctx, value, query, args...)
}

func (s *kbRequestFaultSession) ExecCtx(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if s.insertError != nil && strings.HasPrefix(query, "INSERT INTO ai_kb_ingest_job(") {
		return nil, s.insertError
	}
	return s.Session.ExecCtx(ctx, query, args...)
}

func TestKbBackgroundDuplicateRequestRechecksOriginalPayloadAndPreservesErrors(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	r := model.KbEnqueueRequest{KbId: h.kb, OwnerId: h.actor.UserId, Operation: "create", RequestId: uuid.NewString(), Title: "原始资料", Content: "原始正文", EmbeddingModel: "local-test"}
	accepted, err := h.s.KbIngestQueue.Enqueue(ctx, r, chunkText(r.Content))
	if err != nil {
		t.Fatal(err)
	}
	storageFailure := errors.New("隔离作业写入故障")
	duplicateWithoutWinner := &mysql.MySQLError{Number: 1062, Message: "isolated duplicate without accepted request"}
	for _, scenario := range []string{"same_payload", "changed_payload", "storage_error", "duplicate_without_request"} {
		t.Run(scenario, func(t *testing.T) {
			candidate := r
			conn := &kbRequestFaultConn{SqlConn: h.s.DB}
			switch scenario {
			case "changed_payload":
				candidate.Content = "不同正文"
			case "storage_error":
				candidate.Content = "写入必须回滚"
				conn.insertError = storageFailure
			case "duplicate_without_request":
				candidate.RequestId = uuid.NewString()
				candidate.Content = "无已接收请求"
				conn.insertError = duplicateWithoutWinner
			}
			queue := model.NewKbIngestQueue(conn, h.s.AiKbDocModel)
			replayed, err := queue.Enqueue(ctx, candidate, chunkText(candidate.Content))
			switch scenario {
			case "same_payload":
				if err != nil || replayed == nil || replayed.Job.Id != accepted.Job.Id || replayed.Doc.Id != accepted.Doc.Id || replayed.BatchId != r.RequestId {
					t.Errorf("相同原始载荷必须返回已接收作业，不将事务内DocId用于哈希: result=%v err=%v", replayed, err)
				}
			case "changed_payload":
				if !hasKbCode(kbQueueError(err), 400) {
					t.Errorf("已存在的不同载荷必须返回400: %v", err)
				}
			case "storage_error":
				if !errors.Is(err, storageFailure) || conn.rechecks != 0 {
					t.Errorf("普通写入故障不得重查并吞掉错误: err=%v rechecks=%d", err, conn.rechecks)
				}
			case "duplicate_without_request":
				if !errors.Is(err, duplicateWithoutWinner) {
					t.Errorf("无权威请求记录的重复键错误必须保留: %v", err)
				}
			}
			for _, query := range []string{
				"SELECT COUNT(*) FROM ai_kb_ingest_job WHERE owner_id=?",
				"SELECT COUNT(*) FROM ai_kb_ingest_batch WHERE owner_id=?",
				"SELECT COUNT(*) FROM ai_kb_doc d JOIN ai_kb k ON k.id=d.kb_id WHERE k.owner_id=?",
				"SELECT COUNT(*) FROM ai_kb_chunk c JOIN ai_kb k ON k.id=c.kb_id WHERE k.owner_id=?",
			} {
				var count int64
				if err := h.db.QueryRowContext(ctx, query, h.actor.UserId).Scan(&count); err != nil || count != 1 {
					t.Error("失败事务必须完整回滚，只保留原始作业、批次、文档和分块")
				}
			}
		})
	}
	if h.calls.Load() != 0 {
		t.Fatal("重复请求与回滚验证不应调用模型")
	}
}

func TestKbBackgroundConcurrentRequestAcrossKnowledgeBasesIsConflict(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	inserted, err := h.db.ExecContext(ctx, "INSERT INTO ai_kb(owner_id,name) VALUES (?,?)", h.actor.UserId, "跨库请求冲突隔离回归")
	if err != nil {
		t.Fatal(err)
	}
	otherKb, err := inserted.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("跨库冲突回归 owner_id=%d kb_ids=%d,%d", h.actor.UserId, h.kb, otherKb)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		var docIds []int64
		if err := h.s.DB.QueryRowsCtx(cleanup, &docIds, "SELECT id FROM ai_kb_doc WHERE kb_id=?", otherKb); err != nil {
			t.Error("读取本轮第二知识库文档ID失败")
		}
		for _, query := range []string{
			"DELETE FROM ai_kb_ingest_job WHERE kb_id=?",
			"DELETE FROM ai_kb_ingest_batch WHERE kb_id=?",
			"DELETE FROM ai_kb_chunk WHERE kb_id=?",
			"DELETE FROM ai_kb_doc WHERE kb_id=?",
			"DELETE FROM ai_kb WHERE id=?",
		} {
			if _, err := h.db.ExecContext(cleanup, query, otherKb); err != nil {
				t.Error("清理本轮第二知识库数据失败")
			}
		}
		keys := []string{fmt.Sprintf("cache:gozeroAdmin:aiKb:id:%d", otherKb)}
		for _, id := range docIds {
			keys = append(keys, fmt.Sprintf("cache:gozeroAdmin:aiKbDoc:id:%d", id))
		}
		for _, key := range keys {
			if _, err := h.s.Redis.DelCtx(cleanup, key); err != nil {
				t.Error("清理本轮第二知识库缓存失败")
			}
			if exists, err := h.s.Redis.ExistsCtx(cleanup, key); err != nil || exists {
				t.Error("本轮第二知识库缓存清理未确认")
			}
		}
		for _, query := range []string{
			"SELECT COUNT(*) FROM ai_kb_ingest_job WHERE kb_id=?",
			"SELECT COUNT(*) FROM ai_kb_ingest_batch WHERE kb_id=?",
			"SELECT COUNT(*) FROM ai_kb_chunk WHERE kb_id=?",
			"SELECT COUNT(*) FROM ai_kb_doc WHERE kb_id=?",
			"SELECT COUNT(*) FROM ai_kb WHERE id=?",
		} {
			var remaining int64
			if err := h.db.QueryRowContext(cleanup, query, otherKb).Scan(&remaining); err != nil || remaining != 0 {
				t.Error("本轮第二知识库数据清理未确认")
			}
		}
	})
	// 复用各库已有文档，避免不存在文档的间隙锁干扰请求唯一键竞争。
	for _, kbId := range []int64{h.kb, otherKb} {
		if _, err := h.db.ExecContext(ctx, "INSERT INTO ai_kb_doc(kb_id,title,content,status,ingest_stage,embedding_model) VALUES (?,'冲突资料','相同正文','ready','ready','local-test')", kbId); err != nil {
			t.Fatal(err)
		}
	}
	arrived, release := make(chan struct{}, 2), make(chan struct{})
	conn := &kbRequestBarrierConn{SqlConn: h.s.DB, arrived: arrived, release: release}
	queue := model.NewKbIngestQueue(conn, h.s.AiKbDocModel)
	requestId := uuid.NewString()
	results := make(chan error, 2)
	for _, kbId := range []int64{h.kb, otherKb} {
		go func() {
			_, err := queue.Enqueue(ctx, model.KbEnqueueRequest{KbId: kbId, OwnerId: h.actor.UserId, Operation: "create", RequestId: requestId, Title: "冲突资料", Content: "相同正文", EmbeddingModel: "local-test"}, nil)
			results <- err
		}()
	}
	for range 2 {
		select {
		case <-arrived:
		case <-ctx.Done():
			t.Fatal("两个事务未到达请求查询屏障")
		}
	}
	close(release)
	var failures []error
	for range 2 {
		if err := <-results; err != nil {
			failures = append(failures, err)
		}
	}
	if len(failures) != 1 {
		t.Fatalf("竞争应只有一个失败，实际=%d", len(failures))
	}
	var sqlError *mysql.MySQLError
	if errors.As(failures[0], &sqlError) {
		t.Logf("竞争失败的底层MySQL错误码=%d", sqlError.Number)
	}
	for _, query := range []string{"SELECT COUNT(*) FROM ai_kb_ingest_job WHERE owner_id=?", "SELECT COUNT(*) FROM ai_kb_ingest_batch WHERE owner_id=?"} {
		var count int64
		if err := h.db.QueryRowContext(ctx, query, h.actor.UserId).Scan(&count); err != nil || count != 1 {
			t.Fatal("失败事务必须回滚，只保留一个批次和一个作业")
		}
	}
	if mapped := kbQueueError(failures[0]); !hasKbCode(mapped, 400) {
		t.Fatalf("同owner同request_id跨库并发冲突必须返回400，实际=%v", mapped)
	}
	if h.calls.Load() != 0 {
		t.Fatal("并发接收验证不应调用模型")
	}
}
