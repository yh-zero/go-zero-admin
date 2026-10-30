package aiservicelogic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"go-zero-admin/pkg/xerr"
)

type CreateWorkflowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateWorkflowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateWorkflowLogic {
	return &CreateWorkflowLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateWorkflowLogic) CreateWorkflow(in *aipb.CreateWorkflowRequest) (*aipb.CreateWorkflowResponse, error) {
	if in.Name == "" {
		return nil, xerr.NewErrCodeMsg(400, "工作流名称不能为空")
	}
	if !json.Valid([]byte(in.Definition)) {
		return nil, xerr.NewErrCodeMsg(400, "工作流定义必须是合法 JSON")
	}
	now := time.Now()
	var nextRunAt interface{}
	if in.IntervalSeconds > 0 {
		nextRunAt = now.Add(time.Duration(in.IntervalSeconds) * time.Second)
	}
	result, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"INSERT INTO ai_workflow (name, definition, interval_seconds, enabled, next_run_at, created_by, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?)",
		in.Name, in.Definition, in.IntervalSeconds, 1, nextRunAt, in.Actor.UserId, now, now)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	id, _ := result.LastInsertId()
	return &aipb.CreateWorkflowResponse{Workflow: &aipb.WorkflowInfo{
		Id: id, Name: in.Name, Definition: in.Definition, IntervalSeconds: in.IntervalSeconds, Enabled: 1, CreatedBy: in.Actor.UserId, CreatedAt: now.Unix(), UpdatedAt: now.Unix(),
	}}, nil
}

type workflowRow struct {
	Id              int64  `db:"id" `
	Name            string `db:"name" `
	Definition      string `db:"definition" `
	IntervalSeconds int64  `db:"interval_seconds" `
	Enabled         int64  `db:"enabled" `
	NextRunAt       int64  `db:"next_run_at" `
	CreatedBy       int64  `db:"created_by" `
	CreatedAt       int64  `db:"created_at" `
	UpdatedAt       int64  `db:"updated_at" `
}

func (r *workflowRow) toPb() *aipb.WorkflowInfo {
	info := &aipb.WorkflowInfo{Id: r.Id, Name: r.Name, Definition: r.Definition, IntervalSeconds: r.IntervalSeconds, Enabled: r.Enabled, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	if r.NextRunAt > 0 {
		info.NextRunAt = r.NextRunAt
	}
	return info
}

type taskRow struct {
	Id             int64  `db:"id" `
	WorkflowId     int64  `db:"workflow_id" `
	ConversationId int64  `db:"conversation_id" `
	Status         string `db:"status" `
	Input          string `db:"input" `
	Result         string `db:"result" `
	Error          string `db:"error" `
	ApprovedBy     int64  `db:"approved_by" `
	CreatedAt      int64  `db:"created_at" `
	CompletedAt    int64  `db:"completed_at" `
}

func (r *taskRow) toPb() *aipb.AgentTaskInfo {
	return &aipb.AgentTaskInfo{Id: r.Id, WorkflowId: r.WorkflowId, ConversationId: r.ConversationId, Status: r.Status, Input: r.Input, Result: r.Result, Error: r.Error, ApprovedBy: r.ApprovedBy, CreatedAt: r.CreatedAt, CompletedAt: r.CompletedAt}
}
