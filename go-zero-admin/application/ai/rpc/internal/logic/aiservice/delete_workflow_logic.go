package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"go-zero-admin/pkg/xerr"
)

type DeleteWorkflowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteWorkflowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteWorkflowLogic {
	return &DeleteWorkflowLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteWorkflowLogic) DeleteWorkflow(in *aipb.DeleteWorkflowRequest) (*aipb.NoDataResponse, error) {
	_, err := l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE ai_workflow SET deleted_at=NOW() WHERE id=?", in.WorkflowId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &aipb.NoDataResponse{}, nil
}
