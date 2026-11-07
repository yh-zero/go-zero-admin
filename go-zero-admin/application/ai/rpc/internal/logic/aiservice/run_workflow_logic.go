package aiservicelogic

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type RunWorkflowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRunWorkflowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RunWorkflowLogic {
	return &RunWorkflowLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *RunWorkflowLogic) RunWorkflow(in *aipb.RunWorkflowRequest) (*aipb.RunWorkflowResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	input := in.Input
	if input == "" {
		input = "{}"
	}
	if len(input) > 16000 || !json.Valid([]byte(input)) {
		return nil, xerr.NewErrCodeMsg(400, "input 必须为最多 16000 字节的 JSON")
	}
	var object map[string]json.RawMessage
	if json.Unmarshal([]byte(input), &object) != nil || object == nil {
		return nil, xerr.NewErrCodeMsg(400, "运行输入必须为 JSON 对象")
	}
	canonical, _ := json.Marshal(object)
	input = string(canonical)
	parsed, err := uuid.Parse(in.RequestId)
	if err != nil || parsed.String() != in.RequestId {
		return nil, xerr.NewErrCodeMsg(400, "request_id 必须为规范 UUID")
	}
	if existing, readErr := l.svcCtx.AiWorkflowModel.FindTaskRequestOwned(l.ctx, in.Actor.UserId, in.RequestId); readErr == nil {
		if existing.WorkflowId != in.WorkflowId || existing.Input != input {
			return nil, aiModelError(model.ErrTaskRequestConflict)
		}
		return &aipb.RunWorkflowResponse{TaskId: existing.Id, Status: existing.Status}, nil
	} else if !errors.Is(readErr, model.ErrNotFound) {
		return nil, aiModelError(readErr)
	}
	if !reserveWorkflowSlot() {
		return nil, xerr.NewErrCodeMsg(503, "工作流繁忙，请稍后使用同一请求标识重试")
	}
	task, created, err := l.svcCtx.AiWorkflowModel.CreateTaskIdempotent(l.ctx, in.WorkflowId, in.Actor.UserId, input, in.RequestId)
	if err != nil {
		if created && task != nil && errors.Is(err, model.ErrTaskCommitted) {
			launchWorkflow(l.ctx, l.svcCtx, task, false)
			l.Error("workflow task created; cache invalidation failed after SQL commit")
			return nil, xerr.NewErrCodeMsg(503, "任务已创建，缓存刷新失败，请使用同一请求标识核对")
		}
		releaseWorkflowSlot()
		return nil, aiModelError(err)
	}
	if created {
		launchWorkflow(l.ctx, l.svcCtx, task, false)
	} else {
		releaseWorkflowSlot()
	}
	return &aipb.RunWorkflowResponse{TaskId: task.Id, Status: task.Status}, nil
}
