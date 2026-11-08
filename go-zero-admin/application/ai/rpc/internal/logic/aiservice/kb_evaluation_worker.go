package aiservicelogic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
)

func launchEvaluation(ctx context.Context, service *svc.ServiceContext, actor *aipb.Actor, row *model.KbEvaluationRow) {
	background, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Hour)
	background = svc.WithUsageSource(background, actor.UserId, "evaluation", row.RequestId, row.Id, -1)
	copiedActor := &aipb.Actor{UserId: actor.UserId, Username: actor.Username, DeptId: actor.DeptId, Ip: actor.Ip, UserAgent: actor.UserAgent}
	go func() {
		defer releaseEvaluationSlot()
		defer cancel()
		token := uuid.NewString()
		finishUnknown := func() {
			clean, stop := context.WithTimeout(context.WithoutCancel(background), 5*time.Second)
			defer stop()
			_, _ = service.AiKbEvaluationModel.FinishOwned(clean, row.OwnerId, row.KbId, row.Id, token, "unknown", false, "评测执行未确认，请保留原请求标识核对，不会自动重放")
		}
		defer func() {
			if recover() != nil {
				finishUnknown()
				logx.WithContext(background).Errorw("知识库评测执行异常", logx.Field("evaluation_id", row.Id))
			}
		}()
		active, err := service.AiKbEvaluationModel.ActivateOwned(background, row.OwnerId, row.KbId, row.Id, token)
		if err != nil {
			if errors.Is(err, model.ErrEvaluationCommitted) {
				finishUnknown()
			}
			return
		}
		row = active
		done := make(chan struct{})
		defer close(done)
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-background.Done():
					return
				case <-ticker.C:
					heartbeat, stop := context.WithTimeout(background, 5*time.Second)
					err := service.AiKbEvaluationModel.HeartbeatOwned(heartbeat, row.OwnerId, row.KbId, row.Id, token)
					stop()
					if err != nil {
						cancel()
						return
					}
				}
			}
		}()
		runtime := &evaluationServiceRuntime{service: service}
		if err := runEvaluationCases(background, copiedActor, row, token, service.AiKbEvaluationModel, runtime, 750*time.Millisecond); err != nil {
			finishUnknown()
			logx.WithContext(background).Errorw("知识库评测状态未确认", logx.Field("evaluation_id", row.Id))
		}
	}()
}

type evaluationCaseRuntime interface {
	Search(context.Context, *aipb.Actor, int64, string) ([]agenttools.KbSearchHit, int64, error)
	Submit(context.Context, *aipb.Actor, int64, string, string) (int64, error)
	Task(context.Context, *aipb.Actor, int64) (*aipb.AgentTaskInfo, error)
}
type evaluationServiceRuntime struct{ service *svc.ServiceContext }

func (r *evaluationServiceRuntime) Search(ctx context.Context, actor *aipb.Actor, kb int64, query string) ([]agenttools.KbSearchHit, int64, error) {
	result, err := NewSearchKbLogic(ctx, r.service).SearchKb(&aipb.SearchKbRequest{Actor: actor, KbId: kb, Query: query})
	if err != nil {
		return nil, 0, err
	}
	hits := make([]agenttools.KbSearchHit, 0, len(result.List))
	for _, hit := range result.List {
		if hit == nil {
			return nil, 0, errors.New("search result invalid")
		}
		hits = append(hits, agenttools.KbSearchHit{ChunkID: hit.ChunkId, KbID: hit.KbId, DocID: hit.DocId, DocTitle: hit.DocTitle, Snippet: hit.Snippet, Score: hit.Score})
	}
	return hits, result.ElapsedMs, nil
}
func (r *evaluationServiceRuntime) Submit(ctx context.Context, actor *aipb.Actor, kb int64, query, requestId string) (int64, error) {
	definition, _ := json.Marshal(map[string]any{"steps": []any{map[string]any{"name": "质量验收分析", "tool": "kb_analyze", "input": map[string]any{"kb_id": kb, "query": query}}}})
	input, _ := json.Marshal(map[string]any{"question": query})
	result, err := NewRunDraftLogic(ctx, r.service).RunDraft(&aipb.RunDraftRequest{Actor: actor, Name: "知识库质量验收-" + requestId, Definition: string(definition), Input: string(input), RequestId: requestId})
	if err != nil {
		if receipt, readErr := r.service.AiWorkflowModel.FindTaskRequestOwned(ctx, actor.UserId, requestId); readErr == nil {
			return receipt.Id, err
		}
		return 0, err
	}
	if result == nil || result.TaskId <= 0 {
		return 0, errors.New("task receipt invalid")
	}
	return result.TaskId, nil
}
func (r *evaluationServiceRuntime) Task(ctx context.Context, actor *aipb.Actor, id int64) (*aipb.AgentTaskInfo, error) {
	result, err := NewGetAgentTaskLogic(ctx, r.service).GetAgentTask(&aipb.GetAgentTaskRequest{Actor: actor, TaskId: id})
	if err != nil {
		return nil, err
	}
	return result.Task, nil
}

