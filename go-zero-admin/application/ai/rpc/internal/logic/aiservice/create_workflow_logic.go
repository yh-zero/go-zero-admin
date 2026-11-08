package aiservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
	"strings"
	"unicode/utf8"
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
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 128 {
		return nil, xerr.NewErrCodeMsg(400, "工作流名称须为 1..128 字符")
	}
	if in.IntervalSeconds != 0 {
		return nil, xerr.NewErrCodeMsg(400, "本轮仅支持 interval_seconds=0 手动执行")
	}
	if _, err := parseWorkflow(in.Definition); err != nil {
		return nil, err
	}
	result, err := l.svcCtx.AiWorkflowModel.Insert(l.ctx, &model.AiWorkflow{Name: name, Definition: in.Definition, Enabled: 1, CreatedBy: uint64(in.Actor.UserId), RevisionSeq: 1})
	if err != nil {
		return nil, aiModelError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, aiModelError(err)
	}
	row, err := l.svcCtx.AiWorkflowModel.FindOwned(l.ctx, id, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.CreateWorkflowResponse{Workflow: workflowProto(row)}, nil
}
