package aiservicelogic

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
	"google.golang.org/protobuf/proto"
)

type blockingWorkflowTool struct {
	name    string
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (t *blockingWorkflowTool) Name() string { return t.name }
func (*blockingWorkflowTool) Description() string {
	return "隔离测试：模拟不立即响应取消的外部工具"
}
func (*blockingWorkflowTool) Params() map[string]*schema.ParameterInfo {
	return map[string]*schema.ParameterInfo{}
}
func (t *blockingWorkflowTool) Execute(context.Context, json.RawMessage) (string, error) {
	t.calls.Add(1)
	close(t.started)
	<-t.release
	return `{"late":true}`, nil
}

func TestAIWorkspaceIntegrationAsyncIdempotencyCancelLateResult(t *testing.T) {
	h := newWorkspaceHarness(t)
	tool := &blockingWorkflowTool{name: fmt.Sprintf("blocking_%d", h.actor.UserId), started: make(chan struct{}), release: make(chan struct{})}
	agenttools.RegisterIfAbsent(tool)
	id := h.create(fmt.Sprintf(`{"steps":[{"name":"当前工具","tool":"%s","input":{}},{"name":"不得执行","tool":"get_current_time","input":{}}]}`, tool.name))
	request := &aipb.RunWorkflowRequest{Actor: h.actor, WorkflowId: id, Input: `{}`, RequestId: uuid.NewString()}
	response, err := NewRunWorkflowLogic(h.ctx, h.service).RunWorkflow(request)
	if err != nil || response.Status != "running" {
		t.Fatalf("应先返回任务ID: %v", err)
	}
	select {
	case <-tool.started:
	case <-h.ctx.Done():
		t.Fatal("工具未启动")
	}
	defer func() {
		select {
		case <-tool.release:
		default:
			close(tool.release)
		}
	}()
	replay, err := NewRunWorkflowLogic(h.ctx, h.service).RunWorkflow(request)
	if err != nil || replay.TaskId != response.TaskId || tool.calls.Load() != 1 {
		t.Fatal("相同request_id须复用任务且不重放")
	}
	changed := &aipb.RunWorkflowRequest{Actor: request.Actor, WorkflowId: request.WorkflowId, RequestId: request.RequestId, Input: `{"changed":true}`}
	_, err = NewRunWorkflowLogic(h.ctx, h.service).RunWorkflow(changed)
	assertWorkspaceCode(t, err, 400)
	foreign := &aipb.Actor{UserId: h.actor.UserId + 1}
	_, err = NewCancelTaskLogic(h.ctx, h.service).CancelTask(&aipb.CancelTaskRequest{Actor: foreign, TaskId: response.TaskId})
	assertWorkspaceCode(t, err, 404)
	if _, err = NewCancelTaskLogic(h.ctx, h.service).CancelTask(&aipb.CancelTaskRequest{Actor: h.actor, TaskId: response.TaskId}); err != nil {
		t.Fatal(err)
	}
	if cancelled := h.task(response.TaskId); cancelled.Status != "cancelled" || cancelled.NextStep != 0 || len(cancelled.Steps) != 1 || cancelled.Steps[0].Status != "cancelled" {
		t.Fatalf("取消须原子终结当前记录: %+v", cancelled)
	}
	if _, err = NewCancelTaskLogic(h.ctx, h.service).CancelTask(&aipb.CancelTaskRequest{Actor: h.actor, TaskId: response.TaskId}); err != nil {
		t.Fatal("重复取消应幂等")
	}
	close(tool.release)
	deadline := time.Now().Add(3 * time.Second)
	for {
		if _, active := workflowCancels.Load(workflowJobKey{h.service, response.TaskId}); !active {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("取消后的后台资源未释放")
		}
		time.Sleep(10 * time.Millisecond)
	}
	after := h.task(response.TaskId)
	if after.Status != "cancelled" || after.Result != "" || after.NextStep != 0 || after.Steps[0].Output != "" || tool.calls.Load() != 1 {
		t.Fatal("迟到工具输出不得提交或继续后续步骤")
	}
}

func TestAIWorkspaceIntegrationBoundInputsAndApprovalPreview(t *testing.T) {
	h := newWorkspaceHarness(t)
	definition := `{"version":2,"steps":[{"id":"first","name":"整理输入","tool":"json_formatter","input":{"text":{"$ref":"input.text"}}},{"id":"next","name":"统计结果","tool":"text_stats","input":{"text":{"$ref":"steps.first.message"}},"approval":true}]}`
	id := h.create(definition)
	response, err := NewRunWorkflowLogic(h.ctx, h.service).RunWorkflow(&aipb.RunWorkflowRequest{Actor: h.actor, WorkflowId: id, Input: `{"text":"{\"message\":\"真实内容\"}"}`, RequestId: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	paused := h.task(response.TaskId)
	if paused.Status != "pending_approval" || paused.NextStep != 1 || len(paused.Steps) != 2 || paused.Steps[1].Input != `{"text":"真实内容"}` || paused.Steps[1].StartedAt != 0 {
		t.Fatalf("审批前必须保存真实解析输入: %+v", paused)
	}
	changed := `{"steps":[{"name":"恶意替换","tool":"text_stats","input":{"text":"被修改"}}]}`
	if _, err := NewUpdateWorkflowLogic(h.ctx, h.service).UpdateWorkflow(&aipb.UpdateWorkflowRequest{Actor: h.actor, WorkflowId: id, ExpectedRevision: 1, Definition: &changed}); err != nil {
		t.Fatal(err)
	}
	if err := h.approve(response.TaskId, 1); err != nil {
		t.Fatal(err)
	}
	done := h.task(response.TaskId)
	if done.Status != "succeeded" || done.Steps[1].Input != paused.Steps[1].Input || !containsJSONField(done.Steps[1].Output, "chars", float64(4)) || done.Steps[1].CompletedAt < done.Steps[1].StartedAt {
		t.Fatalf("执行须使用审批快照: %+v", done)
	}
	_, err = NewCancelTaskLogic(h.ctx, h.service).CancelTask(&aipb.CancelTaskRequest{Actor: h.actor, TaskId: response.TaskId})
	assertWorkspaceCode(t, err, 400)
}

func containsJSONField(raw, key string, want any) bool {
	var value map[string]any
	return json.Unmarshal([]byte(raw), &value) == nil && value[key] == want
}

func TestAIWorkspaceIntegrationTaskSummaryBoundsAndStaleState(t *testing.T) {
	h := newWorkspaceHarness(t)
	id := h.create(`{"steps":[{"name":"占位","tool":"get_current_time","input":{}}]}`)
	var latest int64
	for i := 0; i < 2; i++ {
		task, err := h.service.AiWorkflowModel.CreateTaskOwned(h.ctx, id, h.actor.UserId, `{"text":"`+strings.Repeat("a", 15900)+`"}`)
		if err != nil {
			t.Fatal(err)
		}
		latest = task.Id
		records := []model.WorkflowStepRecord{{Index: 0, Id: "one", Name: "大记录", Tool: "get_current_time", Status: "succeeded", Input: `{"text":"` + strings.Repeat("a", 63000) + `"}`, Output: strings.Repeat("b", 59000)}}
		if err := h.service.AiTaskModel.AdvanceRecorded(h.ctx, task.Id, h.actor.UserId, 0, 1, "succeeded", strings.Repeat("r", 59000), "", records); err != nil {
			t.Fatal(err)
		}
	}
	response, err := NewListAgentTasksLogic(h.ctx, h.service).ListAgentTasks(&aipb.ListAgentTasksRequest{Actor: h.actor, Page: &aipb.PageRequest{Page: 1, PageSize: 1}})
	if err != nil || response.Page.Total != 2 || len(response.List) != 1 || response.List[0].Id != latest {
		t.Fatalf("列表分页/排序错误: %+v %v", response, err)
	}
	summary := response.List[0]
	if summary.Input != "" || summary.Result != "" || len(summary.Steps) != 0 || proto.Size(response) > 4096 {
		t.Fatal("列表必须是有界摘要，不携带大字段")
	}
	detail := h.task(latest)
	if len(detail.Result) != 59000 || len(detail.Steps) != 1 || len(detail.Steps[0].Input) < 63000 {
		t.Fatal("摘要不能影响详情读取")
	}
	t.Logf("同一记录完整列表载荷=%d字节，摘要列表载荷=%d字节；SQL仍为COUNT+分页两次，列表不读取input/result/snapshot/step_records", proto.Size(&aipb.ListAgentTasksResponse{List: []*aipb.AgentTaskInfo{detail}, Page: response.Page}), proto.Size(response))
	foreign, err := NewListAgentTasksLogic(h.ctx, h.service).ListAgentTasks(&aipb.ListAgentTasksRequest{Actor: &aipb.Actor{UserId: h.actor.UserId + 1}})
	if err != nil || foreign.Page.Total != 0 {
		t.Fatal("摘要仍须按用户隔离")
	}
	stale, err := h.service.AiWorkflowModel.CreateTaskOwned(h.ctx, id, h.actor.UserId, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.service.DB.ExecCtx(h.ctx, "UPDATE ai_task SET updated_at=DATE_SUB(NOW(),INTERVAL 101 SECOND) WHERE id=? AND created_by=?", stale.Id, h.actor.UserId); err != nil {
		t.Fatal(err)
	}
	read, err := NewGetAgentTaskLogic(h.ctx, h.service).GetAgentTask(&aipb.GetAgentTaskRequest{Actor: h.actor, TaskId: stale.Id})
	if err != nil || read.Task.Status != "failed" || !strings.Contains(read.Task.Error, "未知") || read.Task.NextStep != 0 {
		t.Fatal("过期执行权必须标记未知失败且不重放")
	}
}

func TestAIWorkspaceIntegrationCacheFailureKeepsCommittedExecution(t *testing.T) {
	h := newWorkspaceHarness(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
		for {
			connection, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer connection.Close()
				_ = connection.SetDeadline(time.Now().Add(time.Second))
				_, _ = bufio.NewReader(connection).ReadString('\n')
				_, _ = connection.Write([]byte("-ERR injected cache failure\r\n"))
			}()
		}
	}()
	conf := cache.CacheConf{{RedisConf: redis.RedisConf{Host: listener.Addr().String(), Type: "node", NonBlock: true}, Weight: 100}}
	faulty := &svc.ServiceContext{DB: h.service.DB, Redis: h.service.Redis, AiWorkflowModel: model.NewAiWorkflowModel(h.service.DB, conf), AiTaskModel: model.NewAiTaskModel(h.service.DB, conf)}
	workflow := h.create(`{"steps":[{"name":"确认后统计","tool":"text_stats","input":{"text":"只执行一次"},"approval":true}]}`)
	request := &aipb.RunWorkflowRequest{Actor: h.actor, WorkflowId: workflow, Input: `{}`, RequestId: uuid.NewString()}
	_, err = NewRunWorkflowLogic(h.ctx, faulty).RunWorkflow(request)
	if err != nil {
		t.Fatal("当前框架缓存节点将删除故障记录并异步补偿，任务创建仍应成功")
	}
	row, err := faulty.AiWorkflowModel.FindTaskRequestOwned(h.ctx, h.actor.UserId, request.RequestId)
	if err != nil {
		t.Fatal("SQL已提交任务必须能按原request_id核对")
	}
	wait := func(id int64) *aipb.AgentTaskInfo {
		t.Helper()
		deadline := time.Now().Add(6 * time.Second)
		for {
			result, err := NewGetAgentTaskLogic(h.ctx, faulty).GetAgentTask(&aipb.GetAgentTaskRequest{Actor: h.actor, TaskId: id})
			if err != nil {
				t.Fatal(err)
			}
			if result.Task.Status != "running" {
				return result.Task
			}
			if time.Now().After(deadline) {
				t.Fatal("缓存失败不能使已提交任务滞留running")
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	paused := wait(row.Id)
	if paused.Status != "pending_approval" || len(paused.Steps) != 1 {
		t.Fatal("创建后缓存失败仍须执行至审批断点")
	}
	replay, err := NewRunWorkflowLogic(h.ctx, faulty).RunWorkflow(request)
	if err != nil || replay.TaskId != row.Id {
		t.Fatal("缓存失败后重试不得再次创建")
	}
	zero := int64(0)
	_, err = NewApproveTaskLogic(h.ctx, faulty).ApproveTask(&aipb.ApproveTaskRequest{Actor: h.actor, TaskId: row.Id, ExpectedStep: &zero})
	if err != nil {
		t.Fatal(err)
	}
	done := wait(row.Id)
	if done.Status != "succeeded" || done.NextStep != 1 || strings.Count(done.Result, "[1/1]") != 1 {
		t.Fatal("批准缓存失败不能丢失已领取执行权或重放步骤")
	}
	_, err = NewApproveTaskLogic(h.ctx, faulty).ApproveTask(&aipb.ApproveTaskRequest{Actor: h.actor, TaskId: row.Id, ExpectedStep: &zero})
	assertWorkspaceCode(t, err, 400)
	cancelTask, err := h.service.AiWorkflowModel.CreateTaskOwned(h.ctx, workflow, h.actor.UserId, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewCancelTaskLogic(h.ctx, faulty).CancelTask(&aipb.CancelTaskRequest{Actor: h.actor, TaskId: cancelTask.Id})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled := wait(cancelTask.Id); cancelled.Status != "cancelled" {
		t.Fatal("取消缓存失败必须保留已提交终态")
	}
}

func TestAIWorkspaceIntegrationCancelCannotOverwriteStartedRecord(t *testing.T) {
	h := newWorkspaceHarness(t)
	workflow := h.create(`{"steps":[{"name":"占位","tool":"get_current_time","input":{}}]}`)
	for i := 0; i < 20; i++ {
		task, err := h.service.AiWorkflowModel.CreateTaskOwned(h.ctx, workflow, h.actor.UserId, `{}`)
		if err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		persisted := make(chan error, 1)
		cancelled := make(chan error, 1)
		record := model.WorkflowStepRecord{Index: 0, Id: "started", Name: "执行步骤", Tool: "text_stats", Input: `{"text":"实际输入必须保留"}`, Status: "running", StartedAt: 123}
		go func() {
			<-start
			persisted <- h.service.AiTaskModel.AdvanceRecorded(h.ctx, task.Id, h.actor.UserId, 0, 0, "running", "", "", []model.WorkflowStepRecord{record})
		}()
		go func() { <-start; cancelled <- h.service.AiTaskModel.CancelOwned(h.ctx, task.Id, h.actor.UserId) }()
		close(start)
		writeErr, cancelErr := <-persisted, <-cancelled
		if cancelErr != nil {
			t.Fatal(cancelErr)
		}
		if writeErr != nil && !errors.Is(writeErr, model.ErrTaskState) {
			t.Fatal(writeErr)
		}
		row, err := h.service.AiTaskModel.FindOwned(h.ctx, task.Id, h.actor.UserId)
		if err != nil || row.Status != "cancelled" {
			t.Fatal("竞争后必须保持取消终态")
		}
		records, err := model.ReadWorkflowStepRecords(row.StepRecords)
		if err != nil {
			t.Fatal(err)
		}
		if writeErr == nil && (len(records) != 1 || records[0].Input != record.Input || records[0].StartedAt != 123 || records[0].Status != "cancelled") {
			t.Fatal("取消不能用旧快照覆盖已提交的实际输入和开始时间")
		}
	}
}
