package ai

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"
)

type GetKbLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetKbLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKbLogic {
	return &GetKbLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetKbLogic) GetKb(req *types.AiIdReq) (*types.AiKbResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.GetKb(l.ctx, &aipb.GetKbRequest{Actor: actor, KbId: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.AiKbResp{Kb: mapAiKb(out.Kb), Capabilities: capabilitiesInfo(out.Capabilities)}, nil
}
