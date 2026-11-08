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

type SetTaskFeedbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetTaskFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetTaskFeedbackLogic {
	return &SetTaskFeedbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetTaskFeedbackLogic) SetTaskFeedback(req *types.AiTaskFeedbackReq) (resp *types.AiKbFeedbackResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.SetTaskFeedback(l.ctx, &aipb.SetTaskFeedbackRequest{Actor: actor, TaskId: req.Id, StepIndex: req.StepIndex, ExpectedRevision: req.ExpectedRevision, Feedback: req.Feedback, Note: req.Note})
	if err != nil {
		return nil, err
	}
	return &types.AiKbFeedbackResp{Feedback: mapKbFeedback(out.Feedback)}, nil
}
