package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetKbDocLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetKbDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKbDocLogic {
	return &GetKbDocLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetKbDocLogic) GetKbDoc(in *aipb.GetKbDocRequest) (*aipb.GetKbDocResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	var row *model.KbDocRow
	var err error
	if in.MetaOnly {
		row, err = l.svcCtx.AiKbDocModel.FindMetadataOwned(l.ctx, in.KbId, in.DocId, in.Actor.UserId)
	} else {
		row, err = l.svcCtx.AiKbDocModel.FindOwned(l.ctx, in.KbId, in.DocId, in.Actor.UserId)
	}
	if err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.GetKbDocResponse{Doc: kbDocRowToPb(row), Content: row.Content}, nil
}
