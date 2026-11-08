package aiservicelogic

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
	"google.golang.org/protobuf/proto"
)

type feedbackSourceFixture struct {
	model.AiKbFeedbackModel
	rows  []*model.FeedbackSourceRow
	calls int
}

func (m *feedbackSourceFixture) FindReadySourcesOwned(_ context.Context, ownerId, kbId int64, chunkIds []int64) ([]*model.FeedbackSourceRow, error) {
	m.calls++
	if ownerId != 7 || kbId != 2 || len(chunkIds) != 1 || chunkIds[0] != 9 {
		return nil, model.ErrNotFound
	}
	return m.rows, nil
}

func TestKbFeedbackCapturesNormalizedAndBudgetedSources(t *testing.T) {
	for _, raw := range []string{"  段落甲。\n\n\n 段落乙。  ", `{"first":"这是较长的规则说明，需要在预算收缩后保留有效JSON结构","second":"另一个完整字段也包含足够长的原文说明"}`} {
		chunks := chunkText(raw)
		if len(chunks) != 1 {
			t.Fatal("夹具须生成一个实际分块")
		}
		snippet := chunks[0].Content
		if strings.HasPrefix(raw, "{") {
			snippet = agenttools.CompactToolResult(snippet, 100)
		}
		if strings.Contains(raw, snippet) {
			t.Fatal("夹具必须实际触发分块规范化或JSON结构压缩")
		}
		fixture := &feedbackSourceFixture{rows: []*model.FeedbackSourceRow{{ChunkId: 9, DocId: 3, KbId: 2, Title: "原文", Version: 4, Content: raw, ChunkContent: chunks[0].Content, Seq: 0}}}
		snapshot := &feedbackSnapshot{KbId: 2, Sources: []agenttools.KbAnalysisSource{{Index: 1, KbSearchHit: agenttools.KbSearchHit{ChunkID: 9, DocID: 3, KbID: 2, DocTitle: "原文", Snippet: snippet}}}}
		if err := captureFeedbackDocuments(context.Background(), &svc.ServiceContext{AiKbFeedbackModel: fixture}, 7, snapshot); err != nil {
			t.Fatal(err)
		}
		if fixture.calls != 1 || len(snapshot.Documents) != 1 || snapshot.Documents[0].Status != "verified" || snapshot.Sources[0].Snippet != snippet {
			t.Fatal("实际分块须可核对，历史片段不能改写")
		}
		fixture.rows[0].ChunkContent = "已变更内容"
		snapshot.Documents = nil
		if err := captureFeedbackDocuments(context.Background(), &svc.ServiceContext{AiKbFeedbackModel: fixture}, 7, snapshot); err != nil || snapshot.Documents[0].Status != "stale" {
			t.Fatal("分块与当前正文不一致须标记陈旧")
		}
	}
}

type feedbackTitleSearcher struct {
	title string
	body  string
}

func (s *feedbackTitleSearcher) Search(context.Context, int64, int64, string) ([]agenttools.KbSearchHit, error) {
	return []agenttools.KbSearchHit{{ChunkID: 9, DocID: 3, KbID: 2, DocTitle: s.title, Snippet: s.body, Score: .9}}, nil
}

type feedbackTitleChat struct{}

func (*feedbackTitleChat) ChatBounded(context.Context, []*schema.Message, int) (*schema.Message, *schema.TokenUsage, error) {
	return schema.AssistantMessage(`{"answers":[{"target":"限制","state":"supported","text":"批次最多20篇。","citations":[1]}]}`, nil), &schema.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}, nil
}

func TestKbFeedbackCapturesLongSourceTitle(t *testing.T) {
	for _, title := range []string{strings.Repeat("题", 100), "A" + strings.Repeat("题", 100), strings.Repeat("🧭", 100)} {
		t.Run(string([]rune(title)[:2]), func(t *testing.T) {
			body := "批次最多20篇。"
			tool := agenttools.NewKbAnalyze(&feedbackTitleSearcher{title: title, body: body}, func(context.Context, int64, int64) error { return nil }, &feedbackTitleChat{}, 32000)
			input := `{"kb_id":2,"query":"限制"}`
			output, err := tool.Execute(agenttools.WithActor(context.Background(), 7), []byte(input))
			if err != nil {
				t.Fatal(err)
			}
			records, _ := json.Marshal([]model.WorkflowStepRecord{{Index: 0, Tool: "kb_analyze", Status: "succeeded", Input: input, Output: output}})
			snapshot, _, _, err := analysisFeedbackSnapshot(&model.TaskRow{StepRecords: string(records)}, 0)
			if err != nil || len(snapshot.Sources) != 1 {
				t.Fatalf("实际分析输出应可采集: %v", err)
			}
			originalTitle := snapshot.Sources[0].DocTitle
			if len(originalTitle) > 256 || !utf8.ValidString(originalTitle) || originalTitle == title {
				t.Fatal("夹具必须实际触发工具UTF-8标题截断")
			}
			fixture := &feedbackSourceFixture{rows: []*model.FeedbackSourceRow{{ChunkId: 9, DocId: 3, KbId: 2, Title: title, Version: 1, Content: body, ChunkContent: chunkText(body)[0].Content, Seq: 0}}}
			service := &svc.ServiceContext{AiKbFeedbackModel: fixture}
			if err := captureFeedbackDocuments(context.Background(), service, 7, snapshot); err != nil {
				t.Fatal(err)
			}
			if snapshot.Documents[0].Status != "verified" || snapshot.Documents[0].Title != title || snapshot.Sources[0].DocTitle != originalTitle {
				t.Fatal("文档须保留全标题并核对成功，历史来源标题不能改写")
			}
			fixture.rows[0].Title = "错误标题"
			snapshot.Documents = nil
			if err := captureFeedbackDocuments(context.Background(), service, 7, snapshot); err != nil || snapshot.Documents[0].Status != "stale" {
				t.Fatal("错误标题仍须拒绝核对")
			}
		})
	}
}

