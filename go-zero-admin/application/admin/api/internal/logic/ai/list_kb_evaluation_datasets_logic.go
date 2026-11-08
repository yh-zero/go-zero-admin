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

type ListKbEvaluationDatasetsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListKbEvaluationDatasetsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbEvaluationDatasetsLogic {
	return &ListKbEvaluationDatasetsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListKbEvaluationDatasetsLogic) ListKbEvaluationDatasets(req *types.AiIdReq) (resp *types.AiKbEvaluationDatasetsResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ListKbEvaluationDatasets(l.ctx, &aipb.ListKbEvaluationDatasetsRequest{Actor: actor, KbId: req.Id})
	if err != nil {
		return nil, err
	}
	list := make([]*types.AiKbEvaluationDatasetInfo, 0, len(out.GetList()))
	for _, row := range out.GetList() {
		list = append(list, &types.AiKbEvaluationDatasetInfo{Key: row.Key, Name: row.Name, Sha256: row.Sha256, Dataset: row.Dataset})
	}
	return &types.AiKbEvaluationDatasetsResp{List: list}, nil
}
