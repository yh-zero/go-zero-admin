package aiservicelogic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type workspaceHarness struct {
	t       *testing.T
	ctx     context.Context
	service *svc.ServiceContext
	actor   *aipb.Actor
}

func workspaceEnvironment(t *testing.T) (string, string) {
	t.Helper()
	dsn, address := os.Getenv("GOZERO_AI_TEST_DSN"), os.Getenv("GOZERO_AI_TEST_REDIS")
	if dsn == "" || address == "" {
		t.Skip("未设置 GOZERO_AI_TEST_DSN / GOZERO_AI_TEST_REDIS，跳过隔离工作流集成验证")
	}
	return dsn, address
}

// 仅使用显式传入的隔离数据库和 Redis，不读取开发服务配置，不调用模型。
func newWorkspaceHarness(t *testing.T) *workspaceHarness {
	t.Helper()
	dsn, address := workspaceEnvironment(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("隔离 MySQL 初始化失败")
	}
	t.Cleanup(func() { _ = db.Close() })
	if db.PingContext(ctx) != nil {
		t.Fatal("隔离 MySQL 不可达")
	}
	conf := cache.CacheConf{{RedisConf: redis.RedisConf{Host: address, Type: "node"}, Weight: 100}}
	rdb, err := redis.NewRedis(conf[0].RedisConf)
	if err != nil || !rdb.PingCtx(ctx) {
		t.Fatal("隔离 Redis 不可达")
	}
	conn := sqlx.NewSqlConnFromDB(db)
	h := &workspaceHarness{t: t, ctx: ctx, actor: &aipb.Actor{UserId: time.Now().UnixNano()},
		service: &svc.ServiceContext{DB: conn, Redis: rdb,
			AiWorkflowModel: model.NewAiWorkflowModel(conn, conf), AiTaskModel: model.NewAiTaskModel(conn, conf)}}
	t.Logf("隔离工作流测试 owner_id=%d；清理范围仅该归属的任务、工作流及精确缓存键", h.actor.UserId)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		for _, item := range []struct{ table, prefix string }{
			{"ai_task", "cache:gozeroAdmin:aiTask:id:"},
			{"ai_workflow", "cache:gozeroAdmin:aiWorkflow:id:"},
		} {
			var ids []int64
			if err := conn.QueryRowsCtx(cleanupCtx, &ids, "SELECT id FROM "+item.table+" WHERE created_by=?", h.actor.UserId); err != nil {
				t.Error("读取本轮隔离清理 ID 失败")
				continue
			}
			for _, id := range ids {
				if _, err := conn.ExecCtx(cleanupCtx, "DELETE FROM "+item.table+" WHERE id=? AND created_by=?", id, h.actor.UserId); err != nil {
					t.Error("清理本轮隔离行失败")
				}
				key := fmt.Sprintf("%s%d", item.prefix, id)
				if _, err := rdb.DelCtx(cleanupCtx, key); err != nil {
					t.Error("清理本轮精确缓存键失败")
				}
				if exists, err := rdb.ExistsCtx(cleanupCtx, key); err != nil || exists {
					t.Error("本轮精确缓存键清理后仍存在或无法核对")
				}
			}
			var remaining int64
			if err := conn.QueryRowCtx(cleanupCtx, &remaining, "SELECT COUNT(*) FROM "+item.table+" WHERE created_by=?", h.actor.UserId); err != nil || remaining != 0 {
				t.Error("本轮隔离行清理后仍存在或无法核对")
			}
			t.Logf("清理核对 %s：本轮 %d 行及对应缓存键，剩余 %d 行", item.table, len(ids), remaining)
		}
	})
	return h
}

func (h *workspaceHarness) create(definition string) int64 {
	h.t.Helper()
	response, err := NewCreateWorkflowLogic(h.ctx, h.service).CreateWorkflow(&aipb.CreateWorkflowRequest{
		Actor: h.actor, Name: "  隔离工作流回归  ", Definition: definition})
	if err != nil {
		h.t.Fatal(err)
	}
	if response.Workflow.Name != "隔离工作流回归" || response.Workflow.Enabled != 1 || response.Workflow.IntervalSeconds != 0 {
		h.t.Fatal("新工作流必须去除名称空白，默认启用且不调度")
	}
	return response.Workflow.Id
}

