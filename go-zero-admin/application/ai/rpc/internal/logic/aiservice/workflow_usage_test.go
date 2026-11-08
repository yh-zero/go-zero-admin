package aiservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/llm"
)

type usageTaskStore struct {
	model.AiTaskModel
	row model.TaskRow
}

type delayedUsageChat struct {
	entered chan struct{}
	release chan struct{}
}

func (c *delayedUsageChat) ChatBounded(context.Context, []*schema.Message, int) (*schema.Message, *schema.TokenUsage, error) {
	close(c.entered)
	<-c.release
	return nil, &schema.TokenUsage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}, errors.New("PRIVATE_LATE_PROVIDER_ERROR")
}

func TestAIWorkspaceIntegrationCancellationKeepsLateKnownUsage(t *testing.T) {
	h := newWorkspaceHarness(t)
	chat := &delayedUsageChat{entered: make(chan struct{}), release: make(chan struct{})}
	name := "test_cancel_usage_" + uuid.NewString()
	agenttools.RegisterIfAbsent(&workflowAnalysisTool{name: name, Tool: agenttools.NewKbAnalyze(usageAnalysisSearcher{}, func(context.Context, int64, int64) error { return nil }, chat, 32000)})
	workflow := h.create(fmt.Sprintf(`{"steps":[{"name":"分析","tool":"%s","input":{"kb_id":3,"query":"预算"}},{"name":"后续","tool":"get_current_time","input":{}}]}`, name))
	receipt, err := NewRunWorkflowLogic(h.ctx, h.service).RunWorkflow(&aipb.RunWorkflowRequest{Actor: h.actor, WorkflowId: workflow, Input: "{}", RequestId: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-chat.entered:
	case <-time.After(time.Second):
		t.Fatal("本地模型调用未开始")
	}
	if _, err := NewCancelTaskLogic(h.ctx, h.service).CancelTask(&aipb.CancelTaskRequest{Actor: h.actor, TaskId: receipt.TaskId}); err != nil {
		close(chat.release)
		t.Fatal(err)
	}
	close(chat.release)
	deadline := time.Now().Add(2 * time.Second)
	for {
		row := h.task(receipt.TaskId)
		if len(row.Steps) == 1 && row.Steps[0].Usage != nil {
			if row.Status != "cancelled" || row.NextStep != 0 || row.Result != "" || row.Steps[0].Status != "cancelled" || row.Steps[0].Output != "" || row.Steps[0].Usage.TotalTokens != 15 || row.Steps[0].Usage.Complete {
				t.Fatal("迟到计量只能补回15 tokens，不能恢复取消任务或答案")
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("取消先提交后，已知15 tokens在任务详情中丢失")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func (s *usageTaskStore) AdvanceRecorded(_ context.Context, id, owner, expected, next int64, status, result, message string, records []model.WorkflowStepRecord) error {
	if id != s.row.Id || owner != s.row.CreatedBy || expected != s.row.NextStep || s.row.Status != "running" {
		return model.ErrTaskState
	}
	raw, err := json.Marshal(records)
	if err != nil {
		return err
	}
	s.row.Status, s.row.NextStep, s.row.Result, s.row.Error, s.row.StepRecords = status, next, result, message, string(raw)
	return nil
}

func (s *usageTaskStore) FindOwned(_ context.Context, id, owner int64) (*model.TaskRow, error) {
	if id != s.row.Id || owner != s.row.CreatedBy {
		return nil, model.ErrNotFound
	}
	copy := s.row
	return &copy, nil
}

type usageAnalysisSearcher struct{ empty bool }

func (s usageAnalysisSearcher) Search(context.Context, int64, int64, string) ([]agenttools.KbSearchHit, error) {
	if s.empty {
		return nil, nil
	}
	return []agenttools.KbSearchHit{{KbID: 3, DocID: 4, ChunkID: 5, DocTitle: "本地资料", Snippet: "预算五万元。", Score: .9}}, nil
}

func TestWorkflowSuccessfulAnalysisUsageDistinguishesNoModelCall(t *testing.T) {
	for _, noMatches := range []bool{false, true} {
		t.Run(fmt.Sprintf("noMatches=%v", noMatches), func(t *testing.T) {
			name := "test_usage_success_" + uuid.NewString()
			tool := &workflowAnalysisTool{name: name, Tool: agenttools.NewKbAnalyze(usageAnalysisSearcher{empty: noMatches}, func(context.Context, int64, int64) error { return nil }, usageAnalysisChat{response: `{"answers":[{"target":"预算","state":"supported","text":"预算五万元。","citations":[1]}]}`}, 32000)}
			agenttools.RegisterIfAbsent(tool)
			store := &usageTaskStore{row: model.TaskRow{Id: 11, CreatedBy: 7, Status: "running", Input: "{}", StepRecords: "[]", WorkflowSnapshot: fmt.Sprintf(`{"steps":[{"name":"分析","tool":"%s","input":{"kb_id":3,"query":"预算"}}]}`, name)}}
			service := &svc.ServiceContext{AiTaskModel: store}
			task := store.row
			if status, err := executeWorkflow(context.Background(), service, &task, false); err != nil || status != "succeeded" {
				t.Fatalf("合法分析未成功: status=%s err=%v", status, err)
			}
			detail, err := NewGetAgentTaskLogic(context.Background(), service).GetAgentTask(&aipb.GetAgentTaskRequest{Actor: &aipb.Actor{UserId: 7}, TaskId: 11})
			if err != nil || len(detail.GetTask().GetSteps()) != 1 {
				t.Fatalf("任务步骤不可读: %v", err)
			}
			usage := detail.Task.Steps[0].GetUsage()
			if noMatches {
				if usage != nil {
					t.Fatal("无匹配未调用模型，不能填造步骤用量")
				}
			} else if usage == nil || usage.Scope != "chat" || usage.PromptTokens != 100 || usage.CompletionTokens != 20 || usage.TotalTokens != 120 || !usage.Complete {
				t.Fatalf("实际成功模型用量应与分析结果一起返回: %v", usage)
			}
		})
	}
}

type usageAnalysisChat struct {
	response string
	err      error
}

func (c usageAnalysisChat) ChatBounded(context.Context, []*schema.Message, int) (*schema.Message, *schema.TokenUsage, error) {
	if c.err != nil {
		return nil, nil, c.err
	}
	return schema.AssistantMessage(c.response, nil), &schema.TokenUsage{PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120}, nil
}

func TestWorkflowFailedAnalysisUsageSurvivesTaskDetail(t *testing.T) {
	for _, tc := range []struct {
		name     string
		chatErr  error
		ownedErr error
		want     *agenttools.KbAnalysisUsage
	}{
		{name: "回复被拒仍有已知用量", want: &agenttools.KbAnalysisUsage{Scope: "chat", PromptTokens: 100, CompletionTokens: 20, TotalTokens: 120, Complete: true}},
		{name: "调用失败保持未知", chatErr: errors.New("PRIVATE_VENDOR_DETAIL"), want: &agenttools.KbAnalysisUsage{Scope: "chat"}},
		{name: "调用前拒绝不填造", ownedErr: errors.New("not owned")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			name := "test_usage_" + uuid.NewString()
			tool := &workflowAnalysisTool{name: name, Tool: agenttools.NewKbAnalyze(usageAnalysisSearcher{}, func(context.Context, int64, int64) error { return tc.ownedErr }, usageAnalysisChat{response: "PRIVATE_MODEL_BODY", err: tc.chatErr}, 32000)}
			agenttools.RegisterIfAbsent(tool)
			store := &usageTaskStore{row: model.TaskRow{Id: 10, CreatedBy: 7, Status: "running", Input: "{}", StepRecords: "[]", WorkflowSnapshot: fmt.Sprintf(`{"steps":[{"name":"分析","tool":"%s","input":{"kb_id":3,"query":"预算"}}]}`, name)}}
			service := &svc.ServiceContext{AiTaskModel: store}
			task := store.row
			status, err := executeWorkflow(context.Background(), service, &task, false)
			if err != nil || status != "failed" {
				t.Fatalf("分析必须维持失败状态: status=%s err=%v", status, err)
			}
			detail, err := NewGetAgentTaskLogic(context.Background(), service).GetAgentTask(&aipb.GetAgentTaskRequest{Actor: &aipb.Actor{UserId: 7}, TaskId: 10})
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(detail.Task)
			var decoded struct {
				Steps []struct {
					Status string                      `json:"status"`
					Output string                      `json:"output"`
					Usage  *agenttools.KbAnalysisUsage `json:"usage"`
				} `json:"steps"`
			}
			if json.Unmarshal(raw, &decoded) != nil || len(decoded.Steps) != 1 || decoded.Steps[0].Status != "failed" || decoded.Steps[0].Output != "" || detail.Task.Result != "" || strings.Contains(string(raw), "PRIVATE_") {
				t.Fatal("失败详情必须拒绝正文且不泄露供应商信息")
			}
			got := decoded.Steps[0].Usage
			if tc.want == nil && got != nil || tc.want != nil && (got == nil || *got != *tc.want) {
				t.Fatalf("步骤计量必须随失败状态持久化并经详情可读: got=%+v want=%+v", got, tc.want)
			}
		})
	}
}

func TestAIWorkspaceIntegrationRejectedAnalysisKeepsKnownUsage(t *testing.T) {
	h := newWorkspaceHarness(t)
	name := "test_usage_persist_" + uuid.NewString()
	agenttools.RegisterIfAbsent(&workflowAnalysisTool{name: name, Tool: agenttools.NewKbAnalyze(usageAnalysisSearcher{}, func(context.Context, int64, int64) error { return nil }, usageAnalysisChat{response: "PRIVATE_MODEL_BODY"}, 32000)})
	id := h.run(h.create(fmt.Sprintf(`{"steps":[{"name":"分析","tool":"%s","input":{"kb_id":3,"query":"预算"}}]}`, name)))
	failed := h.task(id)
	if failed.Status != "failed" || failed.Result != "" || len(failed.Steps) != 1 || failed.Steps[0].Output != "" || strings.Contains(failed.Error, "PRIVATE_") {
		t.Fatal("拒绝的模型正文不得进入任务或步骤结果")
	}
	usage := failed.Steps[0].GetUsage()
	if usage == nil || usage.Scope != "chat" || usage.PromptTokens != 100 || usage.CompletionTokens != 20 || usage.TotalTokens != 120 || !usage.Complete {
		t.Fatalf("失败步骤须从数据库读回实际已知用量: %v", usage)
	}
}

func TestAIWorkspaceIntegrationSDKRejectedReplyKeepsOriginalStepUsage(t *testing.T) {
	h := newWorkspaceHarness(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	}))
	defer server.Close()
	client, err := llm.New(llm.Config{BaseURL: server.URL, APIKey: "local-test-only", Model: "local-test", Timeout: 2})
	if err != nil {
		t.Fatal(err)
	}
	name := "test_sdk_usage_" + uuid.NewString()
	agenttools.RegisterIfAbsent(&workflowAnalysisTool{name: name, Tool: agenttools.NewKbAnalyze(usageAnalysisSearcher{}, func(context.Context, int64, int64) error { return nil }, client, 32000)})
	id := h.run(h.create(fmt.Sprintf(`{"steps":[{"name":"分析","tool":"%s","input":{"kb_id":3,"query":"预算"}}]}`, name)))
	task := h.task(id)
	if task.Status != "failed" || task.Result != "" || len(task.Steps) != 1 || task.Steps[0].Output != "" || task.Steps[0].Usage == nil || task.Steps[0].Usage.TotalTokens != 15 || task.Steps[0].Usage.Complete {
		t.Fatal("SDK拒绝choices也必须把15个已知tokens保存到原失败步骤，不展示正文")
	}
}
