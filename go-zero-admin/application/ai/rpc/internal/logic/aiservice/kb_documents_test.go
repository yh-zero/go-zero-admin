package aiservicelogic

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"net/http"
	"strings"
	"sync"
	"testing"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/xerr"
)

func TestKbDocumentsReadyEditReusesIndexAndReceipts(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	initial, err := NewCreateKbDocLogic(ctx, h.s).CreateKbDoc(&aipb.CreateKbDocRequest{
		Actor: h.actor, KbId: h.kb, Title: "原题", Content: "原文\r\n", RequestId: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	stop := h.s.StartKbIngestWorkers(ctx)
	ready, err := h.waitDoc(ctx, initial.Doc.Id)
	stop()
	if err != nil || ready.Status != "ready" || h.calls.Load() != 1 {
		t.Fatalf("初次摄取失败: %v", err)
	}
	id := initial.Doc.Id
	var chunkID uint64
	if err = h.db.QueryRow("SELECT id FROM ai_kb_chunk WHERE doc_id=?", id).Scan(&chunkID); err != nil {
		t.Fatal(err)
	}
	// 预热生成模型的文档缓存，验证改名提交后按原键失效。
	cached, err := h.s.AiKbDocModel.FindOne(ctx, uint64(id))
	if err != nil || cached.Title != "原题" {
		t.Fatal("初始标题读取失败", err)
	}
	h.s.Embedder = nil
	h.s.VectorStore = nil
	update := NewUpdateKbDocLogic(ctx, h.s).UpdateKbDoc
	rename := &aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, ExpectedVersion: 1, Title: "新题", Content: "原文\r\n", RequestId: uuid.NewString()}
	result, err := update(rename)
	if err != nil || result.Doc.Version != 2 || result.Doc.Title != "新题" || result.Doc.Status != "ready" || result.Doc.Stage != "ready" || result.JobId <= 0 {
		t.Fatalf("纯改名未复用: %v %v", result, err)
	}
	replayed, err := update(rename)
	if err != nil || replayed.JobId != result.JobId || replayed.BatchId != result.BatchId || replayed.Doc.Version != 2 {
		t.Fatalf("改名重放失败: %v %v", replayed, err)
	}
	cached, err = h.s.AiKbDocModel.FindOne(ctx, uint64(id))
	if err != nil || cached.Title != "新题" {
		t.Fatal("缓存未更新", err)
	}
	unchanged := &aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, ExpectedVersion: 2, Title: "新题", Content: "原文\r\n", RequestId: uuid.NewString()}
	next, err := update(unchanged)
	if err != nil || next.Doc.Version != 2 || next.JobId == result.JobId {
		t.Fatalf("无变更回执错误: %v %v", next, err)
	}
	for _, job := range []*aipb.UpdateKbDocResponse{result, next} {
		view, err := NewGetKbImportLogic(ctx, h.s).GetKbImport(&aipb.GetKbImportRequest{Actor: h.actor, KbId: h.kb, BatchId: job.BatchId})
		if err != nil || view.Batch.Reused != 1 || len(view.List) != 1 || view.List[0].Status != "reused" {
			t.Fatalf("正式批次回执错误: %v %v", view, err)
		}
	}
	hits, err := h.s.AiKbChunkModel.FindReadyByIDs(ctx, []uint64{chunkID}, h.actor.UserId, h.kb, 0)
	if err != nil || len(hits) != 1 || hits[0].DocTitle != "新题" || hits[0].Content != "原文" || h.calls.Load() != 1 {
		t.Fatalf("旧分块/标题未复用: %v %v", hits, err)
	}
	if points, err := h.store.AllIDs(ctx); err != nil || len(points) != 1 {
		t.Fatalf("索引变化: %v", err)
	} else if _, ok := points[chunkID]; !ok {
		t.Fatal("索引 ID 变化")
	}
	unchanged.Title = "篡改同一请求"
	if _, err = update(unchanged); !hasKbCode(err, 400) {
		t.Fatalf("同ID不同载荷应冲突: %v", err)
	}
	unchanged.RequestId = uuid.NewString()
	unchanged.ExpectedVersion = 1
	if _, err = update(unchanged); !hasKbCode(err, 400) {
		t.Fatalf("过期版本应冲突: %v", err)
	}
	unchanged.ExpectedVersion = 2
	unchanged.Actor = &aipb.Actor{UserId: h.actor.UserId + 1}
	if _, err = update(unchanged); !hasKbCode(err, 404) {
		t.Fatalf("跨身份应拒绝: %v", err)
	}
	unchanged.Actor = h.actor
	unchanged.Content = "原文\n"
	if _, err = update(unchanged); !hasKbCode(err, 503) {
		t.Fatalf("换行变化是正文更新，缺少摄取能力应拒绝: %v", err)
	}
	unchanged.Content = "原文\r\n"
	h.s.Config.AI.EmbeddingModel = "different"
	if _, err = update(unchanged); !hasKbCode(err, 503) {
		t.Fatalf("模型变化需要重新摄取: %v", err)
	}
}