func (h *workspaceHarness) run(id int64) int64 {
	h.t.Helper()
	response, err := NewRunWorkflowLogic(h.ctx, h.service).RunWorkflow(&aipb.RunWorkflowRequest{
		Actor: h.actor, WorkflowId: id, Input: `{"record":"only"}`})
	if err != nil {
		h.t.Fatal(err)
	}
	task := h.task(response.TaskId)
	if response.Status != task.Status || task.Input != `{"record":"only"}` {
		h.t.Fatal("运行响应必须匹配落库状态，输入须保存为本次记录")
	}
	return response.TaskId
}

func (h *workspaceHarness) task(id int64) *aipb.AgentTaskInfo {
	h.t.Helper()
	response, err := NewGetAgentTaskLogic(h.ctx, h.service).GetAgentTask(&aipb.GetAgentTaskRequest{Actor: h.actor, TaskId: id})
	if err != nil {
		h.t.Fatal(err)
	}
	// 每次读详情同时预热/核对生成 CRUD 缓存，覆盖后续审批和终态的精确失效。
	cached, err := h.service.AiTaskModel.FindOne(h.ctx, uint64(id))
	if err != nil || cached.Status != response.Task.Status || cached.NextStep != uint64(response.Task.NextStep) || cached.Result.String != response.Task.Result || cached.Error != response.Task.Error {
		h.t.Fatal("任务 ID 缓存与详情状态不一致")
	}
	return response.Task
}

func (h *workspaceHarness) approve(id, step int64) error {
	_, err := NewApproveTaskLogic(h.ctx, h.service).ApproveTask(&aipb.ApproveTaskRequest{Actor: h.actor, TaskId: id, ExpectedStep: &step})
	return err
}

func assertWorkspaceCode(t *testing.T, err error, code uint32) {
	t.Helper()
	actual, ok := xerr.AsCodeError(err)
	if !ok || actual.GetErrCode() != code {
		t.Fatalf("预期业务错误 %d，实际 %v", code, err)
	}
}

const twoApprovalDefinition = `{"steps":[{"name":"首次审批","tool":"text_stats","input":{"text":"原始快照"},"approval":true},{"name":"自动步骤","tool":"get_current_time","input":{}},{"name":"末步审批","tool":"json_formatter","input":{"text":"{\"snapshot\":true}"},"approval":true}]}`

func TestAIWorkspaceIntegrationApprovalReplayAndFinalStep(t *testing.T) {
	h := newWorkspaceHarness(t)
	id := h.run(h.create(twoApprovalDefinition))
	if task := h.task(id); task.Status != "pending_approval" || task.NextStep != 0 || task.Result != "" || task.CompletedAt != 0 {
		t.Fatal("首次审批必须在执行前暂停，expected_step=0 有效")
	}
	_, err := NewApproveTaskLogic(h.ctx, h.service).ApproveTask(&aipb.ApproveTaskRequest{Actor: h.actor, TaskId: id})
	assertWorkspaceCode(t, err, 400)
	// 同一旧断点同时批准，只有一个请求能领取执行权。
	start, results := make(chan struct{}), make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { <-start; results <- h.approve(id, 0) }()
	}
	close(start)
	successes := 0
	for i := 0; i < 2; i++ {
		if err := <-results; err == nil {
			successes++
		} else {
			assertWorkspaceCode(t, err, 400)
		}
	}
	if successes != 1 {
		t.Fatalf("同一审批点只能成功一次，实际 %d", successes)
	}
	before := h.task(id)
	if before.Status != "pending_approval" || before.NextStep != 2 || before.CompletedAt != 0 ||
		strings.Count(before.Result, "[1/3]") != 1 || strings.Count(before.Result, "[2/3]") != 1 || strings.Contains(before.Result, "[3/3]") {
		t.Fatal("末步执行前必须再次暂停，已执行步骤不得重复")
	}
	assertWorkspaceCode(t, h.approve(id, 0), 400)
	zero := int64(0)
	_, err = NewRejectTaskLogic(h.ctx, h.service).RejectTask(&aipb.RejectTaskRequest{Actor: h.actor, TaskId: id, ExpectedStep: &zero, Reason: "迟到拒绝"})
	assertWorkspaceCode(t, err, 400)
	if after := h.task(id); after.Status != before.Status || after.NextStep != before.NextStep || after.Result != before.Result {
		t.Fatal("迟到的批准或拒绝不得改变第二审批点")
	}
	if err := h.approve(id, before.NextStep); err != nil {
		t.Fatal(err)
	}
	done := h.task(id)
	if done.Status != "succeeded" || done.NextStep != 3 || done.CompletedAt == 0 || done.ApprovedBy != h.actor.UserId || strings.Count(done.Result, "[3/3]") != 1 {
		t.Fatal("末步批准必须完成并记录审批人、终态及进度")
	}
	assertWorkspaceCode(t, h.approve(id, 2), 400)
	if replayed := h.task(id); replayed.Result != done.Result || replayed.CompletedAt != done.CompletedAt {
		t.Fatal("终态批准重放不得再次执行")
	}
}

