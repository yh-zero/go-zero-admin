package aiservicelogic

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"google.golang.org/protobuf/proto"
)

func TestAIWorkspaceProductDraftRejectsInvalidBeforePersistence(t *testing.T) {
	request := &aipb.RunDraftRequest{Actor: &aipb.Actor{UserId: 1}, Name: "临时统计", Definition: `{"version":2,"steps":[{"id":"stats","name":"统计","tool":"text_stats","input":{"text":{"$ref":"input.text"}}}]}`, Input: `{"text":"示例"}`, RequestId: uuid.NewString()}
	for _, field := range []string{"name", "definition", "input", "request_id"} {
		t.Run(field, func(t *testing.T) {
			invalid := proto.Clone(request).(*aipb.RunDraftRequest)
			switch field {
			case "name":
				invalid.Name = "  "
			case "definition":
				invalid.Definition = `{"steps":[]}`
			case "input":
				invalid.Input = `[]`
			case "request_id":
				invalid.RequestId = "invalid"
			}
			_, err := NewRunDraftLogic(context.Background(), &svc.ServiceContext{}).RunDraft(invalid)
			assertWorkspaceCode(t, err, 400)
		})
	}
}

func TestAIWorkspaceProductDraftSnapshotIdempotencyAndMetadata(t *testing.T) {
	h := newWorkspaceHarness(t)
	definition := `{"version":2,"steps":[{"id":"stats","name":"统计输入","tool":"text_stats","input":{"text":{"$ref":"input.text"}}}]}`
	request := &aipb.RunDraftRequest{Actor: h.actor, Name: "一次性文本统计", Definition: definition, Input: `{"text":"测试文本","question":"检查本次资料"}`, RequestId: uuid.NewString()}
	response, err := NewRunDraftLogic(h.ctx, h.service).RunDraft(request)
	if err != nil || response.TaskId <= 0 || response.Status != "running" {
		t.Fatalf("未保存草稿应直接返回有效任务: response=%v err=%v", response, err)
	}
	done := h.task(response.TaskId)
	if done.Status != "succeeded" || done.WorkflowId != 0 || done.Title != request.Name || done.WorkflowName != request.Name || done.InputSummary != "检查本次资料" || len(done.Steps) != 1 {
		t.Fatalf("临时运行须保留标题、问题、定义与实际步骤: %+v", done)
	}
	var expectedDefinition, storedDefinition any
	if json.Unmarshal([]byte(definition), &expectedDefinition) != nil || json.Unmarshal([]byte(done.Definition), &storedDefinition) != nil || !reflect.DeepEqual(expectedDefinition, storedDefinition) {
		t.Fatal("任务定义须保留原始语义，允许MySQL JSON格式化")
	}
	replay, err := NewRunDraftLogic(h.ctx, h.service).RunDraft(request)
	if err != nil || replay.TaskId != response.TaskId || replay.Status != "succeeded" {
		t.Fatal("重查必须复用同一完成任务")
	}
	for _, change := range []string{"name", "definition", "input"} {
		changed := proto.Clone(request).(*aipb.RunDraftRequest)
		switch change {
		case "name":
			changed.Name = "不同用途"
		case "definition":
			changed.Definition = `{"steps":[{"name":"当前时间","tool":"get_current_time","input":{}}]}`
		case "input":
			changed.Input = `{"text":"另一内容"}`
		}
		_, err = NewRunDraftLogic(h.ctx, h.service).RunDraft(changed)
		assertWorkspaceCode(t, err, 400)
	}
	metadata, err := NewGetAgentTaskLogic(h.ctx, h.service).GetAgentTask(&aipb.GetAgentTaskRequest{Actor: h.actor, TaskId: response.TaskId, MetaOnly: true})
	if err != nil || metadata.Task.Status != "succeeded" || metadata.Task.Input != "" || metadata.Task.Result != "" || metadata.Task.Definition != "" || len(metadata.Task.Steps) != 0 || metadata.Task.Title != request.Name {
		t.Fatalf("轻量读取不返回大字段且保留可读摘要: %v %v", metadata, err)
	}
	_, err = NewGetAgentTaskLogic(h.ctx, h.service).GetAgentTask(&aipb.GetAgentTaskRequest{Actor: &aipb.Actor{UserId: h.actor.UserId + 1}, TaskId: response.TaskId, MetaOnly: true})
	assertWorkspaceCode(t, err, 404)
	var workflowCount int64
	if err := h.service.DB.QueryRowCtx(h.ctx, &workflowCount, "SELECT COUNT(*) FROM ai_workflow WHERE created_by=?", h.actor.UserId); err != nil || workflowCount != 0 {
		t.Fatal("试运行不能制造已保存流程")
	}
	listed, err := NewListAgentTasksLogic(h.ctx, h.service).ListAgentTasks(&aipb.ListAgentTasksRequest{Actor: h.actor, Query: "本次资料", Page: &aipb.PageRequest{Page: 1, PageSize: 1}})
	if err != nil || listed.Page.Total != 1 || len(listed.List) != 1 || listed.List[0].Result != "" || listed.List[0].Definition != "" || listed.List[0].Title != request.Name {
		t.Fatal("任务全量搜索与摘要列表必须匹配")
	}
}

func TestAIWorkspaceProductWorkflowSearchLiteralAndSummary(t *testing.T) {
	h := newWorkspaceHarness(t)
	definition := `{"steps":[{"name":"时间","tool":"get_current_time","input":{}}]}`
	for _, name := range []string{"资料%_检查", "资料甲乙检查", "另一流程"} {
		if _, err := NewCreateWorkflowLogic(h.ctx, h.service).CreateWorkflow(&aipb.CreateWorkflowRequest{Actor: h.actor, Name: name, Definition: definition}); err != nil {
			t.Fatal(err)
		}
	}
	response, err := NewListWorkflowsLogic(h.ctx, h.service).ListWorkflows(&aipb.ListWorkflowsRequest{Actor: h.actor, Query: "%_", Page: &aipb.PageRequest{Page: 1, PageSize: 1}})
	if err != nil || response.Page.Total != 1 || len(response.List) != 1 || response.List[0].Name != "资料%_检查" || response.List[0].Definition != "" || response.List[0].StepCount != 1 {
		t.Fatalf("搜索应在分页前按文字筛选，列表仅返回步骤摘要: %v %v", response, err)
	}
	foreign, err := NewListWorkflowsLogic(h.ctx, h.service).ListWorkflows(&aipb.ListWorkflowsRequest{Actor: &aipb.Actor{UserId: h.actor.UserId + 1}, Query: "%_"})
	if err != nil || foreign.Page.Total != 0 {
		t.Fatal("搜索不能扩大归属范围")
	}
}
