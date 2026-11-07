package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
)

type GetAgentTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAgentTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAgentTaskLogic {
	return &GetAgentTaskLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *GetAgentTaskLogic) GetAgentTask(in *aipb.GetAgentTaskRequest) (*aipb.GetAgentTaskResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	if err := l.svcCtx.AiTaskModel.FailStaleOwned(l.ctx, in.TaskId, in.Actor.UserId); err != nil {
		return nil, aiModelError(err)
	}
	row, err := l.svcCtx.AiTaskModel.FindOwned(l.ctx, in.TaskId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.GetAgentTaskResponse{Task: taskProto(row)}, nil
}