func TestAIWorkspaceIntegrationSnapshotSurvivesWorkflowChanges(t *testing.T) {
	workspaceEnvironment(t)
	for _, action := range []string{"edit", "disable", "delete"} {
		t.Run(action, func(t *testing.T) {
			h := newWorkspaceHarness(t)
			definition := `{"steps":[{"name":"原始快照步骤","tool":"text_stats","input":{"text":"旧"},"approval":true}]}`
			workflowId := h.create(definition)
			id := h.run(workflowId)
			// 预热生成 CRUD 缓存，随后修改必须使其失效。
			if _, err := h.service.AiWorkflowModel.FindOne(h.ctx, uint64(workflowId)); err != nil {
				t.Fatal(err)
			}
			var err error
			switch action {
			case "edit":
				changed := `{"steps":[{"name":"新定义步骤","tool":"text_stats","input":{"text":"新定义"}}]}`
				_, err = NewUpdateWorkflowLogic(h.ctx, h.service).UpdateWorkflow(&aipb.UpdateWorkflowRequest{Actor: h.actor, WorkflowId: workflowId, Definition: &changed})
			case "disable":
				disabled := int64(0)
				_, err = NewUpdateWorkflowLogic(h.ctx, h.service).UpdateWorkflow(&aipb.UpdateWorkflowRequest{Actor: h.actor, WorkflowId: workflowId, Enabled: &disabled})
			case "delete":
				_, err = NewDeleteWorkflowLogic(h.ctx, h.service).DeleteWorkflow(&aipb.DeleteWorkflowRequest{Actor: h.actor, WorkflowId: workflowId})
			}
			if err != nil {
				t.Fatal(err)
			}
			cached, err := h.service.AiWorkflowModel.FindOne(h.ctx, uint64(workflowId))
			if err != nil || (action == "disable" && cached.Enabled != 0) || (action == "delete" && !cached.DeletedAt.Valid) || (action == "edit" && !strings.Contains(cached.Definition, "新定义步骤")) {
				t.Fatal("工作流写入后 ID 缓存必须反映新状态")
			}
			if action != "edit" {
				_, err := NewRunWorkflowLogic(h.ctx, h.service).RunWorkflow(&aipb.RunWorkflowRequest{Actor: h.actor, WorkflowId: workflowId})
				code := uint32(400)
				if action == "delete" {
					code = 404
				}
				assertWorkspaceCode(t, err, code)
				_, total, err := h.service.AiTaskModel.ListOwned(h.ctx, h.actor.UserId, 1, 20, "", workflowId)
				if err != nil || total != 1 {
					t.Fatal("停用或删除后不得创建新任务")
				}
			}
			if err := h.approve(id, 0); err != nil {
				t.Fatal(err)
			}
			old := h.task(id)
			if old.Status != "succeeded" || !strings.Contains(old.Result, "原始快照步骤") || !strings.Contains(old.Result, `"chars":1`) || strings.Contains(old.Result, "新定义步骤") {
				t.Fatal("历史任务必须使用创建时快照完成")
			}
			if action == "edit" {
				fresh := h.task(h.run(workflowId))
				if fresh.Status != "succeeded" || !strings.Contains(fresh.Result, "新定义步骤") {
					t.Fatal("后续新任务必须使用已保存的新定义")
				}
			}
		})
	}
}