func evaluationSnapshot(row *model.KbEvaluationRow) (*evaluationDataset, []model.EvaluationObservation, error) {
	sum := sha256.Sum256([]byte(row.Snapshot))
	if hex.EncodeToString(sum[:]) != row.DatasetSHA {
		return nil, nil, errEvaluationDataset
	}
	var dataset evaluationDataset
	var items []model.EvaluationObservation
	if json.Unmarshal([]byte(row.Snapshot), &dataset) != nil || dataset.Version != 1 || json.Unmarshal([]byte(row.Observations), &items) != nil || len(items) < 1 || len(items) > 30 || int64(len(items)) != row.CaseCount {
		return nil, nil, errEvaluationDataset
	}
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.CaseId
		if !evaluationUUID(item.RequestId) {
			return nil, nil, errEvaluationDataset
		}
	}
	selected, err := selectEvaluationCases(&dataset, row.GroupKey, ids)
	if err != nil {
		return nil, nil, err
	}
	for i, item := range items {
		if selected[i].Id != item.CaseId || selected[i].Query != item.Query {
			return nil, nil, errEvaluationDataset
		}
	}
	return &dataset, items, nil
}

func runEvaluationCases(ctx context.Context, actor *aipb.Actor, row *model.KbEvaluationRow, token string, store model.AiKbEvaluationModel, runtime evaluationCaseRuntime, poll time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	finish := func(status string, verified bool, message string) error {
		_, err := store.FinishOwned(ctx, row.OwnerId, row.KbId, row.Id, token, status, verified, message)
		return err
	}
	dataset, items, err := evaluationSnapshot(row)
	if err != nil {
		return finish("failed", false, "评测快照无法识别，未提交新的分析")
	}
	corpus, err := store.ReadCorpusOwned(ctx, row.OwnerId, row.KbId)
	if err != nil {
		return finish("failed", false, "评测资料核对失败，未提交新的分析")
	}
	bindings, err := buildEvaluationBindings(dataset, row.GroupKey, row.KbId, corpus)
	if err != nil || !evaluationJSONEqual(bindings, row.Bindings) {
		return finish("failed", false, "资料版本或状态与批次快照不一致，未提交新的分析")
	}
	for index := range items {
		claimed, claimErr := store.TryStartCase(ctx, row.OwnerId, row.KbId, row.Id, int64(index), token)
		if errors.Is(claimErr, model.ErrEvaluationStopped) {
			return finish("cancelled", false, "已停止，未开始的题目不会执行")
		}
		if claimErr != nil {
			return claimErr
		}
		var latest []model.EvaluationObservation
		if json.Unmarshal([]byte(claimed.Observations), &latest) != nil || len(latest) != len(items) {
			return model.ErrEvaluationState
		}
		observed, executeErr := executeEvaluationCase(ctx, actor, row, token, int64(index), latest[index], store, runtime, poll)
		if executeErr != nil {
			return executeErr
		}
		if _, err = store.SaveCase(ctx, row.OwnerId, row.KbId, row.Id, int64(index), token, observed); err != nil {
			return err
		}
		if observed.Status == "failed" || observed.Status == "unknown" {
			return finish(observed.Status, false, "本题执行未完成，后续题目没有启动；请核对原任务和请求标识")
		}
	}
	corpus, err = store.ReadCorpusOwned(ctx, row.OwnerId, row.KbId)
	if err != nil {
		return finish("failed", false, "分析结果已保留，但结束资料核对失败，不能比较提升")
	}
	bindings, err = buildEvaluationBindings(dataset, row.GroupKey, row.KbId, corpus)
	if err != nil || !evaluationJSONEqual(bindings, row.Bindings) {
		return finish("failed", false, "分析结果已保留，但运行期间资料变化，不能比较提升")
	}
	return finish("succeeded", true, "")
}

