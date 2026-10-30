package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"go-zero-admin/pkg/xerr"
)

type UpdateWorkflowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateWorkflowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateWorkflowLogic {
	return &UpdateWorkflowLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateWorkflowLogic) UpdateWorkflow(in *aipb.UpdateWorkflowRequest) (*aipb.NoDataResponse, error) {
	_, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"UPDATE ai_workflow SET name=?, definition=?, interval_seconds=?, enabled=?, updated_at=NOW() WHERE id=? AND deleted_at IS NULL",
		in.Name, in.Definition, in.IntervalSeconds, in.Enabled, in.WorkflowId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &aipb.NoDataResponse{}, nil
}
