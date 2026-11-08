package aiservicelogic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/xerr"
)

func feedbackFixture(t *testing.T) (*workspaceHarness, int64, int64, int64) {
	t.Helper()
	h := newWorkspaceHarness(t)
	// 共用 harness 的连接与 Redis；缓存配置在测试中显式提供，禁止读取开发 YAML。
	_, address := workspaceEnvironment(t)
	conf := cache.CacheConf{{RedisConf: redis.RedisConf{Host: address, Type: "node"}, Weight: 100}}
	h.service.AiKbModel = model.NewAiKbModel(h.service.DB, conf)
	h.service.AiKbDocModel = model.NewAiKbDocModel(h.service.DB, conf)
	h.service.AiKbFeedbackModel = model.NewAiKbFeedbackModel(h.service.DB, conf)
	result, err := h.service.DB.ExecCtx(h.ctx, "INSERT INTO ai_kb (name,description,owner_id) VALUES (?,?,?)", "反馈隔离资料", "本轮回归", h.actor.UserId)
	if err != nil {
		t.Fatal(err)
	}
	kbId, _ := result.LastInsertId()
	content := "每个批次最多20篇，同时处理2篇。摄取失败可查看安全原因。"
	result, err = h.service.DB.ExecCtx(h.ctx, "INSERT INTO ai_kb_doc (kb_id,title,content,status,version) VALUES (?,?,?,'ready',1)", kbId, "摄取规则", content)
	if err != nil {
		t.Fatal(err)
	}
	docId, _ := result.LastInsertId()
	result, err = h.service.DB.ExecCtx(h.ctx, "INSERT INTO ai_kb_chunk (kb_id,doc_id,seq,content,token_count,status) VALUES (?,?,0,?,20,'ready')", kbId, docId, content)
	if err != nil {
		t.Fatal(err)
	}
	chunkId, _ := result.LastInsertId()
	analysis := agenttools.KbAnalysisResult{Version: 1, Type: "kb_analysis", Status: "completed", Analysis: "最多20篇，同时处理2篇 [1]", Sources: []agenttools.KbAnalysisSource{{Index: 1, KbSearchHit: agenttools.KbSearchHit{ChunkID: chunkId, KbID: kbId, DocID: docId, DocTitle: "摄取规则", Snippet: content, Score: .9}}}, Usage: agenttools.KbAnalysisUsage{Scope: "chat"}}
	output, _ := json.Marshal(analysis)
	input := fmt.Sprintf(`{"kb_id":%d,"query":"批次和并发限制是多少？"}`, kbId)
	steps, _ := json.Marshal([]model.WorkflowStepRecord{{Index: 0, Id: "analyze", Tool: "kb_analyze", Name: "分析", Input: input, Output: string(output), Status: "succeeded"}, {Index: 1, Id: "later", Tool: "text_stats", Status: "failed"}})
	result, err = h.service.AiTaskModel.Insert(h.ctx, &model.AiTask{CreatedBy: uint64(h.actor.UserId), Status: "failed", StepRecords: sql.NullString{String: string(steps), Valid: true}, NextStep: 1, RequestId: sql.NullString{String: uuid.NewString(), Valid: true}})
	if err != nil {
		t.Fatal(err)
	}
	taskId, _ := result.LastInsertId()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		var ids []int64
		if err := h.service.DB.QueryRowsCtx(ctx, &ids, "SELECT id FROM ai_kb_feedback WHERE owner_id=? AND kb_id=?", h.actor.UserId, kbId); err != nil {
			t.Error(err)
		}
		for _, id := range ids {
			if _, err := h.service.DB.ExecCtx(ctx, "DELETE FROM ai_kb_feedback WHERE id=? AND owner_id=?", id, h.actor.UserId); err != nil {
				t.Error(err)
			}
			if _, err := h.service.Redis.DelCtx(ctx, fmt.Sprintf("cache:gozeroAdmin:aiKbFeedback:id:%d", id)); err != nil {
				t.Error(err)
			}
		}
		if _, err := h.service.Redis.DelCtx(ctx, fmt.Sprintf("cache:gozeroAdmin:aiKbFeedback:ownerId:taskId:stepIndex:%d:%d:0", h.actor.UserId, taskId)); err != nil {
			t.Error(err)
		}
		if _, err := h.service.DB.ExecCtx(ctx, "DELETE FROM ai_kb_doc WHERE id=? AND kb_id=?", docId, kbId); err != nil {
			t.Error(err)
		}
		if _, err := h.service.DB.ExecCtx(ctx, "DELETE FROM ai_kb_chunk WHERE doc_id=? AND kb_id=?", docId, kbId); err != nil {
			t.Error(err)
		}
		if _, err := h.service.DB.ExecCtx(ctx, "DELETE FROM ai_kb WHERE id=? AND owner_id=?", kbId, h.actor.UserId); err != nil {
			t.Error(err)
		}
	})
	return h, kbId, docId, taskId
}