func TestKbFeedbackMultipleQuotesFromSameSource(t *testing.T) {
	raw, _ := json.Marshal(feedbackSnapshot{Version: 1, Sources: []agenttools.KbAnalysisSource{{Index: 1, KbSearchHit: agenttools.KbSearchHit{Snippet: "批次最多20篇，同时处理2篇"}}}})
	review := &aipb.KbFeedbackReview{Evidence: []*aipb.KbFeedbackEvidence{{SourceIndex: 1, Quote: "最多20篇"}, {SourceIndex: 1, Quote: "同时处理2篇"}}}
	if err := validateReviewEvidence(string(raw), review); err != nil {
		t.Fatalf("同片段的两个不同事实均可摘录: %v", err)
	}
	review.Evidence[1].Quote = "最多20篇"
	if err := validateReviewEvidence(string(raw), review); err == nil {
		t.Fatal("相同来源与摘录不能重复")
	}
}

func TestKbFeedbackRejectsInvalidBeforeDB(t *testing.T) {
	valid := &aipb.SetTaskFeedbackRequest{Actor: &aipb.Actor{UserId: 1}, TaskId: 1, StepIndex: 0, Feedback: "incorrect_answer"}
	for _, name := range []string{"actor", "task", "step-negative", "step-overflow", "revision", "feedback", "note-bytes", "utf8"} {
		t.Run(name, func(t *testing.T) {
			in := proto.Clone(valid).(*aipb.SetTaskFeedbackRequest)
			switch name {
			case "actor":
				in.Actor = nil
			case "task":
				in.TaskId = 0
			case "step-negative":
				in.StepIndex = -1
			case "step-overflow":
				in.StepIndex = 32
			case "revision":
				in.ExpectedRevision = -1
			case "feedback":
				in.Feedback = "unknown"
			case "note-bytes":
				in.Note = strings.Repeat("中", 334)
			case "utf8":
				in.Note = string([]byte{0xff})
			}
			_, err := NewSetTaskFeedbackLogic(context.Background(), &svc.ServiceContext{}).SetTaskFeedback(in)
			code := uint32(400)
			if name == "actor" {
				code = 403
			}
			assertWorkspaceCode(t, err, code)
		})
	}
}

func TestKbFeedbackReviewRejectsInvalidBeforeDB(t *testing.T) {
	valid := &aipb.ReviewKbFeedbackRequest{Actor: &aipb.Actor{UserId: 1}, KbId: 1, FeedbackId: 1, ExpectedRevision: 1, Status: "accepted", Review: &aipb.KbFeedbackReview{ExpectedAnswer: "资料不足", Reason: "已核对原文"}}
	for _, name := range []string{"actor", "kb", "feedback-id", "revision", "status", "review", "reason", "answer-bytes", "facts-overflow", "fact-bytes", "quote-bytes"} {
		t.Run(name, func(t *testing.T) {
			in := proto.Clone(valid).(*aipb.ReviewKbFeedbackRequest)
			switch name {
			case "actor":
				in.Actor = nil
			case "kb":
				in.KbId = 0
			case "feedback-id":
				in.FeedbackId = 0
			case "revision":
				in.ExpectedRevision = 0
			case "status":
				in.Status = "pending"
			case "review":
				in.Review = nil
			case "reason":
				in.Review.Reason = "  "
			case "answer-bytes":
				in.Review.ExpectedAnswer = strings.Repeat("中", 1334)
			case "facts-overflow":
				in.Review.RequiredFacts = make([]string, 11)
			case "fact-bytes":
				in.Review.RequiredFacts = []string{strings.Repeat("中", 67)}
			case "quote-bytes":
				in.Review.Evidence = []*aipb.KbFeedbackEvidence{{SourceIndex: 1, Quote: strings.Repeat("中", 167)}}
			}
			_, err := NewReviewKbFeedbackLogic(context.Background(), &svc.ServiceContext{}).ReviewKbFeedback(in)
			code := uint32(400)
			if name == "actor" {
				code = 403
			}
			assertWorkspaceCode(t, err, code)
		})
	}
}
