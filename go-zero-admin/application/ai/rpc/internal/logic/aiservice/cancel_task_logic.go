package aiservicelogic

import (
	"context"
	"errors"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type CancelTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCancelTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CancelTaskLogic {
	return &CancelTaskLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CancelTaskLogic) CancelTask(in *aipb.CancelTaskRequest) (*aipb.NoDataResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	if err := l.svcCtx.AiTaskModel.CancelOwned(l.ctx, in.TaskId, in.Actor.UserId); err != nil {
		if errors.Is(err, model.ErrTaskCommitted) {
			cancelLocalWorkflow(l.svcCtx, in.TaskId)
			l.Error("workflow cancellation committed; cache invalidation failed")
		}
		return nil, aiModelError(err)
	}
	cancelLocalWorkflow(l.svcCtx, in.TaskId)
	return &aipb.NoDataResponse{}, nil
}
