package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKbEvaluationDatasetsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListKbEvaluationDatasetsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbEvaluationDatasetsLogic {
	return &ListKbEvaluationDatasetsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListKbEvaluationDatasetsLogic) ListKbEvaluationDatasets(in *aipb.ListKbEvaluationDatasetsRequest) (*aipb.ListKbEvaluationDatasetsResponse, error) {
	if err := checkEvaluationKb(l.ctx, l.svcCtx, in.GetActor(), in.GetKbId()); err != nil {
		return nil, err
	}
	list := make([]*aipb.KbEvaluationDatasetInfo, 0, len(evaluationDatasetKeys))
	for _, key := range evaluationDatasetKeys {
		dataset, err := loadEvaluationDataset(key)
		if err != nil {
			l.Error("评测内置题集不可用")
			return nil, evaluationError(err)
		}
		list = append(list, &aipb.KbEvaluationDatasetInfo{Key: key, Name: evaluationDatasetNames[key], Sha256: dataset.SHA, Dataset: dataset.Raw})
	}
	return &aipb.ListKbEvaluationDatasetsResponse{List: list}, nil
}
