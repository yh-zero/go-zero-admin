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

type UpdateKbDocLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateKbDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateKbDocLogic {
	return &UpdateKbDocLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateKbDocLogic) UpdateKbDoc(req *types.AiUpdateKbDocReq) (resp *types.AiKbDocResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.UpdateKbDoc(l.ctx, &aipb.UpdateKbDocRequest{Actor: actor, KbId: req.Id, DocId: req.DocId, Title: req.Title, Content: req.Content, ExpectedVersion: req.ExpectedVersion, RequestId: req.RequestId, BatchId: req.BatchId})
	if err != nil {
		return nil, err
	}
	return &types.AiKbDocResp{Doc: mapAiKbDoc(out.Doc), JobId: out.JobId, BatchId: out.BatchId}, nil
}
