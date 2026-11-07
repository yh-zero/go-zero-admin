package aiservicelogic

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/vectordb"
	"go-zero-admin/pkg/xerr"
	"google.golang.org/protobuf/proto"
)

func backgroundDocument(h *kbIngestHarness, batch, title string) *aipb.CreateKbDocRequest {
	return &aipb.CreateKbDocRequest{Actor: h.actor, KbId: h.kb, Title: title, Content: "后台资料 " + title, RequestId: uuid.NewString(), BatchId: batch}
}

func TestKbBackgroundAcceptReplayReuseStopAndOwnership(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	batch := uuid.NewString()
	req := backgroundDocument(h, batch, "持久接收")
	first, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(req)
	if err != nil || first.Doc.Status != "pending" || first.Doc.Stage != "queued" || first.JobId == 0 || h.calls.Load() != 0 {
		t.Fatalf("accept=%+v err=%v calls=%d", first, err, h.calls.Load())
	}
	replay, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(req)
	if err != nil || replay.JobId != first.JobId || replay.Doc.Id != first.Doc.Id {
		t.Fatalf("replay=%+v err=%v", replay, err)
	}
	changed := proto.Clone(req).(*aipb.CreateKbDocRequest)
	changed.Content += " changed"
	if _, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(changed); !hasKbCode(err, 400) {
		t.Fatalf("request conflict=%v", err)
	}
	duplicate := proto.Clone(req).(*aipb.CreateKbDocRequest)
	duplicate.RequestId = uuid.NewString()
	reused, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(duplicate)
	if err != nil || reused.Doc.Id != first.Doc.Id || reused.JobId == first.JobId {
		t.Fatalf("reuse=%+v %v", reused, err)
	}
	view, err := NewGetKbImportLogic(ctx, h.s).GetKbImport(&aipb.GetKbImportRequest{Actor: h.actor, KbId: h.kb, BatchId: batch})
	if err != nil || view.Batch.Total != 2 || view.Batch.Queued != 1 || view.Batch.Reused != 1 || len(view.List) != 2 || view.List[1].Status != "reused" {
		t.Fatalf("batch=%+v err=%v", view, err)
	}
	stranger := &aipb.Actor{UserId: h.actor.UserId + 1}
	if _, err := NewGetKbImportLogic(ctx, h.s).GetKbImport(&aipb.GetKbImportRequest{Actor: stranger, KbId: h.kb, BatchId: batch}); !hasKbCode(err, 404) {
		t.Fatalf("ownership=%v", err)
	}
	stopped, err := NewStopKbImportLogic(ctx, h.s).StopKbImport(&aipb.StopKbImportRequest{Actor: h.actor, KbId: h.kb, BatchId: batch})
	if err != nil || !stopped.Batch.Stopped || stopped.Batch.Cancelled != 1 || stopped.Batch.Reused != 1 {
		t.Fatalf("stop=%+v err=%v", stopped, err)
	}
	if _, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(backgroundDocument(h, batch, "停止后追加")); !hasKbCode(err, 400) {
		t.Fatalf("stopped batch accepted new item: %v", err)
	}
	if out, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(req); err != nil || out.JobId != first.JobId {
		t.Fatalf("stopped replay=%+v %v", out, err)
	}
	h.background()
	time.Sleep(350 * time.Millisecond)
	if h.calls.Load() != 0 {
		t.Fatal("cancelled or reused job called embedding")
	}
	listed, err := NewListKbImportsLogic(ctx, h.s).ListKbImports(&aipb.ListKbImportsRequest{Actor: h.actor, KbId: h.kb})
	if err != nil || listed.Page.Total != 1 || !listed.List[0].Stopped {
		t.Fatalf("list=%+v %v", listed, err)
	}
}