type failedTitleInvalidation struct {
	model.AiKbDocModel
	fail bool
}

func (m *failedTitleInvalidation) InvalidateCommitted(ctx context.Context, id int64) error {
	if m.fail {
		return &model.KbDocCommittedError{Err: errors.New("isolated cache failure")}
	}
	return m.AiKbDocModel.InvalidateCommitted(ctx, id)
}

func TestKbDocumentsRenameCacheFailureAndReplay(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	id, ids := h.pending()
	if err := h.s.AiKbDocModel.MarkIngested(ctx, id, ids, h.s.Config.AI.EmbeddingModel); err != nil {
		t.Fatal(err)
	}
	old, err := h.s.AiKbDocModel.FindOne(ctx, uint64(id))
	if err != nil || old.Title != "待重试" {
		t.Fatal("缓存预热失败", err)
	}
	fault := &failedTitleInvalidation{AiKbDocModel: h.s.AiKbDocModel, fail: true}
	h.s.KbIngestQueue = model.NewKbIngestQueue(h.s.DB, fault)
	rename := &aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, ExpectedVersion: 1, Title: "新标题", Content: "正文", RequestId: uuid.NewString()}
	update := NewUpdateKbDocLogic(ctx, h.s).UpdateKbDoc
	if _, err = update(rename); !hasKbCode(err, 503) {
		t.Fatalf("已提交的缓存失败应返回503: %v", err)
	}
	current, err := h.s.AiKbDocModel.FindMetadataOwned(ctx, h.kb, id, h.actor.UserId)
	if err != nil || current.Title != "新标题" || current.Version != 2 || current.Status != "ready" {
		t.Fatalf("已提交改名被回滚: %v %v", current, err)
	}
	if _, err = update(rename); !hasKbCode(err, 503) {
		t.Fatalf("缓存仍失败时不能确认恢复: %v", err)
	}
	fault.fail = false
	receipt, err := update(rename)
	if err != nil || receipt.JobId <= 0 || receipt.Doc.Version != 2 {
		t.Fatalf("原请求核对失败: %v %v", receipt, err)
	}
	cached, err := h.s.AiKbDocModel.FindOne(ctx, uint64(id))
	if err != nil || cached.Title != "新标题" {
		t.Fatalf("重放未修复缓存: %v", err)
	}
	var jobs int
	if err = h.db.QueryRow("SELECT COUNT(*) FROM ai_kb_ingest_job WHERE kb_id=?", h.kb).Scan(&jobs); err != nil || jobs != 1 || h.calls.Load() != 0 {
		t.Fatalf("重放重复执行: jobs=%d err=%v", jobs, err)
	}
}

