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

type GetKbEvaluationLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetKbEvaluationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKbEvaluationLogic {
	return &GetKbEvaluationLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetKbEvaluationLogic) GetKbEvaluation(req *types.AiKbEvaluationReq) (resp *types.AiKbEvaluationResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.GetKbEvaluation(l.ctx, &aipb.GetKbEvaluationRequest{Actor: actor, KbId: req.Id, EvaluationId: req.EvaluationId})
	if err != nil {
		return nil, err
	}
	return &types.AiKbEvaluationResp{Evaluation: mapKbEvaluation(out.GetEvaluation())}, nil
}
