package aiservicelogic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/xerr"
)

type feedbackDocument struct {
	DocId      int64  `json:"doc_id"`
	KbId       int64  `json:"kb_id"`
	Title      string `json:"title"`
	Version    int64  `json:"version"`
	ContentSHA string `json:"content_sha256"`
	Status     string `json:"status"`
}

type feedbackSnapshot struct {
	Version    int                           `json:"version"`
	KbId       int64                         `json:"kb_id"`
	Query      string                        `json:"query"`
	StepInput  string                        `json:"step_input"`
	StepOutput string                        `json:"step_output"`
	Sources    []agenttools.KbAnalysisSource `json:"sources"`
	Documents  []feedbackDocument            `json:"documents"`
}

func feedbackError(err error) error {
	if errors.Is(err, model.ErrFeedbackConflict) {
		return xerr.NewErrCodeMsg(409, "评价或复核已更新，请刷新核对后重新提交")
	}
	if errors.Is(err, model.ErrFeedbackCommitted) {
		return xerr.NewErrCodeMsg(503, "评价已保存，缓存刷新失败，请刷新核对，不要重复创建")
	}
	return aiModelError(err)
}

func validFeedbackText(text string, maximum int, required bool) bool {
	return utf8.ValidString(text) && len(text) <= maximum && (!required || strings.TrimSpace(text) != "")
}

func validateReviewShape(review *aipb.KbFeedbackReview, status string) error {
	bad := xerr.NewErrCodeMsg(400, "复核内容无效，请核对预期答案、事实检查项、原文摘录及字节上限")
	if review == nil || (status != "accepted" && status != "dismissed") || !validFeedbackText(review.Reason, 1000, true) || !validFeedbackText(review.ExpectedAnswer, 4000, status == "accepted") || len(review.RequiredFacts) > 10 || len(review.Evidence) > 5 {
		return bad
	}
	seen := make(map[string]bool)
	for _, fact := range review.RequiredFacts {
		if !validFeedbackText(fact, 200, true) || seen[fact] {
			return bad
		}
		seen[fact] = true
	}
	for _, evidence := range review.Evidence {
		if evidence == nil || evidence.SourceIndex < 1 || evidence.SourceIndex > 8 || !validFeedbackText(evidence.Quote, 500, true) {
			return bad
		}
	}
	if status == "accepted" {
		if review.Answerable && (len(review.RequiredFacts) == 0 || len(review.Evidence) == 0) {
			return bad
		}
		if !review.Answerable && (len(review.RequiredFacts) != 0 || len(review.Evidence) != 0) {
			return bad
		}
	}
	return nil
}

func analysisFeedbackSnapshot(task *model.TaskRow, index int64) (*feedbackSnapshot, model.WorkflowStepRecord, string, error) {
	bad := xerr.NewErrCodeMsg(400, "只能评价已有执行记录的成功知识库分析步骤")
	records, err := model.ReadWorkflowStepRecords(task.StepRecords)
	if err != nil || index < 0 || index >= int64(len(records)) {
		return nil, model.WorkflowStepRecord{}, "", bad
	}
	record := records[index]
	var input struct {
		KbId  int64  `json:"kb_id"`
		Query string `json:"query"`
	}
	if record.Tool != "kb_analyze" || record.Status != "succeeded" || json.Unmarshal([]byte(record.Input), &input) != nil || input.KbId <= 0 || !validFeedbackText(input.Query, 2000, true) {
		return nil, record, "", bad
	}
	var result agenttools.KbAnalysisResult
	if !validFeedbackText(record.Output, 24000, true) || json.Unmarshal([]byte(record.Output), &result) != nil || result.Version != 1 || result.Type != "kb_analysis" || !validFeedbackText(result.Analysis, 12000, true) || len(result.Sources) > 8 {
		return nil, record, "", bad
	}
	sourceJSON, err := json.Marshal(result.Sources)
	textJSON, _ := json.Marshal(result.Analysis)
	if err != nil || len(sourceJSON) > 10000 || len(textJSON) > 13000 {
		return nil, record, "", bad
	}
	if result.Status != "completed" && result.Status != "no_matches" && result.Status != "truncated" {
		return nil, record, "", bad
	}
	if (result.Status == "no_matches") != (len(result.Sources) == 0) {
		return nil, record, "", bad
	}
	for i, source := range result.Sources {
		if source.Index != i+1 || source.KbID != input.KbId || source.DocID <= 0 || source.ChunkID <= 0 || !validFeedbackText(source.Snippet, 10000, true) || math.IsNaN(float64(source.Score)) || math.IsInf(float64(source.Score), 0) {
			return nil, record, "", bad
		}
	}
	return &feedbackSnapshot{Version: 1, KbId: input.KbId, Query: input.Query, StepInput: record.Input, StepOutput: record.Output, Sources: result.Sources, Documents: []feedbackDocument{}}, record, result.Analysis, nil
}

