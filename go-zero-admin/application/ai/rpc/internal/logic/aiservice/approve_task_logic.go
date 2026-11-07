package aiservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type ApproveTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewApproveTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApproveTaskLogic {
	return &ApproveTaskLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *ApproveTaskLogic) ApproveTask(in *aipb.ApproveTaskRequest) (*aipb.NoDataResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	if in.ExpectedStep == nil || *in.ExpectedStep < 0 {
		return nil, xerr.NewErrCodeMsg(400, "expected_step 必填且不得为负")
	}
	task, err := l.svcCtx.AiTaskModel.FindOwned(l.ctx, in.TaskId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	if task.Status != "pending_approval" || task.NextStep != *in.ExpectedStep {
		return nil, xerr.NewErrCodeMsg(400, "任务不在待审批状态")
	}
	definition, err := parseWorkflow(task.WorkflowSnapshot)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(400, "任务缺少有效执行快照，不能续跑")
	}
	if task.NextStep < 0 || task.NextStep >= int64(len(definition.Steps)) || !definition.Steps[task.NextStep].Approval {
		return nil, xerr.NewErrCodeMsg(400, "任务审批断点无效")
	}
	if !reserveWorkflowSlot() {
		return nil, xerr.NewErrCodeMsg(503, "工作流繁忙，请刷新后重试批准")
	}
	if err := l.svcCtx.AiTaskModel.ClaimApproval(l.ctx, task.Id, in.Actor.UserId, *in.ExpectedStep); err != nil {
		if errors.Is(err, model.ErrTaskCommitted) {
			launchWorkflow(l.ctx, l.svcCtx, task, true)
			l.Error("workflow approval committed; cache invalidation failed")
			return nil, aiModelError(err)
		}
		releaseWorkflowSlot()
		return nil, aiModelError(err)
	}
	launchWorkflow(l.ctx, l.svcCtx, task, true)
	return &aipb.NoDataResponse{}, nil
}
