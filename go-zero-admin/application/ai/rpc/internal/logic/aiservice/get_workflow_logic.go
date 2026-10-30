package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"go-zero-admin/pkg/xerr"
)

type GetWorkflowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetWorkflowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWorkflowLogic {
	return &GetWorkflowLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetWorkflowLogic) GetWorkflow(in *aipb.GetWorkflowRequest) (*aipb.GetWorkflowResponse, error) {
	var r workflowRow
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &r,
		"SELECT id, name, CAST(definition AS CHAR), interval_seconds, enabled, COALESCE(UNIX_TIMESTAMP(next_run_at),0), created_by, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM ai_workflow WHERE id=? AND deleted_at IS NULL",
		in.WorkflowId)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(404, "工作流不存在")
	}
	return &aipb.GetWorkflowResponse{Workflow: r.toPb()}, nil
}
