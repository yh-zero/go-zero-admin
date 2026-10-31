package aiservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type ListAgentTasksLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAgentTasksLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAgentTasksLogic {
	return &ListAgentTasksLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *ListAgentTasksLogic) ListAgentTasks(in *aipb.ListAgentTasksRequest) (*aipb.ListAgentTasksResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	page, size, err := aiPagination(in.Page)
	if err != nil {
		return nil, err
	}
	switch in.Status {
	case "", "pending", "running", "pending_approval", "succeeded", "failed", "rejected":
	default:
		return nil, xerr.NewErrCodeMsg(400, "任务状态不合法")
	}
	if in.WorkflowId < 0 {
		return nil, xerr.NewErrCodeMsg(400, "工作流 ID 不合法")
	}
	rows, total, err := l.svcCtx.AiTaskModel.ListOwned(l.ctx, in.Actor.UserId, page, size, in.Status, in.WorkflowId)
	if err != nil {
		return nil, aiModelError(err)
	}
	list := make([]*aipb.AgentTaskInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, taskProto(row))
	}
	return &aipb.ListAgentTasksResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
