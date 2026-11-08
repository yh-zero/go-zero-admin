package aiservicelogic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
)

func sourceLimitAnalysis(n int) agenttools.KbAnalysisResult {
	a := agenttools.KbAnalysisResult{Version: 1, Type: "kb_analysis", Status: "completed", Analysis: "实际事实[1]", Sources: []agenttools.KbAnalysisSource{}, Usage: agenttools.KbAnalysisUsage{Scope: "chat"}}
	for i := 0; i < n; i++ {
		a.Sources = append(a.Sources, agenttools.KbAnalysisSource{Index: i + 1, KbSearchHit: agenttools.KbSearchHit{ChunkID: int64(201 + i), DocID: int64(101 + i), KbID: 5, DocTitle: "原文", Snippet: fmt.Sprintf("事实%d。", i+1), Score: .8}})
	}
	return a
}

func TestAnalysisSourceLimitEightAndLegacySnapshotsRemainValid(t *testing.T) {
	for _, count := range []int{1, 5, 6, 8, 9} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			analysis := sourceLimitAnalysis(count)
			raw, _ := json.Marshal(analysis)
			input := `{"kb_id":5,"query":"目标"}`
			task := &aipb.AgentTaskInfo{Steps: []*aipb.WorkflowStepInfo{{Index: 0, Tool: "kb_analyze", Status: "succeeded", Input: input, Output: string(raw)}}}
			_, evaluationErr := evaluationAnalysis(task, 5, "目标")
			records, _ := json.Marshal([]model.WorkflowStepRecord{{Index: 0, Tool: "kb_analyze", Status: "succeeded", Input: input, Output: string(raw)}})
			snapshot, _, _, feedbackErr := analysisFeedbackSnapshot(&model.TaskRow{StepRecords: string(records)}, 0)
			if count <= 8 {
				if evaluationErr != nil || feedbackErr != nil || snapshot == nil || len(snapshot.Sources) != count {
					t.Fatalf("新八来源及原快照须可读: eval=%v feedback=%v", evaluationErr, feedbackErr)
				}
			} else if evaluationErr == nil || feedbackErr == nil {
				t.Fatal("九来源必须拒绝，不能截成八个")
			}
		})
	}
}

func TestFeedbackEighthSourceCanBeReviewedButEvidenceCountStaysFive(t *testing.T) {
	analysis := sourceLimitAnalysis(8)
	snapshot, _ := json.Marshal(feedbackSnapshot{Version: 1, Sources: analysis.Sources})
	review := &aipb.KbFeedbackReview{Answerable: true, ExpectedAnswer: "实际事实", RequiredFacts: []string{"事实8"}, Evidence: []*aipb.KbFeedbackEvidence{{SourceIndex: 8, Quote: "事实8"}}, Reason: "实际原文核对"}
	if err := validateReviewShape(review, "accepted"); err != nil {
		t.Fatalf("第八来源应可复核: %v", err)
	}
	if err := validateReviewEvidence(string(snapshot), review); err != nil {
		t.Fatalf("第八来源摘录须对应原文: %v", err)
	}
	review.Evidence[0].SourceIndex = 9
	if err := validateReviewShape(review, "accepted"); err == nil {
		t.Fatal("第九来源索引须前置拒绝")
	}
	review.Evidence = nil
	for i := 0; i < 6; i++ {
		review.Evidence = append(review.Evidence, &aipb.KbFeedbackEvidence{SourceIndex: int64(i + 1), Quote: "事实"})
	}
	if err := validateReviewShape(review, "accepted"); err == nil {
		t.Fatal("复核摘录仍最多五条，不能跟随来源数扩为八条")
	}
}

type sourceLimitCapture struct {
	model.AiKbFeedbackModel
	calls int
	ids   []int64
}