func TestKbFeedbackIntegrationLifecycleAndIsolation(t *testing.T) {
	h, kbId, docId, taskId := feedbackFixture(t)
	in := &aipb.SetTaskFeedbackRequest{Actor: h.actor, TaskId: taskId, Feedback: "incorrect_answer", Note: "核对数字"}
	created, err := NewSetTaskFeedbackLogic(h.ctx, h.service).SetTaskFeedback(in)
	if err != nil {
		t.Fatal(err)
	}
	row := created.Feedback
	if row.Revision != 1 || row.Status != "pending" || row.StepIndex != 0 || row.KbId != kbId || row.Question != "批次和并发限制是多少？" {
		t.Fatalf("评价目标或版本不符: %+v", row)
	}
	var snapshot feedbackSnapshot
	if json.Unmarshal([]byte(row.Snapshot), &snapshot) != nil || len(snapshot.Documents) != 1 || snapshot.Documents[0].Status != "verified" || len(snapshot.Documents[0].ContentSHA) != 64 {
		t.Fatal("真实来源指纹未保存")
	}
	originalSnapshot := row.Snapshot
	retry, err := NewSetTaskFeedbackLogic(h.ctx, h.service).SetTaskFeedback(in)
	if err != nil || retry.Feedback.Id != row.Id || retry.Feedback.Revision != 1 {
		t.Fatal("重复提交不得创建或改版本")
	}
	foreign := &aipb.Actor{UserId: h.actor.UserId + 1}
	_, err = NewListTaskFeedbackLogic(h.ctx, h.service).ListTaskFeedback(&aipb.ListTaskFeedbackRequest{Actor: foreign, TaskId: taskId})
	assertWorkspaceCode(t, err, 404)
	_, err = NewGetKbFeedbackLogic(h.ctx, h.service).GetKbFeedback(&aipb.GetKbFeedbackRequest{Actor: foreign, KbId: kbId, FeedbackId: row.Id})
	assertWorkspaceCode(t, err, 404)
	_, err = NewSetTaskFeedbackLogic(h.ctx, h.service).SetTaskFeedback(&aipb.SetTaskFeedbackRequest{Actor: foreign, TaskId: taskId, Feedback: "helpful"})
	assertWorkspaceCode(t, err, 404)
	_, err = NewSetTaskFeedbackLogic(h.ctx, h.service).SetTaskFeedback(&aipb.SetTaskFeedbackRequest{Actor: h.actor, TaskId: taskId, StepIndex: 1, Feedback: "helpful"})
	assertWorkspaceCode(t, err, 400)
	list, err := NewListKbFeedbackLogic(h.ctx, h.service).ListKbFeedback(&aipb.ListKbFeedbackRequest{Actor: h.actor, KbId: kbId, Page: &aipb.PageRequest{Page: 1, PageSize: 1}})
	if err != nil || len(list.List) != 1 || list.Page.Total != 1 || list.List[0].Snapshot != "" || list.List[0].Answer != "" || list.List[0].Review != nil {
		t.Fatal("分页须返回摘要，不读取大快照")
	}
	empty, err := NewListKbFeedbackLogic(h.ctx, h.service).ListKbFeedback(&aipb.ListKbFeedbackRequest{Actor: h.actor, KbId: kbId, Page: &aipb.PageRequest{Page: 2, PageSize: 1}})
	if err != nil || empty.List == nil || len(empty.List) != 0 || empty.Page.Total != 1 {
		t.Fatal("空分页或总数异常")
	}
	if _, err = h.service.AiKbFeedbackModel.FindOne(h.ctx, uint64(row.Id)); err != nil {
		t.Fatal(err)
	}
	if _, err = h.service.AiKbFeedbackModel.FindOneByOwnerIdTaskIdStepIndex(h.ctx, uint64(h.actor.UserId), uint64(taskId), 0); err != nil {
		t.Fatal(err)
	}
	review := &aipb.KbFeedbackReview{Answerable: true, ExpectedAnswer: "每批最多20篇，同时处理2篇。", RequiredFacts: []string{"20篇", "2篇"}, Evidence: []*aipb.KbFeedbackEvidence{{SourceIndex: 1, Quote: "最多20篇"}, {SourceIndex: 1, Quote: "同时处理2篇"}}, Reason: "已逐字核对引用"}
	accepted, err := NewReviewKbFeedbackLogic(h.ctx, h.service).ReviewKbFeedback(&aipb.ReviewKbFeedbackRequest{Actor: h.actor, KbId: kbId, FeedbackId: row.Id, ExpectedRevision: 1, Status: "accepted", Review: review})
	if err != nil || accepted.Feedback.Revision != 2 || accepted.Feedback.Status != "accepted" {
		t.Fatalf("复核采纳失败: %v", err)
	}
	cached, err := h.service.AiKbFeedbackModel.FindOne(h.ctx, uint64(row.Id))
	if err != nil || cached.Status != "accepted" || cached.Revision != 2 {
		t.Fatal("主键缓存未失效")
	}
	cached, err = h.service.AiKbFeedbackModel.FindOneByOwnerIdTaskIdStepIndex(h.ctx, uint64(h.actor.UserId), uint64(taskId), 0)
	if err != nil || cached.Revision != 2 {
		t.Fatal("唯一索引缓存未失效")
	}
	review.Evidence[0].Quote = "并不存在的依据"
	_, err = NewReviewKbFeedbackLogic(h.ctx, h.service).ReviewKbFeedback(&aipb.ReviewKbFeedbackRequest{Actor: h.actor, KbId: kbId, FeedbackId: row.Id, ExpectedRevision: 2, Status: "accepted", Review: review})
	assertWorkspaceCode(t, err, 400)
	in.ExpectedRevision = 1
	in.Note = "迟到评价"
	_, err = NewSetTaskFeedbackLogic(h.ctx, h.service).SetTaskFeedback(in)
	assertWorkspaceCode(t, err, 409)
	in.ExpectedRevision = 2
	in.Note = "补充说明"
	updated, err := NewSetTaskFeedbackLogic(h.ctx, h.service).SetTaskFeedback(in)
	if err != nil || updated.Feedback.Revision != 3 || updated.Feedback.Status != "pending" || updated.Feedback.Review != nil || updated.Feedback.ReviewedAt != 0 || updated.Feedback.Snapshot != originalSnapshot {
		t.Fatal("改评价须清空复核，保持原快照")
	}
	if _, err := h.service.DB.ExecCtx(h.ctx, "UPDATE ai_kb_doc SET content='后来修改的资料',version=version+1 WHERE id=? AND kb_id=?", docId, kbId); err != nil {
		t.Fatal(err)
	}
	detail, err := NewGetKbFeedbackLogic(h.ctx, h.service).GetKbFeedback(&aipb.GetKbFeedbackRequest{Actor: h.actor, KbId: kbId, FeedbackId: row.Id})
	if err != nil || detail.Feedback.Snapshot != originalSnapshot {
		t.Fatal("原文更新不能改写历史依据")
	}
	if _, err := h.service.AiKbModel.DeleteOwned(h.ctx, kbId, h.actor.UserId); err != nil {
		t.Fatal(err)
	}
	_, err = NewGetKbFeedbackLogic(h.ctx, h.service).GetKbFeedback(&aipb.GetKbFeedbackRequest{Actor: h.actor, KbId: kbId, FeedbackId: row.Id})
	assertWorkspaceCode(t, err, 404)
	_, err = NewListKbFeedbackLogic(h.ctx, h.service).ListKbFeedback(&aipb.ListKbFeedbackRequest{Actor: h.actor, KbId: kbId})
	assertWorkspaceCode(t, err, 404)
	_, err = NewReviewKbFeedbackLogic(h.ctx, h.service).ReviewKbFeedback(&aipb.ReviewKbFeedbackRequest{Actor: h.actor, KbId: kbId, FeedbackId: row.Id, ExpectedRevision: 3, Status: "dismissed", Review: &aipb.KbFeedbackReview{Reason: "关闭"}})
	assertWorkspaceCode(t, err, 404)
}

