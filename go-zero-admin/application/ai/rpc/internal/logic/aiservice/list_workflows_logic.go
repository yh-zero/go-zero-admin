package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"go-zero-admin/pkg/xerr"
)

type ListWorkflowsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListWorkflowsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListWorkflowsLogic {
	return &ListWorkflowsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListWorkflowsLogic) ListWorkflows(in *aipb.ListWorkflowsRequest) (*aipb.ListWorkflowsResponse, error) {
	var total int64
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total, "SELECT COUNT(*) FROM ai_workflow WHERE deleted_at IS NULL")
	page := in.GetPage().GetPage()
	pageSize := in.GetPage().GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	var rows []workflowRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &rows,
		"SELECT id, name, CAST(definition AS CHAR), interval_seconds, enabled, COALESCE(UNIX_TIMESTAMP(next_run_at),0), created_by, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM ai_workflow WHERE deleted_at IS NULL ORDER BY id DESC LIMIT ? OFFSET ?",
		pageSize, offset)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*aipb.WorkflowInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, r.toPb())
	}
	return &aipb.ListWorkflowsResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