func TestKbDocumentsRenameVersionRaceAndRollback(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	id, ids := h.pending()
	if err := h.s.AiKbDocModel.MarkIngested(ctx, id, ids, h.s.Config.AI.EmbeddingModel); err != nil {
		t.Fatal(err)
	}
	r := model.KbEnqueueRequest{OwnerId: h.actor.UserId, KbId: h.kb, DocId: id, ExpectedVersion: 1, Title: "改名甲", Content: "正文", EmbeddingModel: h.s.Config.AI.EmbeddingModel, Operation: "update", RequestId: uuid.NewString()}
	r.SplitUpdatedContent = func(string) []model.KbChunkRow { panic("纯改名不应执行分块") }
	failure := errors.New("回执写入失败")
	queue := model.NewKbIngestQueue(&kbRequestFaultConn{SqlConn: h.s.DB, insertError: failure}, h.s.AiKbDocModel)
	if _, err := queue.Enqueue(ctx, r, nil); !errors.Is(err, failure) {
		t.Fatalf("未传播回执写入失败: %v", err)
	}
	doc, err := h.s.AiKbDocModel.FindMetadataOwned(ctx, h.kb, id, h.actor.UserId)
	if err != nil || doc.Version != 1 || doc.Title != "待重试" || doc.Status != "ready" {
		t.Fatalf("失败回执遗留改名: %v %v", doc, err)
	}
	answers := make(chan error, 2)
	for _, title := range []string{"改名甲", "改名乙"} {
		request := r
		request.Title = title
		request.RequestId = uuid.NewString()
		go func() { _, err := h.s.KbIngestQueue.Enqueue(ctx, request, nil); answers <- err }()
	}
	var success, conflict int
	for range 2 {
		switch err := <-answers; {
		case err == nil:
			success++
		case errors.Is(err, model.ErrKbDocState):
			conflict++
		default:
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("并发改名 success=%d conflict=%d", success, conflict)
	}
}

func TestKbDocumentsDetailMetadataAndVersionedUpdate(t *testing.T) {
	h := newKbIngestHarness(t)
	created, err := h.create(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	id := created.Doc.Id
	get := func(meta bool) *aipb.GetKbDocResponse {
		t.Helper()
		out, err := NewGetKbDocLogic(context.Background(), h.s).GetKbDoc(&aipb.GetKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, MetaOnly: meta})
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	if got := get(false); got.Content != "相同正文" || got.Doc.Version != 1 || got.Doc.Stage != "ready" {
		t.Fatalf("initial detail=%+v", got)
	}
	if got := get(true); got.Content != "" {
		t.Fatal("metadata request fetched original body")
	}
	page, err := h.s.AiKbDocModel.ReadPage(context.Background(), h.kb, h.actor.UserId, 1, 20, "", "", false)
	if err != nil || page.Total != 1 || len(page.Rows) != 1 || page.Rows[0].Content != "" {
		t.Fatalf("list projection page=%+v err=%v", page, err)
	}
	body := "\ufeff资料原文\r\n第二行\r\n"
	updated, err := NewUpdateKbDocLogic(context.Background(), h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, Title: " 新标题 ", Content: body, ExpectedVersion: 1, RequestId: uuid.NewString()})
	if err == nil {
		updated.Doc, err = h.waitDoc(context.Background(), id)
	}
	if err != nil || updated.Doc.Version != 2 || updated.Doc.Stage != "ready" || updated.Doc.Status != "ready" {
		t.Fatalf("update=%+v err=%v", updated, err)
	}
	if got := get(false); got.Content != body || got.Doc.Title != "新标题" {
		t.Fatalf("original body changed: %+v", got)
	}
	calls := h.calls.Load()
	_, err = NewUpdateKbDocLogic(context.Background(), h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, Title: "旧版本", Content: "不能覆盖", ExpectedVersion: 1, RequestId: uuid.NewString()})
	if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 400 || calls != h.calls.Load() {
		t.Fatalf("version conflict err=%v calls=%d", err, h.calls.Load())
	}
	if got := get(false); got.Content != body || got.Doc.Version != 2 {
		t.Fatal("stale update overwrote document")
	}
}

