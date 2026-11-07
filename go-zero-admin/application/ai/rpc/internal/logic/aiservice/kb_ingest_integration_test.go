package aiservicelogic

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/vectordb"
	"go-zero-admin/pkg/xerr"
)

// 只在显式隔离环境运行，Embedding HTTP 与向量索引均为本地桩。
type kbIngestHarness struct {
	t     *testing.T
	db    *sql.DB
	s     *svc.ServiceContext
	kb    int64
	actor *aipb.Actor
	calls atomic.Int64
	embed func(http.ResponseWriter, *http.Request)
	store *ingestVectorStore
	start sync.Once
}

func newKbIngestHarness(t *testing.T) *kbIngestHarness {
	t.Helper()
	sqlx.DisableLog()
	dsn, address := workspaceEnvironment(t)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	conn := sqlx.NewSqlConnFromDB(db)
	conf := cache.CacheConf{{RedisConf: redis.RedisConf{Host: address, Type: "node"}, Weight: 100}}
	rdb, err := redis.NewRedis(conf[0].RedisConf)
	if err != nil {
		t.Fatal(err)
	}
	h := &kbIngestHarness{t: t, db: db, actor: &aipb.Actor{UserId: time.Now().UnixNano()}, store: &ingestVectorStore{points: make(map[uint64]vectordb.Point)}}
	h.s = &svc.ServiceContext{DB: conn, Redis: rdb, AiKbModel: model.NewAiKbModel(conn, conf), AiKbDocModel: model.NewAiKbDocModel(conn, conf), AiKbChunkModel: model.NewAiKbChunkModel(conn, conf), VectorStore: h.store}
	h.s.Config.AI.EmbeddingModel = "text-embedding-v4"
	h.s.KbIngestQueue = model.NewKbIngestQueue(conn, h.s.AiKbDocModel)
	result, err := db.Exec("INSERT INTO ai_kb(owner_id,name) VALUES (?,?)", h.actor.UserId, "摄取隔离回归")
	if err != nil {
		t.Fatal(err)
	}
	h.kb, _ = result.LastInsertId()
	t.Logf("摄取隔离回归 owner_id=%d kb_id=%d；仅清理该知识库及其文档缓存", h.actor.UserId, h.kb)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		rows, err := db.QueryContext(ctx, "SELECT id FROM ai_kb_doc WHERE kb_id=?", h.kb)
		if err != nil {
			t.Error(err)
			return
		}
		var ids []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				t.Error(err)
			}
			ids = append(ids, id)
		}
		_ = rows.Close()
		for _, id := range ids {
			key := fmt.Sprintf("cache:gozeroAdmin:aiKbDoc:id:%d", id)
			if _, err := rdb.DelCtx(ctx, key); err != nil {
				t.Error(err)
			}
			if exists, err := rdb.ExistsCtx(ctx, key); err != nil || exists {
				t.Error("摄取测试文档缓存清理后仍存在或不可核对")
			}
		}
		for _, q := range []string{"DELETE FROM ai_kb_ingest_job WHERE kb_id=?", "DELETE FROM ai_kb_ingest_batch WHERE kb_id=?", "DELETE FROM ai_kb_chunk WHERE kb_id=?", "DELETE FROM ai_kb_doc WHERE kb_id=?", "DELETE FROM ai_kb WHERE id=?"} {
			if _, err := db.ExecContext(ctx, q, h.kb); err != nil {
				t.Error(err)
			}
		}
		if _, err := rdb.DelCtx(ctx, fmt.Sprintf("cache:gozeroAdmin:aiKb:id:%d", h.kb)); err != nil {
			t.Error(err)
		}
		if exists, err := rdb.ExistsCtx(ctx, fmt.Sprintf("cache:gozeroAdmin:aiKb:id:%d", h.kb)); err != nil || exists {
			t.Error("摄取测试知识库缓存清理后仍存在或不可核对")
		}
		for _, q := range []string{"SELECT COUNT(*) FROM ai_kb_ingest_job WHERE kb_id=?", "SELECT COUNT(*) FROM ai_kb_ingest_batch WHERE kb_id=?", "SELECT COUNT(*) FROM ai_kb_chunk WHERE kb_id=?", "SELECT COUNT(*) FROM ai_kb_doc WHERE kb_id=?", "SELECT COUNT(*) FROM ai_kb WHERE id=?"} {
			var remaining int
			if err := db.QueryRowContext(ctx, q, h.kb).Scan(&remaining); err != nil || remaining != 0 {
				t.Error("摄取测试数据清理后仍存在或不可核对")
			}
		}
		t.Logf("已核对清理 kb_id=%d 的文档/分块/知识库及 %d 个文档缓存键", h.kb, len(ids))
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.calls.Add(1)
		if h.embed != nil {
			h.embed(w, r)
			return
		}
		var body struct {
			Input []string `json:"input"`
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			http.Error(w, "bad input", 400)
			return
		}
		data := make([]map[string]any, len(body.Input))
		for i := range data {
			data[i] = map[string]any{"index": i, "embedding": []float32{1, 0}}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	t.Cleanup(server.Close)
	t.Logf("本地 Embedding 桩 %s；由 httptest cleanup 关闭", server.URL)
	h.s.Embedder, err = llm.NewEmbedder(llm.EmbedConfig{BaseURL: server.URL, APIKey: "local-test", Model: "text-embedding-v4", Dims: 2, Timeout: 5})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func (h *kbIngestHarness) create(ctx context.Context) (*aipb.CreateKbDocResponse, error) {
	h.background()
	out, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(&aipb.CreateKbDocRequest{Actor: h.actor, KbId: h.kb, Title: "相同标题", Content: "相同正文", RequestId: uuid.NewString()})
	if err != nil {
		return out, err
	}
	out.Doc, err = h.waitDoc(ctx, out.Doc.Id)
	return out, err
}

func (h *kbIngestHarness) background() {
	h.t.Helper()
	h.start.Do(func() { stop := h.s.StartKbIngestWorkers(context.Background()); h.t.Cleanup(stop) })
}

func (h *kbIngestHarness) waitDoc(ctx context.Context, id int64) (*aipb.KbDocInfo, error) {
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		row, err := h.s.AiKbDocModel.FindMetadataOwned(ctx, h.kb, id, h.actor.UserId)
		if err != nil {
			return nil, err
		}
		if row.Status != "pending" {
			return kbDocRowToPb(row), nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline.C:
			return nil, errors.New("background ingest did not complete")
		case <-ticker.C:
		}
	}
}
func (h *kbIngestHarness) pending() (int64, []uint64) {
	h.t.Helper()
	r, err := h.db.Exec("INSERT INTO ai_kb_doc(kb_id,title,content,chunk_count,embedding_model) VALUES (?,'待重试','正文',1,'text-embedding-v4')", h.kb)
	if err != nil {
		h.t.Fatal(err)
	}
	id, _ := r.LastInsertId()
	r, err = h.db.Exec("INSERT INTO ai_kb_chunk(doc_id,kb_id,seq,content,token_count) VALUES (?,?,0,'正文',2)", id, h.kb)
	if err != nil {
		h.t.Fatal(err)
	}
	chunk, _ := r.LastInsertId()
	return id, []uint64{uint64(chunk)}
}
func (h *kbIngestHarness) retry(id int64) (*aipb.RetryKbDocResponse, error) {
	h.background()
	out, err := NewRetryKbDocLogic(context.Background(), h.s).RetryKbDoc(&aipb.RetryKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, RequestId: uuid.NewString()})
	if err != nil {
		return out, err
	}
	out.Doc, err = h.waitDoc(context.Background(), id)
	return out, err
}
func (h *kbIngestHarness) state(id int64) string {
	h.t.Helper()
	var s string
	if err := h.db.QueryRow("SELECT status FROM ai_kb_doc WHERE id=?", id).Scan(&s); err != nil {
		h.t.Fatal(err)
	}
	return s
}

