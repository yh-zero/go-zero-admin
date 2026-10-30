package aiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"go-zero-admin/pkg/xerr"
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
	var r taskRow
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &r,
		"SELECT id, COALESCE(workflow_id,0), COALESCE(conversation_id,0), status, COALESCE(input,''), COALESCE(result,''), error, COALESCE(approved_by,0), UNIX_TIMESTAMP(created_at), COALESCE(UNIX_TIMESTAMP(completed_at),0) FROM ai_task WHERE id=?",
		in.TaskId)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(404, "任务不存在")
	}
	return &aipb.GetAgentTaskResponse{Task: r.toPb()}, nil
}
