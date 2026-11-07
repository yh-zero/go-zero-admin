package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetKbLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetKbLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKbLogic {
	return &GetKbLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetKbLogic) GetKb(in *aipb.GetKbRequest) (*aipb.GetKbResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	row, err := l.svcCtx.AiKbModel.FindOwned(l.ctx, in.KbId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.GetKbResponse{Kb: kbRowToPb(row), Capabilities: capabilitiesProto(l.svcCtx)}, nil
}