func TestKbIngestConcurrentCreateDeduplicates(t *testing.T) {
	h := newKbIngestHarness(t)
	const n = 6
	var wg sync.WaitGroup
	ids := make(chan int64, n)
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := h.create(context.Background())
			if e != nil {
				errs <- e
				return
			}
			ids <- r.Doc.Id
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	var id int64
	for got := range ids {
		if id != 0 && got != id {
			t.Errorf("duplicate create IDs %d and %d", id, got)
		}
		id = got
	}
	if got := h.calls.Load(); got != 1 {
		t.Errorf("embedding calls=%d, want 1", got)
	}
}

func TestKbIngestActivePendingRejectsRetry(t *testing.T) {
	h := newKbIngestHarness(t)
	id, ids := h.pending()
	if _, err := h.retry(id); err == nil {
		t.Error("active pending retry accepted")
	} else if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 400 {
		t.Errorf("pending retry error=%v, want code 400", err)
	}
	var count int
	if err := h.db.QueryRow("SELECT COUNT(*) FROM ai_kb_chunk WHERE id=?", ids[0]).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 || h.calls.Load() != 0 {
		t.Error("rejected retry replaced chunks or called embedding")
	}
}

func TestKbIngestCancellationWritesFailed(t *testing.T) {
	h := newKbIngestHarness(t)
	entered := make(chan struct{})
	release := make(chan struct{})
	h.embed = func(w http.ResponseWriter, r *http.Request) {
		close(entered)
		<-release
		http.Error(w, "cancelled", 500)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); _, _ = h.create(ctx) }()
	<-entered
	cancel()
	<-done
	close(release)
	var docId int64
	if err := h.db.QueryRow("SELECT id FROM ai_kb_doc WHERE kb_id=?", h.kb).Scan(&docId); err != nil {
		t.Fatal(err)
	}
	if _, err := h.waitDoc(context.Background(), docId); err != nil {
		t.Fatal(err)
	}
	var state string
	if err := h.db.QueryRow("SELECT status FROM ai_kb_doc WHERE kb_id=?", h.kb).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != "failed" {
		t.Errorf("cancelled ingest state=%s, want failed", state)
	}
}

