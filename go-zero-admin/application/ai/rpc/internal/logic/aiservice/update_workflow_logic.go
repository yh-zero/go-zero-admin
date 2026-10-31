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

type UpdateWorkflowLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateWorkflowLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateWorkflowLogic {
	return &UpdateWorkflowLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *UpdateWorkflowLogic) UpdateWorkflow(in *aipb.UpdateWorkflowRequest) (*aipb.NoDataResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.AiWorkflowModel.FindOwned(l.ctx, in.WorkflowId, in.Actor.UserId); err != nil {
		return nil, aiModelError(err)
	}
	patch := model.WorkflowPatch{Name: in.Name, Definition: in.Definition, IntervalSeconds: in.IntervalSeconds, Enabled: in.Enabled}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" || utf8.RuneCountInString(name) > 128 {
			return nil, xerr.NewErrCodeMsg(400, "工作流名称须为 1..128 字符")
		}
		patch.Name = &name
	}
	if in.Definition != nil {
		if _, err := parseWorkflow(*in.Definition); err != nil {
			return nil, err
		}
	}
	if in.IntervalSeconds != nil && *in.IntervalSeconds != 0 {
		return nil, xerr.NewErrCodeMsg(400, "本轮仅支持 interval_seconds=0 手动执行")
	}
	if in.Enabled != nil && *in.Enabled != 0 && *in.Enabled != 1 {
		return nil, xerr.NewErrCodeMsg(400, "enabled 仅允许 0 或 1")
	}
	if err := l.svcCtx.AiWorkflowModel.UpdateOwned(l.ctx, in.WorkflowId, in.Actor.UserId, patch); err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.NoDataResponse{}, nil
}
