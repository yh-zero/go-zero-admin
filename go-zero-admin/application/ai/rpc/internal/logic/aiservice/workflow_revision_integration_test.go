package aiservicelogic

import (
	"testing"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"
)

func code(err error) uint32 {
	value, ok := xerr.AsCodeError(err)
	if !ok {
		return 0
	}
	return value.GetErrCode()
}

func TestWorkflowRevisionRPCConditionsHistoryAndOwnership(t *testing.T) {
	h := newWorkspaceHarness(t)
	id := h.create(`{"steps":[{"name":"时间","tool":"get_current_time","input":{}}]}`)
	get, err := NewGetWorkflowLogic(h.ctx, h.service).GetWorkflow(&aipb.GetWorkflowRequest{Actor: h.actor, WorkflowId: id})
	if err != nil || get.Workflow.Revision != 1 {
		t.Fatalf("新建工作流版本应为1: %v", err)
	}
	name := "下一版本"
	if _, err := NewUpdateWorkflowLogic(h.ctx, h.service).UpdateWorkflow(&aipb.UpdateWorkflowRequest{Actor: h.actor, WorkflowId: id, Name: &name}); code(err) != 400 {
		t.Fatal("更新必须提供正版本")
	}
	updated, err := NewUpdateWorkflowLogic(h.ctx, h.service).UpdateWorkflow(&aipb.UpdateWorkflowRequest{Actor: h.actor, WorkflowId: id, ExpectedRevision: 1, Name: &name})
	if err != nil || updated.Workflow.Revision != 2 || updated.Workflow.Name != name {
		t.Fatalf("条件更新必须返回已提交版本: %v", err)
	}
	if _, err := NewUpdateWorkflowLogic(h.ctx, h.service).UpdateWorkflow(&aipb.UpdateWorkflowRequest{Actor: h.actor, WorkflowId: id, ExpectedRevision: 1, Name: &name}); code(err) != 409 {
		t.Fatal("过期版本必须返回409")
	}
	list, err := NewListWorkflowRevisionsLogic(h.ctx, h.service).ListWorkflowRevisions(&aipb.ListWorkflowRevisionsRequest{Actor: h.actor, WorkflowId: id, Page: &aipb.PageRequest{Page: 1, PageSize: 10}})
	if err != nil || list.Page.Total != 1 || len(list.List) != 1 || list.List[0].Revision != 1 || list.List[0].Definition != "" {
		t.Fatalf("历史列表必须分页摘要: %v", err)
	}
	old, err := NewGetWorkflowRevisionLogic(h.ctx, h.service).GetWorkflowRevision(&aipb.GetWorkflowRevisionRequest{Actor: h.actor, WorkflowId: id, Revision: 1})
	if err != nil || old.Revision.Definition == "" || old.Revision.WorkflowId != id {
		t.Fatal("历史详情必须包含实际旧快照")
	}
	foreign := &aipb.Actor{UserId: h.actor.UserId + 1}
	if _, err := NewGetWorkflowRevisionLogic(h.ctx, h.service).GetWorkflowRevision(&aipb.GetWorkflowRevisionRequest{Actor: foreign, WorkflowId: id, Revision: 1}); code(err) != 404 {
		t.Fatal("跨owner历史详情必须404")
	}
	if _, err := NewListWorkflowRevisionsLogic(h.ctx, h.service).ListWorkflowRevisions(&aipb.ListWorkflowRevisionsRequest{Actor: foreign, WorkflowId: id}); code(err) != 404 {
		t.Fatal("跨owner历史列表必须404")
	}
	if _, err := NewRestoreWorkflowRevisionLogic(h.ctx, h.service).RestoreWorkflowRevision(&aipb.RestoreWorkflowRevisionRequest{Actor: foreign, WorkflowId: id, Revision: 1, ExpectedRevision: 2}); code(err) != 404 {
		t.Fatal("跨owner恢复必须404")
	}
	restored, err := NewRestoreWorkflowRevisionLogic(h.ctx, h.service).RestoreWorkflowRevision(&aipb.RestoreWorkflowRevisionRequest{Actor: h.actor, WorkflowId: id, Revision: 1, ExpectedRevision: 2})
	if err != nil || restored.Workflow.Revision != 3 || restored.Workflow.Name != get.Workflow.Name {
		t.Fatalf("恢复必须提交新版本: %v", err)
	}
	if _, err := NewRestoreWorkflowRevisionLogic(h.ctx, h.service).RestoreWorkflowRevision(&aipb.RestoreWorkflowRevisionRequest{Actor: h.actor, WorkflowId: id, Revision: 1, ExpectedRevision: 2}); code(err) != 409 {
		t.Fatal("迟到恢复不能覆盖新版本")
	}
	if _, err := NewRestoreWorkflowRevisionLogic(h.ctx, h.service).RestoreWorkflowRevision(&aipb.RestoreWorkflowRevisionRequest{Actor: h.actor, WorkflowId: id, Revision: 99, ExpectedRevision: 3}); code(err) != 404 {
		t.Fatal("不存在的历史版本须404")
	}
	var tasks int64
	if err := h.service.DB.QueryRowCtx(h.ctx, &tasks, "SELECT COUNT(*) FROM ai_task WHERE workflow_id=? AND created_by=?", id, h.actor.UserId); err != nil || tasks != 0 {
		t.Fatal("历史恢复不能执行或新增任务")
	}
}
