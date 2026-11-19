package aiservicelogic

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/vectordb"
)

// 同一篇 ready 资料只改标题，统计实际调用桩的增量，便于与变更前使用相同 fixture 复测。
func TestKbDocumentsTitleOnlyCost(t *testing.T) {
	h := newKbIngestHarness(t)
	var upserts atomic.Int64
	h.store.upsert = func(context.Context, []vectordb.Point) error { upserts.Add(1); return nil }
	ctx := context.Background()
	initial, err := h.create(ctx)
	if err != nil {
		t.Fatal(err)
	}
	embeddingBefore, upsertsBefore := h.calls.Load(), upserts.Load()
	old, err := h.store.AllIDs(ctx)
	if err != nil || len(old) != 1 {
		t.Fatalf("初始向量=%v err=%v", old, err)
	}
	updated, err := NewUpdateKbDocLogic(ctx, h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{
		Actor: h.actor, KbId: h.kb, DocId: initial.Doc.Id, ExpectedVersion: initial.Doc.Version,
		Title: "仅修改展示标题", Content: "相同正文", RequestId: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	ready, err := h.waitDoc(ctx, initial.Doc.Id)
	if err != nil || ready.Status != "ready" || ready.Version != 2 {
		t.Fatalf("改名终态=%v err=%v", ready, err)
	}
	points, err := h.store.AllIDs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	retained := 0
	for id := range old {
		if _, ok := points[id]; ok {
			retained++
		}
	}
	embedding, upsert := h.calls.Load()-embeddingBefore, upserts.Load()-upsertsBefore
	t.Logf("改名回执=%s Embedding=%d QdrantUpsert=%d 原分块保留=%d/%d", updated.Doc.Stage, embedding, upsert, retained, len(old))
	if updated.Doc.Stage != "ready" || embedding != 0 || upsert != 0 || retained != len(old) {
		t.Error("纯改名应保持可检索并保留原索引")
	}
}
