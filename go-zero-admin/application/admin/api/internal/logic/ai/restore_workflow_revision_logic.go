// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package ai

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type RestoreWorkflowRevisionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRestoreWorkflowRevisionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RestoreWorkflowRevisionLogic {
	return &RestoreWorkflowRevisionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RestoreWorkflowRevisionLogic) RestoreWorkflowRevision(req *types.AiRestoreWorkflowRevisionReq) (resp *types.AiWorkflowResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.RestoreWorkflowRevision(l.ctx, &aipb.RestoreWorkflowRevisionRequest{Actor: actor, WorkflowId: req.Id, Revision: req.Revision, ExpectedRevision: req.ExpectedRevision})
	if err != nil {
		return nil, err
	}
	return &types.AiWorkflowResp{Workflow: mapAiWorkflow(out.Workflow)}, nil
}
