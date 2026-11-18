package svc

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"go-zero-admin/pkg/vectordb"
)

// 使用同一隔离库和向量集合，比较两份制度。额外伪造一条错绑载荷，验证第二层 SQL 范围复核。
func verifyDocumentRecall(t *testing.T, ctx context.Context, db *sql.DB, store vectordb.Store, searcher *kbSearcher, owner, kb int64) {
	t.Helper()
	insert := func(statement string, args ...any) int64 {
		t.Helper()
		r, err := db.ExecContext(ctx, statement, args...)
		if err != nil {
			t.Fatal(err)
		}
		id, err := r.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	const addDoc = "INSERT INTO ai_kb_doc(kb_id,title,content,status,version) VALUES (?,?,?,'ready',?)"
	const addChunk = "INSERT INTO ai_kb_chunk(kb_id,doc_id,seq,content,token_count,status) VALUES (?,?,?,?,12,'ready')"
	const standard = "出差住宿每晚最高 300 元。"
	const executive = "出差住宿每晚最高 800 元。"
	target := insert(addDoc, kb, "总部住宿制度", standard, 7)
	other := insert(addDoc, kb, "外派住宿制度", executive, 2)
	targetChunk := insert(addChunk, kb, target, 0, standard)
	points := []vectordb.Point{{ID: uint64(targetChunk), Vector: []float32{0, 1}, OwnerID: owner, KbID: kb, DocID: target}}
	for i := 0; i < kbSearchRecallTopK+2; i++ {
		id := insert(addChunk, kb, other, i, executive)
		payloadDoc := other
		if i == 0 {
			payloadDoc = target
		}
		points = append(points, vectordb.Point{ID: uint64(id), Vector: []float32{1, 0}, OwnerID: owner, KbID: kb, DocID: payloadDoc})
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ids := make([]uint64, 0, len(points))
		for _, p := range points {
			ids = append(ids, p.ID)
		}
		if err := store.Delete(cleanup, ids); err != nil {
			t.Error(err)
		}
		// 两篇合成资料在父 fixture 的同一专用 kb 下，SQL 清理及计数断言由父 fixture 完成。
	})
	if err := store.Upsert(ctx, points); err != nil {
		t.Fatal(err)
	}
	const query = "出差住宿每晚上限是多少？"
	hits, err := searcher.SearchDocument(ctx, owner, kb, target, query)
	if err != nil || len(hits) != 1 || hits[0].DocID != target || hits[0].ChunkID != targetChunk || hits[0].DocVersion != 7 || hits[0].Snippet != standard {
		t.Fatalf("全库前20候选均为其他制度时，仍须命中所选300元条款且排除错绑载荷：%v %+v", err, hits)
	}
	hits, err = searcher.SearchDocument(ctx, owner, kb, other, query)
	if err != nil || len(hits) == 0 {
		t.Fatalf("第二份制度没有证据：%v", err)
	}
	for _, hit := range hits {
		if hit.DocID != other || hit.DocVersion != 2 || !strings.Contains(hit.Snippet, "800") {
			t.Fatal("第二份制度包含了其他范围的证据")
		}
	}
	for _, scope := range [][3]int64{{owner + 1, kb, target}, {owner, kb + 1, target}, {owner, kb, target + other + 1}} {
		hits, err = searcher.SearchDocument(ctx, scope[0], scope[1], scope[2], query)
		if err != nil || len(hits) != 0 {
			t.Fatal("越权、跨库或错误资料应无证据", err)
		}
	}
	if _, err := db.ExecContext(ctx, "UPDATE ai_kb_doc SET status='failed' WHERE id=? AND kb_id=?", target, kb); err != nil {
		t.Fatal(err)
	}
	hits, err = searcher.SearchDocument(ctx, owner, kb, target, query)
	if err != nil || len(hits) != 0 {
		t.Fatal("Qdrant 仍有点但原资料已不可用，不得改用另一个额度", err)
	}
}
