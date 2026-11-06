package vectordb

import (
	"context"
	"os"
	"testing"
	"time"
)

// Qdrant 集成测试：仅在显式设置 GOZERO_QDRANT_TEST_ADDR（如 127.0.0.1:16334）时执行，
// 指向隔离开发实例；缺失时明确跳过，不记为通过。
func qdrantTestStore(t *testing.T) *QdrantStore {
	t.Helper()
	addr := os.Getenv("GOZERO_QDRANT_TEST_ADDR")
	if addr == "" {
		t.Skip("缺少 GOZERO_QDRANT_TEST_ADDR，跳过 Qdrant 集成用例")
	}
	const dims = 4
	store, err := NewQdrantStore(addr, "", CollectionName("test", "it", dims), dims)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = store.client.DeleteCollection(ctx, store.collection)
		_ = store.Close()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := store.EnsureCollection(ctx); err != nil {
		t.Fatalf("建集合失败: %v", err)
	}
	// 服务重启路径：集合与字段索引已存在时必须幂等成功
	if err := store.EnsureCollection(ctx); err != nil {
		t.Fatalf("EnsureCollection 应幂等（重启场景）: %v", err)
	}
	return store
}

func TestQdrantRoundTrip(t *testing.T) {
	store := qdrantTestStore(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	points := []Point{
		{ID: 1, Vector: []float32{1, 0, 0, 0}, KbID: 100, OwnerID: 7, DocID: 1, TokenCount: 10},
		{ID: 2, Vector: []float32{0.9, 0.1, 0, 0}, KbID: 100, OwnerID: 7, DocID: 1, TokenCount: 10},
		{ID: 3, Vector: []float32{1, 0, 0, 0}, KbID: 200, OwnerID: 8, DocID: 2, TokenCount: 10}, // 他人同向量
	}
	if err := store.Upsert(ctx, points); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// 归属过滤：owner 7 不能看到 owner 8 的点，即使向量完全相同
	hits, err := store.Search(ctx, []float32{1, 0, 0, 0}, 7, 0, 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("owner 过滤失效: 命中 %d 个", len(hits))
	}
	for _, h := range hits {
		if h.ID == 3 {
			t.Fatal("越权命中他人知识库")
		}
	}

	// kb 过滤
	hits, err = store.Search(ctx, []float32{1, 0, 0, 0}, 7, 999, 10)
	if err != nil {
		t.Fatalf("search kb: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("kb 过滤失效: 命中 %d 个", len(hits))
	}

	// 幂等删除：重复删除不存在 ID 不报错
	if err := store.Delete(ctx, []uint64{2}); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := store.Delete(ctx, []uint64{2, 99999}); err != nil {
		t.Fatalf("重复/不存在删除应幂等: %v", err)
	}
	hits, err = store.Search(ctx, []float32{1, 0, 0, 0}, 7, 0, 10)
	if err != nil {
		t.Fatalf("search after delete: %v", err)
	}
	if len(hits) != 1 || hits[0].ID != 1 {
		t.Fatalf("删除后命中异常: %+v", hits)
	}

	// 对账清单
	all, err := store.AllIDs(ctx)
	if err != nil {
		t.Fatalf("AllIDs: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("对账清单应为 2 个: %d", len(all))
	}

	// DeleteByKb
	if err := store.DeleteByKb(ctx, 100); err != nil {
		t.Fatalf("DeleteByKb: %v", err)
	}
	all, err = store.AllIDs(ctx)
	if err != nil {
		t.Fatalf("AllIDs 2: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("DeleteByKb 后应剩 1 个: %d", len(all))
	}
}