func TestKbIngestDeletedDocumentCannotBecomeReady(t *testing.T) {
	h := newKbIngestHarness(t)
	id, ids := h.pending()
	if _, err := h.s.AiKbDocModel.DeleteOwned(context.Background(), h.kb, id, h.actor.UserId); err != nil {
		t.Fatal(err)
	}
	if err := h.s.AiKbDocModel.MarkIngested(context.Background(), id, ids, "text-embedding-v4"); err == nil {
		t.Error("deleted document accepted late ready")
	}
	if h.state(id) == "ready" {
		t.Error("late completion revived deleted document")
	}
}

func TestKbIngestExpiredPendingConcurrentRetry(t *testing.T) {
	h := newKbIngestHarness(t)
	id, old := h.pending()
	if _, err := h.db.Exec("UPDATE ai_kb_doc SET updated_at=DATE_SUB(NOW(),INTERVAL 121 SECOND) WHERE id=?", id); err != nil {
		t.Fatal(err)
	}
	const n = 6
	var wg sync.WaitGroup
	var success atomic.Int64
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if r, err := h.retry(id); err == nil {
				if r.Doc.Status != "ready" {
					t.Errorf("retry status=%s", r.Doc.Status)
				}
				success.Add(1)
			}
		}()
	}
	wg.Wait()
	if success.Load() != 1 || h.calls.Load() != 1 {
		t.Errorf("concurrent retry successes=%d embedding=%d, want 1/1", success.Load(), h.calls.Load())
	}
	var remaining int
	if err := h.db.QueryRow("SELECT COUNT(*) FROM ai_kb_chunk WHERE id=?", old[0]).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Error("expired attempt chunks remain")
	}
}

