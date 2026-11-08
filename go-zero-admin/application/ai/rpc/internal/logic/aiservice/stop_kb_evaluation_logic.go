package aiservicelogic

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type StopKbEvaluationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStopKbEvaluationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopKbEvaluationLogic {
	return &StopKbEvaluationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StopKbEvaluationLogic) StopKbEvaluation(in *aipb.StopKbEvaluationRequest) (*aipb.KbEvaluationResponse, error) {
	if err := requireAiActor(in.GetActor()); err != nil {
		return nil, err
	}
	if in.KbId <= 0 || in.EvaluationId <= 0 {
		return nil, xerr.NewErrCode(400)
	}
	if err := checkEvaluationKb(l.ctx, l.svcCtx, in.Actor, in.KbId); err != nil {
		return nil, err
	}
	row, err := l.svcCtx.AiKbEvaluationModel.FindOwned(l.ctx, in.Actor.UserId, in.KbId, in.EvaluationId)
	if err != nil {
		return nil, evaluationError(err)
	}
	if _, err = refreshEvaluationExpiry(l.ctx, l.svcCtx, row); err != nil {
		return nil, evaluationError(err)
	}
	row, err = l.svcCtx.AiKbEvaluationModel.StopOwned(l.ctx, in.Actor.UserId, in.KbId, in.EvaluationId)
	if err != nil {
		return nil, evaluationError(err)
	}
	return &aipb.KbEvaluationResponse{Evaluation: evaluationProto(row, false)}, nil
}
