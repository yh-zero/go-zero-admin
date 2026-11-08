package aiservicelogic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
)

type evaluationMemoryStore struct {
	model.AiKbEvaluationModel
	row          *model.KbEvaluationRow
	corpus       []model.EvaluationCorpusDocument
	corpusReads  int
	changedAfter bool
	stopAfter    int64
}

func (s *evaluationMemoryStore) ReadCorpusOwned(context.Context, int64, int64) ([]model.EvaluationCorpusDocument, error) {
	s.corpusReads++
	rows := append([]model.EvaluationCorpusDocument(nil), s.corpus...)
	if s.changedAfter && s.corpusReads > 1 {
		rows[0].Version++
	}
	return rows, nil
}
func (s *evaluationMemoryStore) TryStartCase(_ context.Context, _, _, _, index int64, _ string) (*model.KbEvaluationRow, error) {
	if s.row.StopRequested {
		return nil, model.ErrEvaluationStopped
	}
	var items []model.EvaluationObservation
	_ = json.Unmarshal([]byte(s.row.Observations), &items)
	if items[index].Status != "waiting" {
		return nil, model.ErrEvaluationState
	}
	items[index].Status = "running"
	items[index].Stage = "search"
	raw, _ := json.Marshal(items)
	s.row.Observations = string(raw)
	return s.row, nil
}
func (s *evaluationMemoryStore) SaveCase(_ context.Context, _, _, _, index int64, _ string, next model.EvaluationObservation) (*model.KbEvaluationRow, error) {
	var items []model.EvaluationObservation
	_ = json.Unmarshal([]byte(s.row.Observations), &items)
	items[index] = next
	raw, _ := json.Marshal(items)
	s.row.Observations = string(raw)
	s.row.CompletedCount = 0
	s.row.FailedCount = 0
	for _, item := range items {
		if item.Status == "succeeded" {
			s.row.CompletedCount++
		}
		if item.Status == "failed" || item.Status == "unknown" {
			s.row.FailedCount++
		}
	}
	if s.stopAfter > 0 && s.row.CompletedCount >= s.stopAfter {
		s.row.StopRequested = true
	}
	return s.row, nil
}
func (s *evaluationMemoryStore) FinishOwned(_ context.Context, _, _, _ int64, _ string, status string, verified bool, message string) (*model.KbEvaluationRow, error) {
	s.row.Status = status
	s.row.CorpusVerified = verified
	s.row.Error = message
	return s.row, nil
}

type evaluationMemoryRuntime struct {
	searches, submissions, reads  int
	searchErr, submitErr, readErr error
	terminal                      string
	queries, requests             []string
	corrupt                       bool
}

func (r *evaluationMemoryRuntime) Search(_ context.Context, actor *aipb.Actor, kb int64, query string) ([]agenttools.KbSearchHit, int64, error) {
	r.searches++
	if actor.UserId != 7 || kb != 5 {
		panic("scope changed")
	}
	if r.searchErr != nil {
		return nil, 0, r.searchErr
	}
	return []agenttools.KbSearchHit{{KbID: kb, DocID: 10, ChunkID: 100, DocTitle: "指南", Snippet: "预算为五万元。", Score: .5}}, 1, nil
}
func (r *evaluationMemoryRuntime) Submit(_ context.Context, _ *aipb.Actor, _ int64, query, request string) (int64, error) {
	r.submissions++
	r.queries = append(r.queries, query)
	r.requests = append(r.requests, request)
	if r.submitErr != nil {
		return 0, r.submitErr
	}
	return int64(1000 + r.submissions), nil
}
func (r *evaluationMemoryRuntime) Task(_ context.Context, _ *aipb.Actor, id int64) (*aipb.AgentTaskInfo, error) {
	r.reads++
	if r.readErr != nil {
		return nil, r.readErr
	}
	index := int(id) - 1001
	status := r.terminal
	if status == "" {
		status = "succeeded"
	}
	input, _ := json.Marshal(map[string]any{"kb_id": 5, "query": r.queries[index]})
	analysis := agenttools.KbAnalysisResult{Version: 1, Type: "kb_analysis", Status: "completed", Analysis: "预算为五万元。[1]", Sources: []agenttools.KbAnalysisSource{{Index: 1, KbSearchHit: agenttools.KbSearchHit{KbID: 5, DocID: 10, ChunkID: 100, DocTitle: "指南", Snippet: "预算为五万元。", Score: .5}}}, Usage: agenttools.KbAnalysisUsage{Scope: "chat", PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15, Complete: true}}
	output, _ := json.Marshal(analysis)
	if r.corrupt {
		output = []byte(`{"corrupt":true}`)
	}
	return &aipb.AgentTaskInfo{Id: id, RequestId: r.requests[index], Status: status, Steps: []*aipb.WorkflowStepInfo{{Index: 0, Tool: "kb_analyze", Input: string(input), Status: "succeeded", Output: string(output)}}}, nil
}

