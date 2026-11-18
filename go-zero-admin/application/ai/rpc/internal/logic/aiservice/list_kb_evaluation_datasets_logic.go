package aiservicelogic

import (
	"context"
	"go-zero-admin/pkg/xerr"

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
	keys := evaluationDatasetKeys
	if in.DatasetKey != "" {
		if evaluationDatasetNames[in.DatasetKey] == "" {
			return nil, xerr.NewErrCode(400)
		}
		keys = []string{in.DatasetKey}
	}
	list := make([]*aipb.KbEvaluationDatasetInfo, 0, len(keys))
	for _, key := range keys {
		dataset, err := loadEvaluationDataset(key)
		if err != nil {
			l.Error("评测内置题集不可用")
			return nil, evaluationError(err)
		}
		item := &aipb.KbEvaluationDatasetInfo{Key: key, Name: evaluationDatasetNames[key], Sha256: dataset.SHA}
		if !in.MetaOnly {
			item.Dataset = dataset.Raw
		}
		list = append(list, item)
	}
	return &aipb.ListKbEvaluationDatasetsResponse{List: list}, nil
}