func TestKbIngestLateCompletionCannotOverwriteNewRetry(t *testing.T) {
	h := newKbIngestHarness(t)
	id, old := h.pending()
	entered, release := make(chan struct{}), make(chan struct{})
	var upserts atomic.Int64
	h.store.upsert = func(context.Context, []vectordb.Point) error {
		if upserts.Add(1) == 1 {
			close(entered)
			<-release
		}
		return nil
	}
	done := make(chan error, 1)
	go func() {
		done <- ingestChunks(context.Background(), h.s, id, h.kb, h.actor.UserId, []model.KbChunkRow{{Seq: 0, Content: "旧尝试", TokenCount: 2}}, old)
	}()
	<-entered
	if _, err := h.db.Exec("UPDATE ai_kb_doc SET updated_at=DATE_SUB(NOW(),INTERVAL 121 SECOND) WHERE id=?", id); err != nil {
		t.Fatal(err)
	}
	response, err := h.retry(id)
	close(release)
	if err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, model.ErrKbDocState) {
		t.Errorf("late attempt result=%v, want state conflict", err)
	}
	if response.Doc.Status != "ready" || h.state(id) != "ready" {
		t.Error("late attempt overwrote newer ready")
	}
	if err := h.s.AiKbDocModel.MarkFailed(context.Background(), id, old, "旧尝试失败"); !errors.Is(err, model.ErrKbDocState) {
		t.Errorf("old failure result=%v", err)
	}
	points, _ := h.store.AllIDs(context.Background())
	if _, exists := points[old[0]]; exists {
		t.Error("late attempt left orphan vector")
	}
	if len(points) != 1 {
		t.Errorf("new attempt vectors=%d, want 1", len(points))
	}
}

func TestKbIngestRetryReplacementRollsBack(t *testing.T) {
	h := newKbIngestHarness(t)
	id, old := h.pending()
	if _, err := h.db.Exec("UPDATE ai_kb_doc SET status='failed',error='原始失败' WHERE id=?", id); err != nil {
		t.Fatal(err)
	}
	_, _, err := h.s.AiKbDocModel.ReplaceChunksForRetry(context.Background(), h.kb, id, h.actor.UserId, []model.KbChunkRow{{Seq: -1, Content: "不能写入", TokenCount: 1}})
	if err == nil {
		t.Fatal("invalid chunk transaction unexpectedly committed")
	}
	var chunks int
	if err := h.db.QueryRow("SELECT COUNT(*) FROM ai_kb_chunk WHERE id=? AND doc_id=?", old[0], id).Scan(&chunks); err != nil {
		t.Fatal(err)
	}
	if chunks != 1 || h.state(id) != "failed" {
		t.Error("failed replacement lost old chunks/state")
	}
	var message string
	if err := h.db.QueryRow("SELECT error FROM ai_kb_doc WHERE id=?", id).Scan(&message); err != nil {
		t.Fatal(err)
	}
	if message != "原始失败" {
		t.Error("rollback changed original error")
	}
}

func TestKbIngestCrossOwnerDenied(t *testing.T) {
	h := newKbIngestHarness(t)
	id, old := h.pending()
	stranger := &aipb.Actor{UserId: h.actor.UserId + 1}
	if _, err := NewCreateKbDocLogic(context.Background(), h.s).CreateKbDoc(&aipb.CreateKbDocRequest{Actor: stranger, KbId: h.kb, Title: "标题", Content: "正文", RequestId: uuid.NewString()}); err == nil {
		t.Error("cross owner create allowed")
	} else if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 404 {
		t.Errorf("cross owner create error=%v, want code 404", err)
	}
	if _, err := NewRetryKbDocLogic(context.Background(), h.s).RetryKbDoc(&aipb.RetryKbDocRequest{Actor: stranger, KbId: h.kb, DocId: id, RequestId: uuid.NewString()}); err == nil {
		t.Error("cross owner retry allowed")
	} else if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 404 {
		t.Errorf("cross owner retry error=%v, want code 404", err)
	}
	if _, _, _, err := h.s.AiKbDocModel.InsertWithChunks(context.Background(), stranger.UserId, &model.KbDocRow{KbId: h.kb, Title: "直接调用", Content: "正文"}, nil); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("model ownership check=%v", err)
	}
	if _, _, err := h.s.AiKbDocModel.ReplaceChunksForRetry(context.Background(), h.kb, id, stranger.UserId, nil); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("model retry ownership check=%v", err)
	}
	if h.calls.Load() != 0 {
		t.Error("denied request called embedding")
	}
	if err := h.s.AiKbDocModel.MarkIngested(context.Background(), id, old, "text-embedding-v4"); err != nil {
		t.Fatal(err)
	}
}

