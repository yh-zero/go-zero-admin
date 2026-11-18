package model

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

func TestKbDocPageAndChoicesOwnershipAndSignatures(t *testing.T) {
	h := newEvaluationStoreHarness(t)
	conf := cache.CacheConf{{RedisConf: redis.RedisConf{Host: os.Getenv("GOZERO_AI_TEST_REDIS"), Type: "node"}, Weight: 1}}
	docs, kbs := NewAiKbDocModel(h.conn, conf), NewAiKbModel(h.conn, conf)
	const title = "资料_%!投影"
	result, err := h.db.ExecContext(h.ctx, "INSERT INTO ai_kb_doc(kb_id,title,content) VALUES (?,?,?),(?,'其它','其它正文')", h.kb, title, strings.Repeat("正文", 100), h.kb)
	if err != nil {
		t.Fatal(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	full, err := docs.ReadPage(h.ctx, h.kb, h.owner, 1, 10, "_%!", "pending", false)
	if err != nil || full.Total != 1 || full.Pending != 2 || len(full.Rows) != 1 || full.Rows[0].Id != id || full.Rows[0].Content != "" || full.Rows[0].Title != title {
		t.Fatalf("筛选资料、全库计数或投影有误: %+v %v", full, err)
	}
	summary, err := docs.ReadPage(h.ctx, h.kb, h.owner, 1, 10, "_%!", "pending", true)
	if err != nil || len(summary.Rows) != 0 || full.Signature != summary.Signature || len(summary.Signature) != 64 {
		t.Fatalf("完整页与探测不一致: %+v %v", summary, err)
	}
	if _, err = h.db.ExecContext(h.ctx, "UPDATE ai_kb_doc SET ingest_stage='embedding', updated_at=updated_at WHERE id=? AND kb_id=?", id, h.kb); err != nil {
		t.Fatal(err)
	}
	changed, err := docs.ReadPage(h.ctx, h.kb, h.owner, 1, 10, "_%!", "pending", true)
	if err != nil || changed.Signature == full.Signature || changed.Pending != full.Pending {
		t.Fatal("同秒改变阶段，计数不变时仍应更新当前页", err)
	}
	empty, err := docs.ReadPage(h.ctx, h.kb, h.owner, 2, 10, "_%!", "pending", false)
	if err != nil || len(empty.Rows) != 0 || empty.Total != 1 || empty.Pending != 2 {
		t.Fatal("空页不丢失全库统计", err)
	}
	if _, err = docs.ReadPage(h.ctx, h.kb, h.owner+1, 1, 10, "", "", true); !errors.Is(err, ErrNotFound) {
		t.Fatal("越权不能探测资料进度", err)
	}
	for _, exact := range []int64{0, h.kb} {
		rows, total, err := kbs.ChoicesOwned(h.ctx, h.owner, exact, 1, 10, "质量验收隔离")
		if err != nil || total != 1 || len(rows) != 1 || rows[0].Id != h.kb || rows[0].Name == "" {
			t.Fatal("选库分页或原值回显错误", err)
		}
		rows, total, err = kbs.ChoicesOwned(h.ctx, h.owner+1, exact, 1, 10, "质量验收隔离")
		if err != nil || total != 0 || len(rows) != 0 {
			t.Fatal("不能返回他人库选项", err)
		}
	}
}

func TestEvaluationStoreProgressChangedAndUnchanged(t *testing.T) {
	h := newEvaluationStoreHarness(t)
	row, _, err := h.store.CreateOwned(h.ctx, h.draft(t))
	if err != nil {
		t.Fatal(err)
	}
	full, err := h.store.FindOwned(h.ctx, h.owner, h.kb, row.Id)
	if err != nil || full.Snapshot == "" || len(full.ObservationsSHA) != 64 {
		t.Fatal("首次读取保留原始快照和观测指纹", err)
	}
	progress, err := h.store.ProgressOwned(h.ctx, h.owner, h.kb, row.Id, full.ObservationsSHA)
	if err != nil || progress.ObservationsSHA != full.ObservationsSHA || progress.Observations != "" || progress.Bindings != "" || progress.Snapshot != "" || progress.Context != "" {
		t.Fatal("无变化轮询不能传送冻结数据和观测", err)
	}
	if _, err = h.db.ExecContext(h.ctx, "UPDATE ai_kb_evaluation SET observations=JSON_SET(observations,'$[0].stage','test-stage') WHERE id=? AND owner_id=? AND kb_id=?", row.Id, h.owner, h.kb); err != nil {
		t.Fatal(err)
	}
	progress, err = h.store.ProgressOwned(h.ctx, h.owner, h.kb, row.Id, full.ObservationsSHA)
	if err != nil || progress.ObservationsSHA == full.ObservationsSHA || !strings.Contains(progress.Observations, "test-stage") || progress.Snapshot != "" {
		t.Fatal("观测变化只传送新观测", err)
	}
	if _, err = h.store.ProgressOwned(h.ctx, h.owner+1, h.kb, row.Id, ""); !errors.Is(err, ErrNotFound) {
		t.Fatal("不能读取他人批次进度", err)
	}
}
