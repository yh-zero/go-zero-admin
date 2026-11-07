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

type StopKbImportLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewStopKbImportLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StopKbImportLogic {
	return &StopKbImportLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StopKbImportLogic) StopKbImport(req *types.AiKbImportReq) (resp *types.AiKbImportResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.StopKbImport(l.ctx, &aipb.StopKbImportRequest{Actor: actor, KbId: req.Id, BatchId: req.BatchId})
	if err != nil {
		return nil, err
	}
	return mapAiKbImport(out), nil
}
