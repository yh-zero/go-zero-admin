package aiservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
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
	task, err := l.svcCtx.AiTaskModel.FindOwned(l.ctx, in.TaskId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	if task.Status != "pending_approval" {
		return nil, xerr.NewErrCodeMsg(400, "任务不在待审批状态")
	}
	definition, err := parseWorkflow(task.WorkflowSnapshot)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(400, "任务缺少有效执行快照，不能续跑")
	}
	if task.NextStep < 0 || task.NextStep >= int64(len(definition.Steps)) || !definition.Steps[task.NextStep].Approval {
		return nil, xerr.NewErrCodeMsg(400, "任务审批断点无效")
	}
	if err := l.svcCtx.AiTaskModel.ClaimApproval(l.ctx, task.Id, in.Actor.UserId, task.NextStep); err != nil {
		return nil, aiModelError(err)
	}
	if _, err := executeWorkflow(l.ctx, l.svcCtx, task, true); err != nil {
		return nil, err
	}
	return &aipb.NoDataResponse{}, nil
}