func TestKbDocumentsAtomicReplaceRejectsLateAttemptsAndRollsBack(t *testing.T) {
	h := newKbIngestHarness(t)
	id, old := h.pending()
	ctx := context.Background()
	_, next, err := h.s.AiKbDocModel.ReplaceDocument(ctx, h.actor.UserId, 1, &model.KbDocRow{Id: id, KbId: h.kb, Title: "新修订", Content: "新内容", EmbeddingModel: "text-embedding-v4"}, []model.KbChunkRow{{Content: "新内容"}})
	if err != nil || len(next) != 1 || next[0] == old[0] {
		t.Fatalf("replacement=%v err=%v", next, err)
	}
	for _, attempt := range []func() error{
		func() error { return h.s.AiKbDocModel.MarkIngested(ctx, id, old, "text-embedding-v4") },
		func() error { return h.s.AiKbDocModel.MarkFailed(ctx, id, old, "迟到失败") },
		func() error { return h.s.AiKbDocModel.MarkIndexing(ctx, id, old) },
	} {
		if err := attempt(); !errors.Is(err, model.ErrKbDocState) {
			t.Fatalf("late generation accepted: %v", err)
		}
	}
	if hits, err := h.s.AiKbChunkModel.FindReadyByIDs(ctx, old, h.actor.UserId, h.kb, 0); err != nil || len(hits) != 0 {
		t.Fatalf("old chunks still searchable: %+v %v", hits, err)
	}
	_, _, err = h.s.AiKbDocModel.ReplaceDocument(ctx, h.actor.UserId, 2, &model.KbDocRow{Id: id, KbId: h.kb, Title: "失败写入", Content: "不可提交"}, []model.KbChunkRow{{Seq: -1, Content: "坏分块"}})
	if err == nil {
		t.Fatal("invalid chunk unexpectedly committed")
	}
	got, err := h.s.AiKbDocModel.FindOwned(ctx, h.kb, id, h.actor.UserId)
	if err != nil || got.Version != 2 || got.Content != "新内容" || got.Title != "新修订" {
		t.Fatalf("rollback lost prior revision: %+v err=%v", got, err)
	}
	if err := h.s.AiKbDocModel.MarkIngested(ctx, id, next, "text-embedding-v4"); err != nil {
		t.Fatalf("rollback replaced previous chunks: %v", err)
	}
}

func TestKbDocumentsConcurrentUpdatesOnlyOneVersionWins(t *testing.T) {
	h := newKbIngestHarness(t)
	id, _ := h.pending()
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, content := range []string{"候选甲", "候选乙"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _, err := h.s.AiKbDocModel.ReplaceDocument(context.Background(), h.actor.UserId, 1, &model.KbDocRow{Id: id, KbId: h.kb, Title: content, Content: content}, []model.KbChunkRow{{Content: content}})
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	var successes, conflicts int
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, model.ErrKbDocState) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d", successes, conflicts)
	}
}

func TestKbDocumentsRetryRejectsBodyReadBeforeConcurrentEdit(t *testing.T) {
	h := newKbIngestHarness(t)
	id, _ := h.pending()
	ctx := context.Background()
	_, next, err := h.s.AiKbDocModel.ReplaceDocument(ctx, h.actor.UserId, 1, &model.KbDocRow{Id: id, KbId: h.kb, Title: "新修订", Content: "新内容"}, []model.KbChunkRow{{Content: "新内容"}})
	if err != nil {
		t.Fatal(err)
	}
	if err := h.s.AiKbDocModel.MarkFailed(ctx, id, next, "本地失败"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := h.s.AiKbDocModel.ReplaceChunksForRetry(ctx, h.kb, id, h.actor.UserId, []model.KbChunkRow{{Content: "编辑前的旧正文"}}, 1); !errors.Is(err, model.ErrKbDocState) {
		t.Fatalf("retry accepted stale body: %v", err)
	}
	row, err := h.s.AiKbDocModel.FindOwned(ctx, h.kb, id, h.actor.UserId)
	if err != nil || row.Content != "新内容" || row.Version != 2 || row.Status != "failed" {
		t.Fatalf("retry changed current revision: %+v %v", row, err)
	}
	if _, _, err := h.s.AiKbDocModel.ReplaceChunksForRetry(ctx, h.kb, id, h.actor.UserId, []model.KbChunkRow{{Content: row.Content}}, row.Version); err != nil {
		t.Fatal(err)
	}
}

func TestKbDocumentsOwnershipDeletedAndInvalidInput(t *testing.T) {
	h := newKbIngestHarness(t)
	id, _ := h.pending()
	stranger := &aipb.Actor{UserId: h.actor.UserId + 1}
	for _, meta := range []bool{false, true} {
		_, err := NewGetKbDocLogic(context.Background(), h.s).GetKbDoc(&aipb.GetKbDocRequest{Actor: stranger, KbId: h.kb, DocId: id, MetaOnly: meta})
		if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 404 {
			t.Fatalf("read ownership err=%v", err)
		}
	}
	_, err := NewUpdateKbDocLogic(context.Background(), h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: stranger, KbId: h.kb, DocId: id, Title: "越权", Content: "越权", ExpectedVersion: 1, RequestId: uuid.NewString()})
	if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 404 {
		t.Fatalf("update ownership err=%v", err)
	}
	for _, content := range []string{"", " \n\t", strings.Repeat("中", 21846), string([]byte{0xff})} {
		_, err := NewUpdateKbDocLogic(context.Background(), h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, Title: "正文校验", Content: content, ExpectedVersion: 1, RequestId: uuid.NewString()})
		if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 400 {
			t.Fatalf("invalid body accepted: %v", err)
		}
	}
	if h.calls.Load() != 0 {
		t.Fatal("invalid input or actor called embedding")
	}
	if _, err := h.s.AiKbDocModel.DeleteOwned(context.Background(), h.kb, id, h.actor.UserId); err != nil {
		t.Fatal(err)
	}
	_, err = NewGetKbDocLogic(context.Background(), h.s).GetKbDoc(&aipb.GetKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id})
	if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 404 {
		t.Fatalf("deleted read err=%v", err)
	}
}