func TestKbFeedbackIntegrationConcurrentReviewAndMissingSource(t *testing.T) {
	h, kbId, docId, taskId := feedbackFixture(t)
	if _, err := h.service.DB.ExecCtx(h.ctx, "UPDATE ai_kb_doc SET content='其他文本',version=2 WHERE id=? AND kb_id=?", docId, kbId); err != nil {
		t.Fatal(err)
	}
	var created [2]*aipb.KbFeedbackResponse
	var errs [2]error
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			created[i], errs[i] = NewSetTaskFeedbackLogic(h.ctx, h.service).SetTaskFeedback(&aipb.SetTaskFeedbackRequest{Actor: h.actor, TaskId: taskId, Feedback: "missing_knowledge"})
		}(i)
	}
	wg.Wait()
	if errs[0] != nil || errs[1] != nil || created[0].Feedback.Id != created[1].Feedback.Id || created[0].Feedback.Revision != 1 {
		t.Fatalf("并发相同评价须仅创建一次: %v", errs)
	}
	row := created[0].Feedback
	if !strings.Contains(row.Snapshot, `"status": "stale"`) && !strings.Contains(row.Snapshot, `"status":"stale"`) {
		t.Fatal("变更资料须标记不可直接回归")
	}
	for i := range 2 {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = NewReviewKbFeedbackLogic(h.ctx, h.service).ReviewKbFeedback(&aipb.ReviewKbFeedbackRequest{Actor: h.actor, KbId: kbId, FeedbackId: row.Id, ExpectedRevision: 1, Status: "dismissed", Review: &aipb.KbFeedbackReview{Reason: fmt.Sprintf("复核%d", i)}})
		}(i)
	}
	wg.Wait()
	succeeded, conflicted := 0, 0
	for _, err := range errs {
		if err == nil {
			succeeded++
		} else if code, ok := xerr.AsCodeError(err); ok && code.GetErrCode() == 409 {
			conflicted++
		} else {
			t.Fatal(err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("竞争复核应一成功一409: %v", errs)
	}
	final, err := h.service.AiKbFeedbackModel.FindOwned(h.ctx, h.actor.UserId, kbId, row.Id)
	if err != nil || final.Revision != 2 || final.Snapshot != row.Snapshot {
		t.Fatal("竞争后版本或快照不一致")
	}
}
