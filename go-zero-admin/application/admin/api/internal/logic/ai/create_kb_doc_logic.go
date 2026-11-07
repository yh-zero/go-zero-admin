package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"
)

type CreateKbDocLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateKbDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateKbDocLogic {
	return &CreateKbDocLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *CreateKbDocLogic) CreateKbDoc(req *types.AiCreateKbDocReq) (*types.AiKbDocResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.CreateKbDoc(l.ctx, &aipb.CreateKbDocRequest{Actor: actor, KbId: req.Id, Title: req.Title, Content: req.Content, RequestId: req.RequestId, BatchId: req.BatchId})
	if err != nil {
		return nil, err
	}
	return &types.AiKbDocResp{Doc: mapAiKbDoc(out.Doc), JobId: out.JobId, BatchId: out.BatchId}, nil
}
