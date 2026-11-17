package svc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	qdrantapi "github.com/qdrant/go-client/qdrant"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/vectordb"
)

type integrationQueryVectorRedis struct {
	*redis.Redis
	mu   sync.Mutex
	keys map[string]struct{}
}

func (r *integrationQueryVectorRedis) SetexCtx(ctx context.Context, key, value string, seconds int) error {
	r.mu.Lock()
	r.keys[key] = struct{}{}
	r.mu.Unlock()
	return r.Redis.SetexCtx(ctx, key, value, seconds)
}

type integrationQueryVectorStore struct {
	vectordb.Store
	searches int
}

func (s *integrationQueryVectorStore) Search(ctx context.Context, v []float32, owner, kb int64, limit int) ([]vectordb.Hit, error) {
	s.searches++
	return s.Store.Search(ctx, v, owner, kb, limit)
}

// 组合回归仅使用显式隔离 Redis/MySQL/Qdrant；Embedding 固定为本地 httptest。
func TestKbQueryVectorCacheRealDependenciesRecheckFacts(t *testing.T) {
	dsn, address, qdrant := os.Getenv("GOZERO_AI_TEST_DSN"), os.Getenv("GOZERO_AI_TEST_REDIS"), os.Getenv("GOZERO_QDRANT_TEST_ADDR")
	if dsn == "" || address == "" || qdrant == "" {
		t.Skip("requires isolated GOZERO_AI_TEST_DSN / GOZERO_AI_TEST_REDIS / GOZERO_QDRANT_TEST_ADDR")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sqlx.DisableLog()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	conf := cache.CacheConf{{RedisConf: redis.RedisConf{Host: address, Type: "node"}, Weight: 100}}
	rdb, err := redis.NewRedis(conf[0].RedisConf)
	if err != nil {
		t.Fatal(err)
	}
	rcache := &integrationQueryVectorRedis{Redis: rdb, keys: map[string]struct{}{}}
	conn := sqlx.NewSqlConnFromDB(db)
	kbs := model.NewAiKbModel(conn, conf)
	owner := time.Now().UnixNano()
	insert, err := db.ExecContext(ctx, "INSERT INTO ai_kb(owner_id,name) VALUES (?,?)", owner, "query-vector-cache-integration")
	if err != nil {
		t.Fatal(err)
	}
	kb, _ := insert.LastInsertId()
	var doc, chunk int64
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		for _, query := range []string{"DELETE FROM ai_kb_ingest_job WHERE kb_id=?", "DELETE FROM ai_kb_ingest_batch WHERE kb_id=?", "DELETE FROM ai_kb_chunk WHERE kb_id=?", "DELETE FROM ai_kb_doc WHERE kb_id=?", "DELETE FROM ai_kb WHERE id=?"} {
			if _, err := db.ExecContext(cleanup, query, kb); err != nil {
				t.Error(err)
			}
		}
		keys := []string{fmt.Sprintf("cache:gozeroAdmin:aiKb:id:%d", kb), fmt.Sprintf("cache:gozeroAdmin:aiKbDoc:id:%d", doc)}
		for key := range rcache.keys {
			keys = append(keys, key)
		}
		if _, err := rdb.DelCtx(cleanup, keys...); err != nil {
			t.Error(err)
		}
		for _, key := range keys {
			if exists, err := rdb.ExistsCtx(cleanup, key); err != nil || exists {
				t.Error("precise cache cleanup failed")
			}
		}
		var count int
		if err := db.QueryRowContext(cleanup, "SELECT (SELECT COUNT(*) FROM ai_kb WHERE id=?)+(SELECT COUNT(*) FROM ai_kb_doc WHERE kb_id=?)+(SELECT COUNT(*) FROM ai_kb_chunk WHERE kb_id=?)", kb, kb, kb).Scan(&count); err != nil || count != 0 {
			t.Error("precise SQL cleanup failed")
		}
		t.Logf("verified cleanup kb_id=%d doc_id=%d chunk_id=%d exact_cache_keys=%d", kb, doc, chunk, len(keys))
	})
	insert, err = db.ExecContext(ctx, "INSERT INTO ai_kb_doc(kb_id,title,content,status,ingest_stage,chunk_count,embedding_model) VALUES (?,'isolated fixture','权限审批条件必须由管理员核对。','ready','ready',1,'fixture')", kb)
	if err != nil {
		t.Fatal(err)
	}
	doc, _ = insert.LastInsertId()
	insert, err = db.ExecContext(ctx, "INSERT INTO ai_kb_chunk(doc_id,kb_id,seq,content,token_count,status) VALUES (?,?,0,'权限审批条件必须由管理员核对。',12,'ready')", doc, kb)
	if err != nil {
		t.Fatal(err)
	}
	chunk, _ = insert.LastInsertId()
	collection := vectordb.CollectionName("query_cache_integration", fmt.Sprint(owner), 2)
	actual, err := vectordb.NewQdrantStore(qdrant, "", collection, 2)
	if err != nil {
		t.Fatal(err)
	}
	defer actual.Close()
	host, portText, err := net.SplitHostPort(qdrant)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}
	cleanupClient, err := qdrantapi.NewClient(&qdrantapi.Config{Host: host, Port: port})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		defer cleanupClient.Close()
		cleanup, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		exists, err := cleanupClient.CollectionExists(cleanup, collection)
		if err != nil {
			t.Error(err)
			return
		}
		if exists {
			if err := cleanupClient.DeleteCollection(cleanup, collection); err != nil {
				t.Error(err)
				return
			}
		}
		if exists, err := cleanupClient.CollectionExists(cleanup, collection); err != nil || exists {
			t.Error("本轮精确测试集合清理未完成")
		}
	})
	if err := actual.EnsureCollection(ctx); err != nil {
		t.Fatal(err)
	}
	if err := actual.Upsert(ctx, []vectordb.Point{{ID: uint64(chunk), Vector: []float32{1, 0}, KbID: kb, OwnerID: owner, DocID: doc, TokenCount: 12}}); err != nil {
		t.Fatal(err)
	}
	var embeds atomic.Int32
	embedder, ai := queryVectorFixture(t, func(w http.ResponseWriter, _ *http.Request) { embeds.Add(1); queryVectorReply(w) }, nil)
	qstore := &integrationQueryVectorStore{Store: actual}
	searcher := &kbSearcher{store: qstore, embedder: embedder, chunks: model.NewAiKbChunkModel(conn, conf), snippetBudget: 1000, queryVectors: newKbQueryVectorCache(rcache, embedder, ai)}
	query := "权限审批条件"
	for range 2 {
		hits, err := searcher.Search(ctx, owner, kb, query)
		if err != nil || len(hits) != 1 || hits[0].ChunkID != chunk || hits[0].Snippet != "权限审批条件必须由管理员核对。" {
			t.Fatalf("real search hits=%+v err=%v", hits, err)
		}
	}
	if embeds.Load() != 1 || qstore.searches != 2 || len(rcache.keys) != 1 {
		t.Fatalf("cache reuse embeds=%d vector_searches=%d keys=%d", embeds.Load(), qstore.searches, len(rcache.keys))
	}
	for key := range rcache.keys {
		raw, err := rdb.GetCtx(ctx, key)
		if err != nil || strings.Contains(key, query) || strings.Contains(raw, query) {
			t.Fatal("cache must only contain hashed identity and vector")
		}
		var vector []float32
		if json.Unmarshal([]byte(raw), &vector) != nil || len(vector) != 2 || vector[0] != 1 {
			t.Fatal("real Redis vector invalid")
		}
		ttl, err := rdb.TtlCtx(ctx, key)
		if err != nil || ttl < 290 || ttl > 300 {
			t.Fatalf("actual TTL=%d err=%v", ttl, err)
		}
		t.Logf("real Redis query cache TTL=%d seconds", ttl)
	}
	searcher.queryVectors = newKbQueryVectorCache(rcache, embedder, ai)
	if hits, err := searcher.Search(ctx, owner, kb, query); err != nil || len(hits) != 1 || embeds.Load() != 1 {
		t.Fatal("new service instance must reuse persisted vector", err)
	}
	if err := kbs.RequireOwned(ctx, kb, owner); err != nil {
		t.Fatal(err)
	}
	if err := kbs.RequireOwned(ctx, kb, owner+1); !errors.Is(err, model.ErrNotFound) {
		t.Fatal("cross owner authorization", err)
	}
	assertEmpty := func(label string) {
		t.Helper()
		if hits, err := searcher.Search(ctx, owner, kb, query); err != nil || len(hits) != 0 || embeds.Load() != 1 {
			t.Fatalf("%s must recheck SQL with cached vector: hits=%+v err=%v embeds=%d", label, hits, err, embeds.Load())
		}
	}
	for _, step := range []struct {
		label, statement, restore string
		id                        int64
	}{
		{"document failed", "UPDATE ai_kb_doc SET status='failed' WHERE id=?", "UPDATE ai_kb_doc SET status='ready' WHERE id=?", doc},
		{"chunk failed", "UPDATE ai_kb_chunk SET status='failed' WHERE id=?", "UPDATE ai_kb_chunk SET status='ready' WHERE id=?", chunk},
	} {
		if _, err := db.ExecContext(ctx, step.statement, step.id); err != nil {
			t.Fatal(err)
		}
		assertEmpty(step.label)
		if _, err := db.ExecContext(ctx, step.restore, step.id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, "UPDATE ai_kb SET owner_id=? WHERE id=?", owner+1, kb); err != nil {
		t.Fatal(err)
	}
	assertEmpty("changed KB owner with stale Qdrant payload")
	if err := kbs.RequireOwned(ctx, kb, owner); !errors.Is(err, model.ErrNotFound) {
		t.Fatal("ownership must not be cached", err)
	}
	if err := kbs.RequireOwned(ctx, kb, owner+1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE ai_kb SET owner_id=? WHERE id=?", owner, kb); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE ai_kb_doc SET deleted_at=NOW() WHERE id=?", doc); err != nil {
		t.Fatal(err)
	}
	assertEmpty("soft-deleted document with stale Qdrant payload")
	if _, err := kbs.FindOne(ctx, uint64(kb)); err != nil {
		t.Fatal(err)
	}
	if _, err := kbs.DeleteOwned(ctx, kb, owner); err != nil {
		t.Fatal(err)
	}
	assertEmpty("soft-deleted KB with stale Qdrant payload")
	if err := kbs.RequireOwned(ctx, kb, owner); !errors.Is(err, model.ErrNotFound) {
		t.Fatal("soft-deleted KB authorization", err)
	}
	if row, err := kbs.FindOne(ctx, uint64(kb)); err != nil || !row.DeletedAt.Valid {
		t.Fatal("KB generated cache must be invalidated after DeleteOwned", err)
	}
	if qstore.searches != 8 || embeds.Load() != 1 {
		t.Fatalf("all lookups must requery vector/SQL: searches=%d embeds=%d", qstore.searches, embeds.Load())
	}
	t.Logf("actual Redis/MySQL/Qdrant: 8 searches, 1 local embedding call; new service reuse and status/owner/document/KB deletion isolated")
}