func executeEvaluationCase(parent context.Context, actor *aipb.Actor, row *model.KbEvaluationRow, token string, index int64, observed model.EvaluationObservation, store model.AiKbEvaluationModel, runtime evaluationCaseRuntime, poll time.Duration) (model.EvaluationObservation, error) {
	ctx, cancel := context.WithTimeout(parent, 100*time.Second)
	defer cancel()
	started := time.Now()
	hits, serverMs, err := runtime.Search(ctx, actor, row.KbId, observed.Query)
	observed.SearchClientMs = time.Since(started).Milliseconds()
	observed.SearchServerMs = serverMs
	if err != nil {
		observed.Status = "failed"
		observed.Stage = "search_failed"
		observed.Error = "资料检索失败，未提交分析任务"
		return observed, nil
	}
	raw, err := json.Marshal(hits)
	if err != nil || len(hits) > 5 || serverMs < 0 {
		observed.Status = "failed"
		observed.Stage = "search_failed"
		observed.Error = "检索结果无法识别，未提交分析任务"
		return observed, nil
	}
	if hits == nil {
		raw = []byte(`[]`)
	}
	observed.List = raw
	observed.Stage = "submitting"
	if _, err = store.SaveCase(parent, row.OwnerId, row.KbId, row.Id, index, token, observed); err != nil {
		return observed, err
	}
	submitted := time.Now()
	id, err := runtime.Submit(ctx, actor, row.KbId, observed.Query, observed.RequestId)
	observed.TaskId = id
	unknown := func(message string) (model.EvaluationObservation, error) {
		observed.Status = "unknown"
		observed.Stage = "unknown"
		observed.Error = message
		observed.AnalysisElapsedMs = time.Since(submitted).Milliseconds()
		return observed, nil
	}
	if err != nil || id <= 0 {
		return unknown("分析提交结果未确认，请按本题原请求标识核对，不会自动重发")
	}
	observed.Stage = "polling"
	if _, err = store.SaveCase(parent, row.OwnerId, row.KbId, row.Id, index, token, observed); err != nil {
		return observed, err
	}
	for {
		if ctx.Err() != nil {
			return unknown("分析等待超时或执行状态未确认，请核对原任务，不会自动重发")
		}
		task, readErr := runtime.Task(ctx, actor, id)
		if readErr != nil {
			return unknown("分析任务读取失败，结果未确认，请核对原任务")
		}
		if task == nil || task.Id != id || task.RequestId != observed.RequestId || task.WorkflowId != 0 {
			return unknown("任务标识与本题快照不一致，已停止后续题")
		}
		observed.AnalysisElapsedMs = time.Since(submitted).Milliseconds()
		switch task.Status {
		case "succeeded":
			analysis, parseErr := evaluationAnalysis(task, row.KbId, observed.Query)
			if parseErr != nil {
				return unknown("分析任务输出无法识别，请核对原任务；不计为错误答案或空召回")
			}
			observed.Analysis = analysis
			observed.Status = "succeeded"
			observed.Stage = "completed"
			return observed, nil
		case "failed", "rejected", "cancelled":
			observed.Status = "failed"
			observed.Stage = "analysis_failed"
			observed.Error = "分析任务未成功，请打开原任务核对原因"
			return observed, nil
		case "pending", "running":
		default:
			return unknown("任务状态不符合本次评测约定，已停止后续题")
		}
		if poll > 0 {
			timer := time.NewTimer(poll)
			select {
			case <-ctx.Done():
				timer.Stop()
				return unknown("分析等待超时或已中断，请核对原任务，不会自动重发")
			case <-timer.C:
			}
		}
	}
}

func evaluationAnalysis(task *aipb.AgentTaskInfo, kb int64, query string) (json.RawMessage, error) {
	if len(task.Steps) != 1 || task.Steps[0] == nil {
		return nil, model.ErrEvaluationState
	}
	step := task.Steps[0]
	var input struct {
		KbId  int64  `json:"kb_id"`
		Query string `json:"query"`
	}
	var analysis agenttools.KbAnalysisResult
	if step.Index != 0 || step.Tool != "kb_analyze" || step.Status != "succeeded" || json.Unmarshal([]byte(step.Input), &input) != nil || input.KbId != kb || input.Query != query || len(step.Output) > 24000 || !utf8.ValidString(step.Output) || json.Unmarshal([]byte(step.Output), &analysis) != nil {
		return nil, model.ErrEvaluationState
	}
	sourceJSON, err := json.Marshal(analysis.Sources)
	textJSON, _ := json.Marshal(analysis.Analysis)
	if err != nil || len(sourceJSON) > 10000 || len(textJSON) > 13000 || analysis.Version != 1 || analysis.Type != "kb_analysis" || strings.TrimSpace(analysis.Analysis) == "" || len(analysis.Analysis) > 12000 || len(analysis.Sources) > 8 || analysis.Usage.Scope != "chat" || analysis.Usage.PromptTokens < 0 || analysis.Usage.CompletionTokens < 0 || analysis.Usage.TotalTokens < 0 {
		return nil, model.ErrEvaluationState
	}
	switch analysis.Status {
	case "completed", "truncated":
		if len(analysis.Sources) == 0 {
			return nil, model.ErrEvaluationState
		}
	case "no_matches":
		if len(analysis.Sources) > 0 || analysis.Usage.Complete || analysis.Usage.TotalTokens != 0 || analysis.Usage.PromptTokens != 0 || analysis.Usage.CompletionTokens != 0 {
			return nil, model.ErrEvaluationState
		}
	default:
		return nil, model.ErrEvaluationState
	}
	if analysis.Usage.Complete && (analysis.Usage.PromptTokens <= 0 || analysis.Usage.CompletionTokens <= 0 || analysis.Usage.TotalTokens != analysis.Usage.PromptTokens+analysis.Usage.CompletionTokens) {
		return nil, model.ErrEvaluationState
	}
	chunks := make(map[int64]bool, len(analysis.Sources))
	for i, source := range analysis.Sources {
		if source.Index != i+1 || source.KbID != kb || source.DocID <= 0 || source.ChunkID <= 0 || chunks[source.ChunkID] || strings.TrimSpace(source.Snippet) == "" || !utf8.ValidString(source.Snippet) || !utf8.ValidString(source.DocTitle) || math.IsNaN(float64(source.Score)) || math.IsInf(float64(source.Score), 0) {
			return nil, model.ErrEvaluationState
		}
		chunks[source.ChunkID] = true
	}
	return json.RawMessage(step.Output), nil
}
