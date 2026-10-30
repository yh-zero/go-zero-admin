package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"go-zero-admin/pkg/xerr"
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
	var status string
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &status, "SELECT status FROM ai_task WHERE id=?", in.TaskId)
	if err != nil || status != "pending_approval" {
		return nil, xerr.NewErrCodeMsg(400, "任务不在待审批状态")
	}
	_, err = l.svcCtx.DB.ExecCtx(l.ctx,
		"UPDATE ai_task SET status='rejected', approved_by=?, error=?, completed_at=NOW() WHERE id=?",
		in.Actor.UserId, in.Reason, in.TaskId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &aipb.NoDataResponse{}, nil
}