func (m *sourceLimitCapture) FindReadySourcesOwned(_ context.Context, owner, kb int64, ids []int64) ([]*model.FeedbackSourceRow, error) {
	m.calls++
	m.ids = append([]int64{}, ids...)
	if owner != 7 || kb != 5 {
		return nil, model.ErrNotFound
	}
	rows := make([]*model.FeedbackSourceRow, 0, len(ids))
	for i, id := range ids {
		body := fmt.Sprintf("事实%d。", i+1)
		rows = append(rows, &model.FeedbackSourceRow{ChunkId: id, DocId: int64(101 + i), KbId: 5, Title: "原文", Version: 1, Content: body, ChunkContent: body, Seq: 0})
	}
	return rows, nil
}
func TestFeedbackEightSourceCaptureUsesOneOwnedLookup(t *testing.T) {
	fixture := &sourceLimitCapture{}
	snapshot := &feedbackSnapshot{Version: 1, KbId: 5, Sources: sourceLimitAnalysis(8).Sources}
	if err := captureFeedbackDocuments(context.Background(), &svc.ServiceContext{AiKbFeedbackModel: fixture}, 7, snapshot); err != nil {
		t.Fatal(err)
	}
	if fixture.calls != 1 || len(fixture.ids) != 8 || len(snapshot.Documents) != 8 {
		t.Fatal("八来源核验仍须一次归属查询")
	}
	for _, doc := range snapshot.Documents {
		if doc.Status != "verified" {
			t.Fatal("实际八来源核验失败")
		}
	}
}

func TestAnalysisSourceJSONBudgetIncludesMetadataAndAllowsBoundedLargeSnippet(t *testing.T) {
	for _, tooLarge := range []bool{false, true} {
		t.Run(fmt.Sprint(tooLarge), func(t *testing.T) {
			analysis := sourceLimitAnalysis(1)
			analysis.Sources[0].Snippet = strings.Repeat("x", 8500)
			if tooLarge {
				analysis = sourceLimitAnalysis(8)
				for i := range analysis.Sources {
					analysis.Sources[i].Snippet = strings.Repeat("x", 1200)
				}
			}
			raw, _ := json.Marshal(analysis)
			input := `{"kb_id":5,"query":"目标"}`
			task := &aipb.AgentTaskInfo{Steps: []*aipb.WorkflowStepInfo{{Index: 0, Tool: "kb_analyze", Status: "succeeded", Input: input, Output: string(raw)}}}
			_, evaluationErr := evaluationAnalysis(task, 5, "目标")
			records, _ := json.Marshal([]model.WorkflowStepRecord{{Index: 0, Tool: "kb_analyze", Status: "succeeded", Input: input, Output: string(raw)}})
			_, _, _, feedbackErr := analysisFeedbackSnapshot(&model.TaskRow{StepRecords: string(records)}, 0)
			if tooLarge {
				if evaluationErr == nil || feedbackErr == nil {
					t.Fatal("八来源JSON元数据合计超过10000也须拒绝")
				}
			} else if evaluationErr != nil || feedbackErr != nil {
				t.Fatalf("完整来源JSON小于10000的8500字节片段应兼容新上限: eval=%v feedback=%v", evaluationErr, feedbackErr)
			}
		})
	}
}

func TestAnalysisReadersEnforceRawAndJSONEncodedTextLimits(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		wantError  bool
	}{
		{name: "原始正文12000边界", text: strings.Repeat("a", 12000)},
		{name: "编码正文13000边界", text: strings.Repeat("\\", 6499)},
		{name: "编码超限但原始正文未超限", text: strings.Repeat("\\", 7000), wantError: true},
		{name: "原始正文超限", text: strings.Repeat("a", 12001), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			analysis := sourceLimitAnalysis(1)
			analysis.Analysis = tc.text
			raw, _ := json.Marshal(analysis)
			if len(raw) > 24000 {
				t.Fatal("反例不能依赖整体大小上限")
			}
			input := `{"kb_id":5,"query":"目标"}`
			task := &aipb.AgentTaskInfo{Steps: []*aipb.WorkflowStepInfo{{Index: 0, Tool: "kb_analyze", Status: "succeeded", Input: input, Output: string(raw)}}}
			_, evaluationErr := evaluationAnalysis(task, 5, "目标")
			records, _ := json.Marshal([]model.WorkflowStepRecord{{Index: 0, Tool: "kb_analyze", Status: "succeeded", Input: input, Output: string(raw)}})
			_, _, _, feedbackErr := analysisFeedbackSnapshot(&model.TaskRow{StepRecords: string(records)}, 0)
			if (evaluationErr != nil) != tc.wantError || (feedbackErr != nil) != tc.wantError {
				t.Fatalf("两个读口须保持正文raw12000及JSON13000双限: eval=%v feedback=%v", evaluationErr, feedbackErr)
			}
		})
	}
}