func TestKbIngestCreateExactBytesOnly(t *testing.T) {
	h := newKbIngestHarness(t)
	var ids []int64
	for _, title := range []string{"Café", "café", "Cafe", "Café "} {
		id, _, created, err := h.s.AiKbDocModel.InsertWithChunks(context.Background(), h.actor.UserId, &model.KbDocRow{KbId: h.kb, Title: title, Content: "正文", EmbeddingModel: "text-embedding-v4"}, []model.KbChunkRow{{Content: "正文"}})
		if err != nil || !created {
			t.Fatalf("distinct bytes incorrectly deduplicated: created=%v err=%v", created, err)
		}
		ids = append(ids, id)
	}
	id, _, created, err := h.s.AiKbDocModel.InsertWithChunks(context.Background(), h.actor.UserId, &model.KbDocRow{KbId: h.kb, Title: "Café", Content: "正文", EmbeddingModel: "text-embedding-v4"}, nil)
	if err != nil || created || id != ids[0] {
		t.Errorf("identical create not reused: id=%d created=%v err=%v", id, created, err)
	}
	for _, doc := range []model.KbDocRow{{KbId: h.kb, Title: "Café", Content: "正文 ", EmbeddingModel: "text-embedding-v4"}, {KbId: h.kb, Title: "Café", Content: "正文", EmbeddingModel: "TEXT-embedding-v4"}} {
		if _, _, created, err := h.s.AiKbDocModel.InsertWithChunks(context.Background(), h.actor.UserId, &doc, []model.KbChunkRow{{Content: doc.Content}}); err != nil || !created {
			t.Errorf("content/model bytes incorrectly deduplicated: created=%v err=%v", created, err)
		}
	}
}

func TestKbIngestDeletionDuringUpsertCompensatesOnlyAttempt(t *testing.T) {
	for _, deleteParent := range []bool{false, true} {
		t.Run(fmt.Sprint(deleteParent), func(t *testing.T) {
			h := newKbIngestHarness(t)
			id, ids := h.pending()
			entered, release := make(chan struct{}), make(chan struct{})
			h.store.upsert = func(context.Context, []vectordb.Point) error { close(entered); <-release; return nil }
			done := make(chan error, 1)
			go func() {
				done <- ingestChunks(context.Background(), h.s, id, h.kb, h.actor.UserId, []model.KbChunkRow{{Content: "正文"}}, ids)
			}()
			<-entered
			var err error
			if deleteParent {
				_, err = h.s.AiKbModel.DeleteOwned(context.Background(), h.kb, h.actor.UserId)
			} else {
				_, err = h.s.AiKbDocModel.DeleteOwned(context.Background(), h.kb, id, h.actor.UserId)
			}
			close(release)
			if err != nil {
				t.Fatal(err)
			}
			if err := <-done; err == nil {
				t.Error("deleted resource accepted ingest completion")
			}
			if h.state(id) == "ready" {
				t.Error("deleted resource became ready")
			}
			points, _ := h.store.AllIDs(context.Background())
			if len(points) != 0 {
				t.Error("deleted attempt left vectors")
			}
			if _, _, _, err := h.s.AiKbDocModel.InsertWithChunks(context.Background(), h.actor.UserId, &model.KbDocRow{KbId: h.kb, Title: "再创建", Content: "正文"}, nil); deleteParent && !errors.Is(err, model.ErrNotFound) {
				t.Errorf("deleted parent accepted create: %v", err)
			}
		})
	}
}