func TestAIWorkspaceIntegrationRejectReasonAndReplay(t *testing.T) {
	h := newWorkspaceHarness(t)
	id := h.run(h.create(twoApprovalDefinition))
	zero := int64(0)
	for _, reason := range []string{" \n\t ", strings.Repeat("拒", 513)} {
		_, err := NewRejectTaskLogic(h.ctx, h.service).RejectTask(&aipb.RejectTaskRequest{Actor: h.actor, TaskId: id, ExpectedStep: &zero, Reason: reason})
		assertWorkspaceCode(t, err, 400)
		if task := h.task(id); task.Status != "pending_approval" || task.Result != "" {
			t.Fatal("无效原因不能改变任务")
		}
	}
	_, err := NewRejectTaskLogic(h.ctx, h.service).RejectTask(&aipb.RejectTaskRequest{Actor: h.actor, TaskId: id, Reason: "缺少断点"})
	assertWorkspaceCode(t, err, 400)
	reason := strings.Repeat("拒", 512)
	_, err = NewRejectTaskLogic(h.ctx, h.service).RejectTask(&aipb.RejectTaskRequest{Actor: h.actor, TaskId: id, ExpectedStep: &zero, Reason: " \n" + reason + "\t "})
	if err != nil {
		t.Fatal(err)
	}
	before := h.task(id)
	if before.Status != "rejected" || before.Error != reason || before.ApprovedBy != h.actor.UserId || before.CompletedAt == 0 || before.NextStep != 0 || before.Result != "" {
		t.Fatal("拒绝必须去除原因空白并终结任务，不执行工具")
	}
	_, err = NewRejectTaskLogic(h.ctx, h.service).RejectTask(&aipb.RejectTaskRequest{Actor: h.actor, TaskId: id, ExpectedStep: &zero, Reason: "重放原因"})
	assertWorkspaceCode(t, err, 400)
	assertWorkspaceCode(t, h.approve(id, 0), 400)
	if after := h.task(id); after.Error != before.Error || after.Status != "rejected" || after.CompletedAt != before.CompletedAt {
		t.Fatal("拒绝后重放不能覆盖原因或恢复执行")
	}
}

