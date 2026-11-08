package aiservicelogic

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKbEvaluationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListKbEvaluationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbEvaluationsLogic {
	return &ListKbEvaluationsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListKbEvaluationsLogic) ListKbEvaluations(in *aipb.ListKbEvaluationsRequest) (*aipb.ListKbEvaluationsResponse, error) {
	if err := checkEvaluationKb(l.ctx, l.svcCtx, in.GetActor(), in.GetKbId()); err != nil {
		return nil, err
	}
	page, size, err := aiPagination(in.Page)
	if err != nil {
		return nil, err
	}
	status := evaluationListStatus(in.Status)
	if !evaluationStatus(status) || (in.RequestId != "" && !evaluationUUID(in.RequestId)) {
		return nil, xerr.NewErrCode(400)
	}
	rows, total, err := l.svcCtx.AiKbEvaluationModel.ListOwned(l.ctx, in.Actor.UserId, in.KbId, page, size, status, in.RequestId)
	if err != nil {
		return nil, evaluationError(err)
	}
	expired := false
	for _, row := range rows {
		if evaluationNeedsExpiry(row) {
			if _, err = refreshEvaluationExpiry(l.ctx, l.svcCtx, row); err != nil {
				return nil, evaluationError(err)
			}
			expired = true
		}
	}
	// 惰性过期可能使记录离开当前状态筛选；总数与当前页一起重读。
	if expired {
		rows, total, err = l.svcCtx.AiKbEvaluationModel.ListOwned(l.ctx, in.Actor.UserId, in.KbId, page, size, status, in.RequestId)
		if err != nil {
			return nil, evaluationError(err)
		}
	}
	list := make([]*aipb.KbEvaluationInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, evaluationProto(row, true))
	}
	return &aipb.ListKbEvaluationsResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
