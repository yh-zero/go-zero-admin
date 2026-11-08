package aiservicelogic

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type RestoreWorkflowRevisionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRestoreWorkflowRevisionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RestoreWorkflowRevisionLogic {
	return &RestoreWorkflowRevisionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RestoreWorkflowRevisionLogic) RestoreWorkflowRevision(in *aipb.RestoreWorkflowRevisionRequest) (*aipb.GetWorkflowResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	if in.WorkflowId < 1 || in.Revision < 1 || in.ExpectedRevision < 1 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	old, err := l.svcCtx.AiWorkflowModel.FindRevisionOwned(l.ctx, in.WorkflowId, in.Actor.UserId, in.Revision)
	if err != nil {
		return nil, workflowModelError(err)
	}
	if _, err := parseWorkflow(old.Definition); err != nil {
		return nil, err
	}
	row, err := l.svcCtx.AiWorkflowModel.RestoreRevisionOwned(l.ctx, in.WorkflowId, in.Actor.UserId, in.Revision, in.ExpectedRevision)
	if err != nil {
		return nil, workflowModelError(err)
	}
	return &aipb.GetWorkflowResponse{Workflow: workflowProto(row)}, nil
}
