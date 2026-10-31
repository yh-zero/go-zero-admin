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
	where := "1=1"
	var args []interface{}
	if in.Status != "" {
		where += " AND status=?"
		args = append(args, in.Status)
	}
	if in.WorkflowId > 0 {
		where += " AND workflow_id=?"
		args = append(args, in.WorkflowId)
	}
	var total int64
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total, "SELECT COUNT(*) FROM ai_task WHERE "+where, args...)
	page := in.GetPage().GetPage()
	pageSize := in.GetPage().GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	var rows []taskRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &rows,
		"SELECT id, COALESCE(workflow_id,0), COALESCE(conversation_id,0), status, COALESCE(input,''), COALESCE(result,''), error, COALESCE(approved_by,0), UNIX_TIMESTAMP(created_at), COALESCE(UNIX_TIMESTAMP(completed_at),0) FROM ai_task WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*aipb.AgentTaskInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, r.toPb())
	}
	return &aipb.ListAgentTasksResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
