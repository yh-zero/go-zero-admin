package aiservicelogic

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetKbEvaluationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetKbEvaluationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKbEvaluationLogic {
	return &GetKbEvaluationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetKbEvaluationLogic) GetKbEvaluation(in *aipb.GetKbEvaluationRequest) (*aipb.KbEvaluationResponse, error) {
	if err := requireAiActor(in.GetActor()); err != nil {
		return nil, err
	}
	if in.KbId <= 0 || in.EvaluationId <= 0 || in.KnownObservationsSha256 != "" && (!in.ProgressOnly || !evaluationSHA.MatchString(in.KnownObservationsSha256)) {
		return nil, xerr.NewErrCode(400)
	}
	if err := checkEvaluationKb(l.ctx, l.svcCtx, in.Actor, in.KbId); err != nil {
		return nil, err
	}
	if in.ProgressOnly {
		store := l.svcCtx.AiKbEvaluationModel
		progress, err := store.ProgressOwned(l.ctx, in.Actor.UserId, in.KbId, in.EvaluationId, in.KnownObservationsSha256)
		if err != nil {
			return nil, evaluationError(err)
		}
		if evaluationNeedsExpiry(progress) {
			if _, err = store.ExpireOwned(l.ctx, progress.OwnerId, progress.KbId, progress.Id); err != nil {
				return nil, evaluationError(err)
			}
			progress, err = store.ProgressOwned(l.ctx, in.Actor.UserId, in.KbId, in.EvaluationId, in.KnownObservationsSha256)
			if err != nil {
				return nil, evaluationError(err)
			}
		}
		return &aipb.KbEvaluationResponse{Evaluation: evaluationProto(progress, false)}, nil
	}
	row, err := l.svcCtx.AiKbEvaluationModel.FindOwned(l.ctx, in.Actor.UserId, in.KbId, in.EvaluationId)
	if err != nil {
		return nil, evaluationError(err)
	}
	row, err = refreshEvaluationExpiry(l.ctx, l.svcCtx, row)
	if err != nil {
		return nil, evaluationError(err)
	}
	return &aipb.KbEvaluationResponse{Evaluation: evaluationProto(row, false)}, nil
}
