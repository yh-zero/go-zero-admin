package aiservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
	"strings"
	"unicode/utf8"
)

type RejectTaskLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRejectTaskLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RejectTaskLogic {
	return &RejectTaskLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *RejectTaskLogic) RejectTask(in *aipb.RejectTaskRequest) (*aipb.NoDataResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	reason := strings.TrimSpace(in.Reason)
	if reason == "" || utf8.RuneCountInString(reason) > 512 {
		return nil, xerr.NewErrCodeMsg(400, "拒绝原因须为 1..512 字符")
	}
	task, err := l.svcCtx.AiTaskModel.FindOwned(l.ctx, in.TaskId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	if task.Status != "pending_approval" {
		return nil, xerr.NewErrCodeMsg(400, "任务不在待审批状态")
	}
	if err := l.svcCtx.AiTaskModel.RejectOwned(l.ctx, in.TaskId, in.Actor.UserId, task.NextStep, reason); err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.NoDataResponse{}, nil
}