// 只标记反馈采集时可核对的正文，不推断模型调用时全库的版本。
func captureFeedbackDocuments(ctx context.Context, service *svc.ServiceContext, ownerId int64, snapshot *feedbackSnapshot) error {
	ids := make([]int64, 0, len(snapshot.Sources))
	for _, source := range snapshot.Sources {
		ids = append(ids, source.ChunkID)
	}
	rows, err := service.AiKbFeedbackModel.FindReadySourcesOwned(ctx, ownerId, snapshot.KbId, ids)
	if err != nil {
		return feedbackError(err)
	}
	byChunk := make(map[int64]*model.FeedbackSourceRow, len(rows))
	for _, row := range rows {
		byChunk[row.ChunkId] = row
	}
	seen := make(map[int64]bool)
	for _, source := range snapshot.Sources {
		if seen[source.DocID] {
			continue
		}
		seen[source.DocID] = true
		fingerprint := feedbackDocument{DocId: source.DocID, KbId: snapshot.KbId, Title: source.DocTitle, Status: "unavailable"}
		doc := byChunk[source.ChunkID]
		if doc != nil && doc.DocId == source.DocID && doc.KbId == snapshot.KbId {
			fingerprint.Title = doc.Title
			fingerprint.Version = doc.Version
			sum := sha256.Sum256([]byte(doc.Content))
			fingerprint.ContentSHA = hex.EncodeToString(sum[:])
			fingerprint.Status = "verified"
			chunks := chunkText(doc.Content)
			for _, evidence := range snapshot.Sources {
				current := byChunk[evidence.ChunkID]
				if evidence.DocID == source.DocID && (current == nil || current.DocId != doc.DocId || feedbackSourceTitle(current.Title) != evidence.DocTitle || current.Version != doc.Version || current.Content != doc.Content || current.Seq < 0 || current.Seq >= int64(len(chunks)) || chunks[current.Seq].Content != current.ChunkContent) {
					fingerprint.Status = "stale"
				}
			}
		}
		snapshot.Documents = append(snapshot.Documents, fingerprint)
	}
	return nil
}

// 分析工具将来源标题限为256个UTF-8字节；文档指纹仍保留数据库完整标题。
func feedbackSourceTitle(title string) string {
	if len(title) <= 256 {
		return title
	}
	title = title[:256]
	for !utf8.ValidString(title) {
		title = title[:len(title)-1]
	}
	return title
}

func validateReviewEvidence(snapshotRaw string, review *aipb.KbFeedbackReview) error {
	var snapshot feedbackSnapshot
	if json.Unmarshal([]byte(snapshotRaw), &snapshot) != nil || snapshot.Version != 1 {
		return xerr.NewErrCodeMsg(400, "历史来源快照无效")
	}
	seen := make(map[struct {
		index int64
		quote string
	}]bool)
	for _, evidence := range review.Evidence {
		index := evidence.SourceIndex
		key := struct {
			index int64
			quote string
		}{index, evidence.Quote}
		if index < 1 || index > int64(len(snapshot.Sources)) || seen[key] || !strings.Contains(snapshot.Sources[index-1].Snippet, evidence.Quote) {
			return xerr.NewErrCodeMsg(400, "摘录须逐字来自所选历史片段，且不能重复")
		}
		seen[key] = true
	}
	return nil
}

func kbFeedbackProto(row *model.KbFeedbackRow) (*aipb.KbFeedbackInfo, error) {
	var review *aipb.KbFeedbackReview
	if row.Review != "" {
		review = &aipb.KbFeedbackReview{}
		if json.Unmarshal([]byte(row.Review), review) != nil {
			return nil, xerr.NewErrCodeMsg(500, "复核记录格式异常")
		}
		if review.RequiredFacts == nil {
			review.RequiredFacts = []string{}
		}
		if review.Evidence == nil {
			review.Evidence = []*aipb.KbFeedbackEvidence{}
		}
	}
	return &aipb.KbFeedbackInfo{Id: row.Id, KbId: row.KbId, TaskId: row.TaskId, StepIndex: row.StepIndex, Feedback: row.Feedback, Note: row.Note, Status: row.Status, Revision: row.Revision, Question: row.Question, Answer: row.Answer, Snapshot: row.Snapshot, Review: review, ReviewedAt: row.ReviewedAt, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}, nil
}

func kbFeedbackProtos(rows []*model.KbFeedbackRow) ([]*aipb.KbFeedbackInfo, error) {
	list := make([]*aipb.KbFeedbackInfo, 0, len(rows))
	for _, row := range rows {
		item, err := kbFeedbackProto(row)
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, nil
}
