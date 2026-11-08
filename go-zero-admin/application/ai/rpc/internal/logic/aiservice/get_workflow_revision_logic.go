package aiservicelogic

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetWorkflowRevisionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetWorkflowRevisionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetWorkflowRevisionLogic {
	return &GetWorkflowRevisionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetWorkflowRevisionLogic) GetWorkflowRevision(in *aipb.GetWorkflowRevisionRequest) (*aipb.GetWorkflowRevisionResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	if in.WorkflowId < 1 || in.Revision < 1 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	row, err := l.svcCtx.AiWorkflowModel.FindRevisionOwned(l.ctx, in.WorkflowId, in.Actor.UserId, in.Revision)
	if err != nil {
		return nil, workflowModelError(err)
	}
	return &aipb.GetWorkflowRevisionResponse{Revision: workflowRevisionProto(row)}, nil
}
