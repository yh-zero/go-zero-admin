package svc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/vectordb"
)

type waitingIngestStore struct {
	vectordb.Store
	ready   atomic.Bool
	probed  chan struct{}
	upserts atomic.Int32
}

func (s *waitingIngestStore) Ready(context.Context) error {
	select {
	case s.probed <- struct{}{}:
	default:
	}
	if !s.ready.Load() {
		return errors.New("index not ready")
	}
	return nil
}
func (s *waitingIngestStore) Upsert(context.Context, []vectordb.Point) error {
	s.upserts.Add(1)
	return nil
}
func (s *waitingIngestStore) Delete(context.Context, []uint64) error { return nil }

func TestKbWorkerWaitsForIndexBeforeClaimAndEmbedding(t *testing.T) {
	dsn, address := os.Getenv("GOZERO_AI_TEST_DSN"), os.Getenv("GOZERO_AI_TEST_REDIS")
	if dsn == "" || address == "" {
		t.Skip("需要显式隔离MySQL/Redis，未执行真实队列回归")
	}
	sqlx.DisableLog()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	owner := time.Now().UnixNano()
	inserted, err := db.ExecContext(ctx, "INSERT INTO ai_kb(owner_id,name) VALUES (?,?)", owner, "就绪等待回归-"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	kb, err := inserted.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("本轮就绪等待夹具kb_id=%d owner_id=%d，按精确ID清理", kb, owner)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, query := range []string{"DELETE FROM ai_kb_ingest_job WHERE kb_id=?", "DELETE FROM ai_kb_ingest_batch WHERE kb_id=?", "DELETE FROM ai_kb_chunk WHERE kb_id=?", "DELETE FROM ai_kb_doc WHERE kb_id=?", "DELETE FROM ai_kb WHERE id=?"} {
			if _, err := db.ExecContext(cleanup, query, kb); err != nil {
				t.Error(err)
			}
		}
	})
	cacheConfig := cache.CacheConf{{RedisConf: redis.RedisConf{Host: address, Type: "node"}, Weight: 100}}
	conn := sqlx.NewMysql(dsn)
	docs := model.NewAiKbDocModel(conn, cacheConfig)
	queue := model.NewKbIngestQueue(conn, docs)
	request := model.KbEnqueueRequest{KbId: kb, OwnerId: owner, Operation: "create", Title: "就绪等待回归", Content: "fixture", RequestId: uuid.NewString(), BatchId: uuid.NewString(), EmbeddingModel: "fixture"}
	job, err := queue.Enqueue(ctx, request, []model.KbChunkRow{{Content: "fixture", TokenCount: 3}})
	if err != nil {
		t.Fatal(err)
	}
	var embeds atomic.Int32
	embedded := make(chan struct{}, 4)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		embeds.Add(1)
		embedded <- struct{}{}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"data":[{"index":0,"embedding":[1,0,0,0]}]}`)
	}))
	defer server.Close()
	embedder, err := llm.NewEmbedder(llm.EmbedConfig{BaseURL: server.URL, APIKey: "local-test-only", Model: "fixture", Dims: 4})
	if err != nil {
		t.Fatal(err)
	}
	store := &waitingIngestStore{probed: make(chan struct{}, 4)}
	service := &ServiceContext{Embedder: embedder, VectorStore: store, KbIngestQueue: queue}
	service.Config.AI.EmbeddingModel = "fixture"
	stop := service.StartKbIngestWorkers(ctx)
	defer stop()
	select {
	case <-store.probed:
	case <-embedded:
		t.Fatal("索引未就绪时已产生Embedding")
	case <-ctx.Done():
		t.Fatal("未观察到依赖探测")
	}
	var status string
	if err := db.QueryRowContext(ctx, "SELECT status FROM ai_kb_ingest_job WHERE id=? AND owner_id=?", job.Job.Id, owner).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "queued" || embeds.Load() != 0 {
		t.Fatalf("未就绪必须保留queued且不调用Embedding: %s/%d", status, embeds.Load())
	}
	store.ready.Store(true)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for status != "succeeded" {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("依赖恢复后作业未完成")
		}
		if err := db.QueryRowContext(ctx, "SELECT status FROM ai_kb_ingest_job WHERE id=? AND owner_id=?", job.Job.Id, owner).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if status == "failed" {
			t.Fatal("依赖恢复不应将排队作业置失败")
		}
	}
	if embeds.Load() != 1 || store.upserts.Load() != 1 {
		t.Fatalf("恢复后只能执行一次: embeds=%d upserts=%d", embeds.Load(), store.upserts.Load())
	}
}