// 缓存失效故障发生在真实 MySQL ready 提交之后，验证其补偿边界。
type cacheFailureDocModel struct{ model.AiKbDocModel }

func (m *cacheFailureDocModel) MarkIngested(ctx context.Context, id int64, ids []uint64, embedding string) error {
	if err := m.AiKbDocModel.MarkIngested(ctx, id, ids, embedding); err != nil {
		return err
	}
	return &model.KbDocCommittedError{Err: errors.New("isolated cache failure")}
}

func (m *cacheFailureDocModel) ReplaceChunksForRetry(ctx context.Context, kb, id, owner int64, chunks []model.KbChunkRow, expectedVersion ...int64) ([]uint64, []uint64, error) {
	old, next, err := m.AiKbDocModel.ReplaceChunksForRetry(ctx, kb, id, owner, chunks, expectedVersion...)
	if err != nil {
		return old, next, err
	}
	return old, next, &model.KbDocCommittedError{Err: errors.New("isolated cache failure")}
}

func (m *cacheFailureDocModel) DeleteOwned(ctx context.Context, kb, id, owner int64) ([]uint64, error) {
	ids, err := m.AiKbDocModel.DeleteOwned(ctx, kb, id, owner)
	if err != nil {
		return ids, err
	}
	return ids, &model.KbDocCommittedError{Err: errors.New("isolated cache failure")}
}

type cacheFailureKbModel struct{ model.AiKbModel }

func (m *cacheFailureKbModel) DeleteOwned(ctx context.Context, kb, owner int64) ([]uint64, error) {
	ids, err := m.AiKbModel.DeleteOwned(ctx, kb, owner)
	if err != nil {
		return ids, err
	}
	return ids, &model.KbDocCommittedError{Err: errors.New("isolated cache failure")}
}

func TestKbIngestRetryCommittedCacheFailureStillIngests(t *testing.T) {
	h := newKbIngestHarness(t)
	id, old := h.pending()
	if _, err := h.db.Exec("UPDATE ai_kb_doc SET status='failed' WHERE id=?", id); err != nil {
		t.Fatal(err)
	}
	h.store.points[old[0]] = vectordb.Point{ID: old[0]}
	h.s.AiKbDocModel = &cacheFailureDocModel{h.s.AiKbDocModel}
	r, err := h.retry(id)
	if err != nil {
		t.Fatal(err)
	}
	if r.Doc.Status != "ready" {
		t.Errorf("committed retry abandoned: status=%s", r.Doc.Status)
	}
	points, _ := h.store.AllIDs(context.Background())
	if _, ok := points[old[0]]; ok {
		t.Error("committed retry left old vector")
	}
	if len(points) != 1 {
		t.Error("committed retry lost new vector")
	}
}

func TestKbIngestDeleteCommittedCacheFailureStillDeletesVectors(t *testing.T) {
	for _, parent := range []bool{false, true} {
		t.Run(fmt.Sprint(parent), func(t *testing.T) {
			h := newKbIngestHarness(t)
			id, ids := h.pending()
			h.store.points[ids[0]] = vectordb.Point{ID: ids[0]}
			if parent {
				h.s.AiKbModel = &cacheFailureKbModel{h.s.AiKbModel}
				_, _ = NewDeleteKbLogic(context.Background(), h.s).DeleteKb(&aipb.DeleteKbRequest{Actor: h.actor, KbId: h.kb})
			} else {
				h.s.AiKbDocModel = &cacheFailureDocModel{h.s.AiKbDocModel}
				_, _ = NewDeleteKbDocLogic(context.Background(), h.s).DeleteKbDoc(&aipb.DeleteKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id})
			}
			points, _ := h.store.AllIDs(context.Background())
			if len(points) != 0 {
				t.Error("committed deletion cache failure skipped precise vector cleanup")
			}
		})
	}
}

