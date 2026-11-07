package aiservicelogic

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/xerr"
)

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
	rows, total, err := h.s.AiKbDocModel.ListInKb(context.Background(), h.kb, h.actor.UserId, 1, 20)
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Content != "" {
		t.Fatalf("list projection rows=%+v total=%d err=%v", rows, total, err)
	}
	body := "\ufeff资料原文\r\n第二行\r\n"
	updated, err := NewUpdateKbDocLogic(context.Background(), h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, Title: " 新标题 ", Content: body, ExpectedVersion: 1})
	if err != nil || updated.Doc.Version != 2 || updated.Doc.Stage != "ready" || updated.Doc.Status != "ready" {
		t.Fatalf("update=%+v err=%v", updated, err)
	}
	if got := get(false); got.Content != body || got.Doc.Title != "新标题" {
		t.Fatalf("original body changed: %+v", got)
	}
	calls := h.calls.Load()
	_, err = NewUpdateKbDocLogic(context.Background(), h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, Title: "旧版本", Content: "不能覆盖", ExpectedVersion: 1})
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
	if hits, err := h.s.AiKbChunkModel.FindReadyByIDs(ctx, old, h.actor.UserId, h.kb); err != nil || len(hits) != 0 {
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
	_, err := NewUpdateKbDocLogic(context.Background(), h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: stranger, KbId: h.kb, DocId: id, Title: "越权", Content: "越权", ExpectedVersion: 1})
	if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 404 {
		t.Fatalf("update ownership err=%v", err)
	}
	for _, content := range []string{"", " \n\t", strings.Repeat("中", 21846), string([]byte{0xff})} {
		_, err := NewUpdateKbDocLogic(context.Background(), h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, Title: "正文校验", Content: content, ExpectedVersion: 1})
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
	list, total, err := h.s.AiKbDocModel.ListInKb(ctx, h.kb, h.actor.UserId, 1, 20)
	if err != nil || total != 1 || len(list) != 1 || list[0].Content != "" {
		t.Fatalf("list projection failed %v", err)
	}
	metadata, err := h.s.AiKbDocModel.FindMetadataOwned(ctx, h.kb, id, h.actor.UserId)
	if err != nil || metadata.Content != "" {
		t.Fatalf("metadata projection failed %v", err)
	}
	t.Logf("同一64KiB文档：旧列表SELECT含正文会扫描返回65536正文bytes；新列表Content=%d bytes（列表仍2条SQL）；单ID状态GET Content=%d bytes且1条SELECT，不再扫描分页", len(list[0].Content), len(metadata.Content))
}

func TestKbDocumentsUpdateFailureRetainsNewRevisionForRetry(t *testing.T) {
	h := newKbIngestHarness(t)
	id, _ := h.pending()
	h.embed = func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "isolated provider failure", 500) }
	out, err := NewUpdateKbDocLogic(context.Background(), h.s).UpdateKbDoc(&aipb.UpdateKbDocRequest{Actor: h.actor, KbId: h.kb, DocId: id, Title: "更新后的资料", Content: "失败后仍应保留的新正文\r\n", ExpectedVersion: 1})
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