func TestAIWorkspaceIntegrationOwnershipAndFilters(t *testing.T) {
	h := newWorkspaceHarness(t)
	workflowId := h.create(twoApprovalDefinition)
	id := h.run(workflowId)
	foreign, zero := &aipb.Actor{UserId: h.actor.UserId + 1}, int64(0)
	_, err := NewGetWorkflowLogic(h.ctx, h.service).GetWorkflow(&aipb.GetWorkflowRequest{Actor: foreign, WorkflowId: workflowId})
	assertWorkspaceCode(t, err, 404)
	_, err = NewUpdateWorkflowLogic(h.ctx, h.service).UpdateWorkflow(&aipb.UpdateWorkflowRequest{Actor: foreign, WorkflowId: workflowId, Enabled: &zero})
	assertWorkspaceCode(t, err, 404)
	_, err = NewDeleteWorkflowLogic(h.ctx, h.service).DeleteWorkflow(&aipb.DeleteWorkflowRequest{Actor: foreign, WorkflowId: workflowId})
	assertWorkspaceCode(t, err, 404)
	_, err = NewRunWorkflowLogic(h.ctx, h.service).RunWorkflow(&aipb.RunWorkflowRequest{Actor: foreign, WorkflowId: workflowId})
	assertWorkspaceCode(t, err, 404)
	_, err = NewGetAgentTaskLogic(h.ctx, h.service).GetAgentTask(&aipb.GetAgentTaskRequest{Actor: foreign, TaskId: id})
	assertWorkspaceCode(t, err, 404)
	_, err = NewApproveTaskLogic(h.ctx, h.service).ApproveTask(&aipb.ApproveTaskRequest{Actor: foreign, TaskId: id, ExpectedStep: &zero})
	assertWorkspaceCode(t, err, 404)
	_, err = NewRejectTaskLogic(h.ctx, h.service).RejectTask(&aipb.RejectTaskRequest{Actor: foreign, TaskId: id, ExpectedStep: &zero, Reason: "跨用户"})
	assertWorkspaceCode(t, err, 404)
	workflows, err := NewListWorkflowsLogic(h.ctx, h.service).ListWorkflows(&aipb.ListWorkflowsRequest{Actor: foreign})
	if err != nil || workflows.Page.Total != 0 || len(workflows.List) != 0 {
		t.Fatal("工作流列表不得暴露其他用户数据")
	}
	tasks, err := NewListAgentTasksLogic(h.ctx, h.service).ListAgentTasks(&aipb.ListAgentTasksRequest{Actor: foreign, WorkflowId: workflowId, Status: "pending_approval"})
	if err != nil || tasks.Page.Total != 0 || len(tasks.List) != 0 {
		t.Fatal("带过滤的任务列表不得暴露其他用户数据")
	}
	tasks, err = NewListAgentTasksLogic(h.ctx, h.service).ListAgentTasks(&aipb.ListAgentTasksRequest{Actor: h.actor, WorkflowId: workflowId, Status: "pending_approval"})
	if err != nil || tasks.Page.Total != 1 || len(tasks.List) != 1 || tasks.List[0].Id != id {
		t.Fatal("本人任务过滤应返回目标任务和正确总数")
	}
	tasks, err = NewListAgentTasksLogic(h.ctx, h.service).ListAgentTasks(&aipb.ListAgentTasksRequest{Actor: h.actor, WorkflowId: workflowId, Status: "succeeded"})
	if err != nil || tasks.Page.Total != 0 || len(tasks.List) != 0 {
		t.Fatal("任务状态过滤必须生效")
	}
}

func TestAIWorkspaceIntegrationToolFailurePersists(t *testing.T) {
	h := newWorkspaceHarness(t)
	definition := `{"steps":[{"name":"输入错误","tool":"json_formatter","input":{"text":"invalid json"}},{"name":"不得执行","tool":"get_current_time","input":{}}]}`
	id := h.run(h.create(definition))
	failed := h.task(id)
	if failed.Status != "failed" || failed.Error != "步骤工具执行失败" || failed.NextStep != 0 || failed.Result != "" || failed.CompletedAt == 0 {
		t.Fatal("工具失败必须保存 failed、安全错误和当前断点，后续步骤不得执行")
	}
	assertWorkspaceCode(t, h.approve(id, 0), 400)
	if after := h.task(id); after.Status != "failed" || after.Result != failed.Result {
		t.Fatal("失败任务不能通过批准重放工具")
	}
}

func TestAIWorkspaceToolCatalogWithoutModel(t *testing.T) {
	service := &svc.ServiceContext{}
	response, err := NewListToolsLogic(context.Background(), service).ListTools(&aipb.ListToolsRequest{Actor: &aipb.Actor{UserId: 1}})
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{"get_current_time": false, "text_stats": false, "json_formatter": false}
	previous := ""
	for _, tool := range response.List {
		if tool.Name <= previous || tool.Description == "" {
			t.Fatal("工具目录必须按名称排序、没有重复且包含说明")
		}
		previous = tool.Name
		var schema map[string]any
		if json.Unmarshal([]byte(tool.ParametersSchema), &schema) != nil || schema["type"] != "object" {
			t.Fatal("工具参数必须提供有效 JSON 对象 schema")
		}
		if _, ok := wanted[tool.Name]; ok {
			wanted[tool.Name] = true
		}
	}
	for name, found := range wanted {
		if !found {
			t.Errorf("缺少现有本地工具 %s", name)
		}
	}
	_, err = NewListToolsLogic(context.Background(), service).ListTools(&aipb.ListToolsRequest{})
	assertWorkspaceCode(t, err, xerr.PERMISSION_DENIED)
}