func TestKbIngestCommittedReadyCacheFailureKeepsVectors(t *testing.T) {
	h := newKbIngestHarness(t)
	id, ids := h.pending()
	h.s.AiKbDocModel = &cacheFailureDocModel{h.s.AiKbDocModel}
	err := ingestChunks(context.Background(), h.s, id, h.kb, h.actor.UserId, []model.KbChunkRow{{Content: "正文"}}, ids)
	if !model.KbDocCommitted(err) {
		t.Errorf("missing committed result: %v", err)
	}
	if h.state(id) != "ready" {
		t.Error("committed ready was changed")
	}
	points, _ := h.store.AllIDs(context.Background())
	if _, ok := points[ids[0]]; !ok {
		t.Error("cache failure deleted committed ready vector")
	}
}

func TestKbIngestPartialVectorFailureRemovesAttempt(t *testing.T) {
	h := newKbIngestHarness(t)
	id, ids := h.pending()
	h.store.upsert = func(_ context.Context, points []vectordb.Point) error {
		h.store.mu.Lock()
		h.store.points[points[0].ID] = points[0]
		h.store.mu.Unlock()
		return errors.New("isolated partial vector failure")
	}
	if err := ingestChunks(context.Background(), h.s, id, h.kb, h.actor.UserId, []model.KbChunkRow{{Content: "正文"}}, ids); err == nil {
		t.Fatal("partial upsert failure was hidden")
	}
	if h.state(id) != "failed" {
		t.Error("partial vector failure left pending")
	}
	points, _ := h.store.AllIDs(context.Background())
	if len(points) != 0 {
		t.Error("partial upsert left attempt vector")
	}
}

func TestKbIngestDeadlineAfterUpsertWritesFailed(t *testing.T) {
	h := newKbIngestHarness(t)
	id, ids := h.pending()
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	h.store.upsert = func(ctx context.Context, _ []vectordb.Point) error { <-ctx.Done(); return nil }
	err := ingestChunks(ctx, h.s, id, h.kb, h.actor.UserId, []model.KbChunkRow{{Content: "正文"}}, ids)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("deadline result=%v", err)
	}
	if h.state(id) != "failed" {
		t.Error("deadline did not write failed independently")
	}
	points, _ := h.store.AllIDs(context.Background())
	if len(points) != 0 {
		t.Error("deadline left vectors")
	}
}

func TestKbIngestDuplicateFailedCreateNeedsExplicitRetry(t *testing.T) {
	h := newKbIngestHarness(t)
	h.embed = func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "isolated embedding failure", 500) }
	first, err := h.create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	second, err := h.create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if first.Doc.Status != "failed" || second.Doc.Status != "failed" || first.Doc.Id != second.Doc.Id || h.calls.Load() != 1 {
		t.Error("duplicate failed create repeated ingestion or changed document")
	}
}

type ingestVectorStore struct {
	mu     sync.Mutex
	points map[uint64]vectordb.Point
	upsert func(context.Context, []vectordb.Point) error
}

func (s *ingestVectorStore) EnsureCollection(context.Context) error { return nil }
func (s *ingestVectorStore) Upsert(ctx context.Context, p []vectordb.Point) error {
	if s.upsert != nil {
		if err := s.upsert(ctx, p); err != nil {
			return err
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range p {
		s.points[v.ID] = v
	}
	return nil
}
func (s *ingestVectorStore) Search(context.Context, []float32, int64, int64, int) ([]vectordb.Hit, error) {
	return nil, errors.New("not used")
}
func (s *ingestVectorStore) Delete(ctx context.Context, ids []uint64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, id := range ids {
		delete(s.points, id)
	}
	return nil
}
func (s *ingestVectorStore) DeleteByKb(context.Context, int64) error { return nil }
func (s *ingestVectorStore) AllIDs(context.Context) (map[uint64]struct{}, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ids := map[uint64]struct{}{}
	for id := range s.points {
		ids[id] = struct{}{}
	}
	return ids, nil
}
func (s *ingestVectorStore) Close() error { return nil }
