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

type GetKbDocLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetKbDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetKbDocLogic {
	return &GetKbDocLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetKbDocLogic) GetKbDoc(req *types.AiGetKbDocReq) (resp *types.AiKbDocDetailResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.GetKbDoc(l.ctx, &aipb.GetKbDocRequest{Actor: actor, KbId: req.Id, DocId: req.DocId, MetaOnly: req.MetaOnly})
	if err != nil {
		return nil, err
	}
	return &types.AiKbDocDetailResp{Doc: mapAiKbDoc(out.Doc), Content: out.Content}, nil
}