func TestKbBackgroundTwoWorkersAndStopLeavesRunning(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	batch := uuid.NewString()
	var docs []int64
	for i := range 3 {
		out, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(backgroundDocument(h, batch, fmt.Sprint(i)))
		if err != nil {
			t.Fatal(err)
		}
		docs = append(docs, out.Doc.Id)
	}
	entered := make(chan struct{}, 3)
	release := make(chan struct{})
	h.embed = func(w http.ResponseWriter, _ *http.Request) {
		entered <- struct{}{}
		<-release
		_, _ = w.Write([]byte(`{"data":[{"index":0,"embedding":[1,0]}]}`))
	}
	h.background()
	for range 2 {
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			state, viewErr := NewGetKbImportLogic(ctx, h.s).GetKbImport(&aipb.GetKbImportRequest{Actor: h.actor, KbId: h.kb, BatchId: batch})
			t.Logf("two worker timeout state=%+v err=%v", state, viewErr)
			close(release)
			t.Fatal("two workers did not start")
		}
	}
	select {
	case <-entered:
		close(release)
		t.Fatal("worker count exceeded two")
	case <-time.After(200 * time.Millisecond):
	}
	view, err := NewStopKbImportLogic(ctx, h.s).StopKbImport(&aipb.StopKbImportRequest{Actor: h.actor, KbId: h.kb, BatchId: batch})
	if err != nil || view.Batch.Running != 2 || view.Batch.Cancelled != 1 {
		close(release)
		t.Fatalf("stop=%+v %v", view, err)
	}
	close(release)
	for _, id := range docs {
		if _, err := h.waitDoc(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	view, err = NewGetKbImportLogic(ctx, h.s).GetKbImport(&aipb.GetKbImportRequest{Actor: h.actor, KbId: h.kb, BatchId: batch})
	if err != nil || view.Batch.Succeeded != 2 || view.Batch.Cancelled != 1 || h.calls.Load() != 2 {
		t.Fatalf("result=%+v calls=%d err=%v", view, h.calls.Load(), err)
	}
}

func TestKbBackgroundRestartResumesQueuedAndCancelledRequestDoesNotReplay(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx, cancel := context.WithCancel(context.Background())
	req := backgroundDocument(h, "", "跨进程队列")
	out, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(req)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if h.calls.Load() != 0 {
		t.Fatal("request performed embedding")
	}
	// 重新组装队列模拟新进程；持久任务不依赖旧请求、内存channel或Redis队列。
	restarted := *h.s
	restarted.KbIngestQueue = model.NewKbIngestQueue(h.s.DB, h.s.AiKbDocModel)
	stop := restarted.StartKbIngestWorkers(context.Background())
	defer stop()
	if doc, err := h.waitDoc(context.Background(), out.Doc.Id); err != nil || doc.Status != "ready" {
		t.Fatalf("restart doc=%+v %v", doc, err)
	}
	replay, err := NewCreateKbDocLogic(context.Background(), &restarted).CreateKbDoc(req)
	if err != nil || replay.JobId != out.JobId || replay.BatchId != req.RequestId || h.calls.Load() != 1 {
		t.Fatalf("recovered=%+v calls=%d %v", replay, h.calls.Load(), err)
	}
}

func TestKbBackgroundExpiredRunningFailsWithoutEmbeddingAndCanExplicitlyRetry(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	req := backgroundDocument(h, "", "中断任务")
	out, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(req)
	if err != nil {
		t.Fatal(err)
	}
	old, err := h.s.KbIngestQueue.Claim(ctx, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.db.Exec("UPDATE ai_kb_ingest_job SET lease_until=DATE_SUB(NOW(),INTERVAL 1 SECOND) WHERE id=?", old.Job.Id); err != nil {
		t.Fatal(err)
	}
	h.background()
	doc, err := h.waitDoc(ctx, out.Doc.Id)
	if err != nil || doc.Status != "failed" || h.calls.Load() != 0 {
		t.Fatalf("expired=%+v calls=%d %v", doc, h.calls.Load(), err)
	}
	if err := h.s.KbIngestQueue.Transition(ctx, old, "succeeded", "", "text-embedding-v4", false); !errors.Is(err, model.ErrKbDocState) {
		t.Fatalf("late lease completed: %v", err)
	}
	retried, err := h.retry(out.Doc.Id)
	if err != nil || retried.Doc.Status != "ready" || retried.Doc.Version != 1 || h.calls.Load() != 1 {
		t.Fatalf("explicit retry=%+v calls=%d %v", retried, h.calls.Load(), err)
	}
}

func TestKbBackgroundExpiredUpdateCleansBothGenerationsWithoutEmbedding(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	first, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(backgroundDocument(h, "", "原索引"))
	if err != nil {
		t.Fatal(err)
	}
	original, err := h.s.KbIngestQueue.Claim(ctx, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if err := h.s.KbIngestQueue.Transition(ctx, original, "succeeded", "", "text-embedding-v4", false); err != nil {
		t.Fatal(err)
	}
	if _, err := NewUpdateKbDocLogic(ctx, h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: first.Doc.Id, Title: "新索引", Content: "新正文", ExpectedVersion: 1, RequestId: uuid.NewString()}); err != nil {
		t.Fatal(err)
	}
	attempt, err := h.s.KbIngestQueue.Claim(ctx, uuid.NewString())
	if err != nil || len(attempt.ChunkIds) == 0 || len(attempt.OldChunkIds) == 0 {
		t.Fatalf("更新作业必须保存新旧分块代次: attempt=%+v err=%v", attempt, err)
	}
	const unrelatedId uint64 = 1 << 63
	points := []vectordb.Point{{ID: unrelatedId}}
	for _, ids := range [][]uint64{attempt.OldChunkIds, attempt.ChunkIds} {
		for _, id := range ids {
			points = append(points, vectordb.Point{ID: id})
		}
	}
	if err := h.store.Upsert(ctx, points); err != nil {
		t.Fatal(err)
	}
	if _, err := h.db.Exec("UPDATE ai_kb_ingest_job SET lease_until=DATE_SUB(NOW(),INTERVAL 1 SECOND) WHERE id=?", attempt.Job.Id); err != nil {
		t.Fatal(err)
	}
	// 模拟领取后进程中断：只运行过期恢复，不允许开始任何新的 Embedding。
	h.s.Embedder = nil
	stop := h.s.StartKbIngestWorkers(ctx)
	defer stop()
	doc, err := h.waitDoc(ctx, first.Doc.Id)
	stop()
	if err != nil || doc.Status != "failed" || h.calls.Load() != 0 {
		t.Fatalf("过期恢复不得重放付费调用: doc=%+v calls=%d err=%v", doc, h.calls.Load(), err)
	}
	remaining, err := h.store.AllIDs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := remaining[unrelatedId]; !exists || len(remaining) != 1 {
		t.Fatalf("过期更新应只删除新旧两代向量并保留其他索引: remaining=%v", remaining)
	}
}

func TestKbBackgroundFiltersCountsAndBatchLimit(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	batch := uuid.NewString()
	var ids []int64
	for i := range 20 {
		title := fmt.Sprint("资料", i)
		if i == 0 {
			title = "百分号%资料"
		}
		out, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(backgroundDocument(h, batch, title))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, out.Doc.Id)
	}
	if _, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(backgroundDocument(h, batch, "第21项")); !hasKbCode(err, 400) {
		t.Fatalf("batch limit=%v", err)
	}
	if _, err := h.db.Exec("UPDATE ai_kb_doc SET status='ready',ingest_stage='ready' WHERE id=?", ids[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := h.db.Exec("UPDATE ai_kb_doc SET status='failed',ingest_stage='failed' WHERE id=?", ids[1]); err != nil {
		t.Fatal(err)
	}
	list, err := NewListKbDocsLogic(ctx, h.s).ListKbDocs(&aipb.ListKbDocsRequest{Actor: h.actor, KbId: h.kb, Query: "%", Status: "ready"})
	if err != nil || list.Page.Total != 1 || len(list.List) != 1 || list.List[0].Id != ids[0] || list.ReadyCount != 1 || list.PendingCount != 18 || list.FailedCount != 1 {
		t.Fatalf("filtered=%+v %v", list, err)
	}
	kbs, err := NewListKbsLogic(ctx, h.s).ListKbs(&aipb.ListKbsRequest{Actor: h.actor, Query: "摄取隔离"})
	if err != nil || kbs.Page.Total != 1 || kbs.List[0].DocCount != 20 || kbs.List[0].ReadyCount != 1 || kbs.List[0].PendingCount != 18 || kbs.List[0].FailedCount != 1 {
		t.Fatalf("kb counts=%+v %v", kbs, err)
	}
	if _, err := NewListKbDocsLogic(ctx, h.s).ListKbDocs(&aipb.ListKbDocsRequest{Actor: &aipb.Actor{UserId: h.actor.UserId + 1}, KbId: h.kb}); !hasKbCode(err, 404) {
		t.Fatalf("list ownership=%v", err)
	}
	var docs, jobs int64
	if err := h.db.QueryRow("SELECT COUNT(*) FROM ai_kb_doc WHERE kb_id=?", h.kb).Scan(&docs); err != nil {
		t.Fatal(err)
	}
	if err := h.db.QueryRow("SELECT COUNT(*) FROM ai_kb_ingest_job WHERE kb_id=?", h.kb).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if docs != 20 || jobs != 20 {
		t.Fatalf("overflow committed docs=%d jobs=%d", docs, jobs)
	}
}

func hasKbCode(err error, code int) bool {
	ce, ok := xerr.AsCodeError(err)
	return ok && int(ce.GetErrCode()) == code
}

func TestKbBackgroundReplacementRollbackAndLateLeaseCannotOverwrite(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	first, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(backgroundDocument(h, "", "原修订"))
	if err != nil {
		t.Fatal(err)
	}
	old, err := h.s.KbIngestQueue.Claim(ctx, uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	bad := model.KbEnqueueRequest{KbId: h.kb, DocId: first.Doc.Id, OwnerId: h.actor.UserId, Operation: "update", RequestId: uuid.NewString(), ExpectedVersion: 1, Title: "不能提交", Content: "新正文", EmbeddingModel: "text-embedding-v4"}
	if _, err := h.s.KbIngestQueue.Enqueue(ctx, bad, []model.KbChunkRow{{Seq: -1, Content: "坏分块"}}); err == nil {
		t.Fatal("invalid chunk committed")
	}
	var batches int
	if err := h.db.QueryRow("SELECT COUNT(*) FROM ai_kb_ingest_batch WHERE kb_id=?", h.kb).Scan(&batches); err != nil || batches != 1 {
		t.Fatalf("rollback left implicit batch count=%d err=%v", batches, err)
	}
	row, err := h.s.AiKbDocModel.FindOwned(ctx, h.kb, first.Doc.Id, h.actor.UserId)
	if err != nil || row.Version != 1 || row.Title != "原修订" {
		t.Fatalf("rollback=%+v %v", row, err)
	}
	updated, err := NewUpdateKbDocLogic(ctx, h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: first.Doc.Id, Title: "第二修订", Content: "保留新原文\r\n", ExpectedVersion: 1, RequestId: uuid.NewString()})
	if err != nil || updated.Doc.Version != 2 || updated.Doc.Stage != "queued" {
		t.Fatalf("update=%+v %v", updated, err)
	}
	if err := h.s.KbIngestQueue.Transition(ctx, old, "succeeded", "", "text-embedding-v4", false); !errors.Is(err, model.ErrKbDocState) {
		t.Fatalf("superseded lease accepted: %v", err)
	}
	h.background()
	if doc, err := h.waitDoc(ctx, first.Doc.Id); err != nil || doc.Version != 2 || doc.Status != "ready" || h.calls.Load() != 1 {
		t.Fatalf("new generation=%+v calls=%d err=%v", doc, h.calls.Load(), err)
	}
}

func TestKbBackgroundOldQueuedCannotBeRetriedButRequestCanBeRecovered(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	req := backgroundDocument(h, "", "长时间排队")
	out, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(req)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.db.Exec("UPDATE ai_kb_doc SET updated_at=DATE_SUB(NOW(),INTERVAL 121 SECOND) WHERE id=?", out.Doc.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := NewRetryKbDocLogic(ctx, h.s).RetryKbDoc(&aipb.RetryKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: out.Doc.Id, RequestId: uuid.NewString()}); !hasKbCode(err, 400) {
		t.Fatalf("old queued retry=%v", err)
	}
	recovered, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(req)
	if err != nil || recovered.JobId != out.JobId || recovered.Doc.Version != 1 || recovered.Doc.Stage != "queued" || h.calls.Load() != 0 {
		t.Fatalf("request recovery=%+v calls=%d err=%v", recovered, h.calls.Load(), err)
	}
	var jobs int
	if err := h.db.QueryRow("SELECT COUNT(*) FROM ai_kb_ingest_job WHERE kb_id=?", h.kb).Scan(&jobs); err != nil || jobs != 1 {
		t.Fatalf("retry created jobs=%d err=%v", jobs, err)
	}
}