func evaluationWorkerFixture(t *testing.T) (*evaluationMemoryStore, *evaluationMemoryRuntime) {
	t.Helper()
	dataset := evaluationDataset{Version: 1, Documents: []evaluationDocument{{Key: "guide", Kb: "ops", Source: "guide.md", Title: "指南", SHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}, Cases: []evaluationCase{{Id: "one", Kb: "ops", Query: "预算是多少？"}, {Id: "two", Kb: "ops", Query: "请再说明预算。"}}}
	raw, _ := json.Marshal(dataset)
	sum := sha256.Sum256(raw)
	dataset.Raw = string(raw)
	dataset.SHA = hex.EncodeToString(sum[:])
	corpus := []model.EvaluationCorpusDocument{{Id: 10, KbId: 5, Title: "指南", Version: 1, Status: "ready", EmbeddingModel: "fixture", ContentSHA: dataset.Documents[0].SHA}}
	bindings, err := buildEvaluationBindings(&dataset, "ops", 5, corpus)
	if err != nil {
		t.Fatal(err)
	}
	items := []model.EvaluationObservation{}
	for _, c := range dataset.Cases {
		items = append(items, model.EvaluationObservation{CaseId: c.Id, Query: c.Query, RequestId: uuid.NewString(), Status: "waiting", Stage: "waiting", List: json.RawMessage(`[]`)})
	}
	observations, _ := json.Marshal(items)
	return &evaluationMemoryStore{row: &model.KbEvaluationRow{Id: 1, OwnerId: 7, KbId: 5, GroupKey: "ops", Status: "running", CaseCount: 2, Snapshot: string(raw), DatasetSHA: dataset.SHA, Bindings: bindings, Observations: string(observations)}, corpus: corpus}, &evaluationMemoryRuntime{}
}
func TestEvaluationWorkerRunsPersistedCasesAndVerifiesCorpus(t *testing.T) {
	store, runtime := evaluationWorkerFixture(t)
	if err := runEvaluationCases(context.Background(), &aipb.Actor{UserId: 7}, store.row, "lease", store, runtime, 0); err != nil {
		t.Fatal(err)
	}
	if store.row.Status != "succeeded" || !store.row.CorpusVerified || store.row.CompletedCount != 2 || runtime.searches != 2 || runtime.submissions != 2 || runtime.reads != 2 || store.corpusReads != 2 {
		t.Fatalf("完整批次应串行且核对前后语料: status=%s search=%d submit=%d read=%d corpus=%d", store.row.Status, runtime.searches, runtime.submissions, runtime.reads, store.corpusReads)
	}
	if runtime.queries[0] != "预算是多少？" || runtime.queries[1] != "请再说明预算。" || runtime.requests[0] == runtime.requests[1] {
		t.Fatal("只发送实际问题并使用各自已持久化UUID")
	}
}
func TestEvaluationWorkerStopsAfterClaimedCase(t *testing.T) {
	store, runtime := evaluationWorkerFixture(t)
	store.stopAfter = 1
	if err := runEvaluationCases(context.Background(), &aipb.Actor{UserId: 7}, store.row, "lease", store, runtime, 0); err != nil {
		t.Fatal(err)
	}
	if store.row.Status != "cancelled" || runtime.submissions != 1 || store.row.CompletedCount != 1 {
		t.Fatal("停止须保留当前结果，且不得提交后续")
	}
}
func TestEvaluationWorkerFailureDoesNotReplay(t *testing.T) {
	for _, tc := range []struct {
		name, status         string
		setup                func(*evaluationMemoryRuntime)
		search, submit, read int
	}{
		{"search", "failed", func(r *evaluationMemoryRuntime) { r.searchErr = errors.New("fixture") }, 1, 0, 0},
		{"submit", "unknown", func(r *evaluationMemoryRuntime) { r.submitErr = errors.New("fixture") }, 1, 1, 0},
		{"read", "unknown", func(r *evaluationMemoryRuntime) { r.readErr = errors.New("fixture") }, 1, 1, 1},
		{"task_failed", "failed", func(r *evaluationMemoryRuntime) { r.terminal = "failed" }, 1, 1, 1},
		{"invalid_output", "unknown", func(r *evaluationMemoryRuntime) { r.corrupt = true }, 1, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, runtime := evaluationWorkerFixture(t)
			tc.setup(runtime)
			if err := runEvaluationCases(context.Background(), &aipb.Actor{UserId: 7}, store.row, "lease", store, runtime, 0); err != nil {
				t.Fatal(err)
			}
			if store.row.Status != tc.status || store.row.CompletedCount != 0 || store.row.FailedCount != 1 || runtime.searches != tc.search || runtime.submissions != tc.submit || runtime.reads != tc.read {
				t.Fatalf("失败应停止且不重放: status=%s counts=%d/%d/%d", store.row.Status, runtime.searches, runtime.submissions, runtime.reads)
			}
		})
	}
}
func TestEvaluationWorkerCorpusChangeDisablesComparison(t *testing.T) {
	store, runtime := evaluationWorkerFixture(t)
	store.changedAfter = true
	if err := runEvaluationCases(context.Background(), &aipb.Actor{UserId: 7}, store.row, "lease", store, runtime, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if store.row.Status != "failed" || store.row.CorpusVerified || store.row.CompletedCount != 2 {
		t.Fatal("语料变化不能抹掉已知结果或当作可比较成功")
	}
}
