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
	// 校验任务处于待审批状态
	var status string
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &status, "SELECT status FROM ai_task WHERE id=?", in.TaskId)
	if err != nil || status != "pending_approval" {
		return nil, xerr.NewErrCodeMsg(400, "任务不在待审批状态")
	}
	// 标记审批人，状态转 succeeded（简版：审批通过即完成）
	_, err = l.svcCtx.DB.ExecCtx(l.ctx,
		"UPDATE ai_task SET status='succeeded', approved_by=?, completed_at=NOW() WHERE id=?",
		in.Actor.UserId, in.TaskId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &aipb.NoDataResponse{}, nil
}