func TestKbDocumentsProjectionPerformanceEvidence(t *testing.T) {
	h := newKbIngestHarness(t)
	ctx := context.Background()
	body := strings.Repeat("a", 65536)
	id, _, _, err := h.s.AiKbDocModel.InsertWithChunks(ctx, h.actor.UserId, &model.KbDocRow{KbId: h.kb, Title: "64KiB读取证据", Content: body}, chunkText(body))
	if err != nil {
		t.Fatal(err)
	}
	full, err := h.s.AiKbDocModel.FindOwned(ctx, h.kb, id, h.actor.UserId)
	if err != nil || len(full.Content) != 65536 {
		t.Fatalf("fixture=%+v %v", full, err)
	}
	page, err := h.s.AiKbDocModel.ReadPage(ctx, h.kb, h.actor.UserId, 1, 20, "", "", false)
	if err != nil || page.Total != 1 || len(page.Rows) != 1 || page.Rows[0].Content != "" {
		t.Fatalf("list projection failed %v", err)
	}
	metadata, err := h.s.AiKbDocModel.FindMetadataOwned(ctx, h.kb, id, h.actor.UserId)
	if err != nil || metadata.Content != "" {
		t.Fatalf("metadata projection failed %v", err)
	}
	t.Logf("同一64KiB文档：分页元信息正文 %d 字节，单一 SQL 返回当前页和统计。单文档元信息正文 %d 字节", len(page.Rows[0].Content), len(metadata.Content))
}

func TestKbDocumentsUpdateFailureRetainsNewRevisionForRetry(t *testing.T) {
	h := newKbIngestHarness(t)
	id, _ := h.pending()
	h.embed = func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "isolated provider failure", 500) }
	h.background()
	out, err := NewUpdateKbDocLogic(context.Background(), h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, Title: "更新后的资料", Content: "失败后仍应保留的新正文\r\n", ExpectedVersion: 1, RequestId: uuid.NewString()})
	if err == nil {
		out.Doc, err = h.waitDoc(context.Background(), id)
	}
	if err != nil || out.Doc.Id != id || out.Doc.Version != 2 || out.Doc.Status != "failed" || out.Doc.Stage != "failed" {
		t.Fatalf("failed update=%+v err=%v", out, err)
	}
	row, err := h.s.AiKbDocModel.FindOwned(context.Background(), h.kb, id, h.actor.UserId)
	if err != nil || row.Content != "失败后仍应保留的新正文\r\n" {
		t.Fatalf("failed ingestion lost original body %v", err)
	}
	h.embed = nil
	retried, err := h.retry(id)
	if err != nil || retried.Doc.Id != id || retried.Doc.Version != 2 || retried.Doc.Status != "ready" || retried.Doc.Stage != "ready" {
		t.Fatalf("retry=%+v err=%v", retried, err)
	}
	if h.calls.Load() != 2 {
		t.Fatalf("calls=%d want exactly failed update + retry", h.calls.Load())
	}
}
