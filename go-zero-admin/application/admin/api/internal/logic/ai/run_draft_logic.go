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

type RunDraftLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRunDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RunDraftLogic {
	return &RunDraftLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RunDraftLogic) RunDraft(req *types.AiRunDraftReq) (resp *types.AiRunWorkflowResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.RunDraft(l.ctx, &aipb.RunDraftRequest{Actor: actor, Name: req.Name, Definition: req.Definition, Input: req.Input, RequestId: req.RequestId})
	if err != nil {
		return nil, err
	}
	return &types.AiRunWorkflowResp{TaskId: out.TaskId, Status: out.Status}, nil
}
