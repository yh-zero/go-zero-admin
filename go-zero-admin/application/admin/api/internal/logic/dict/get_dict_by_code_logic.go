package dict

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDictByCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDictByCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDictByCodeLogic {
	return &GetDictByCodeLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetDictByCodeLogic) GetDictByCode(req *types.GetDictByCodeReq) (resp *types.GetDictByCodeResp, err error) {
	rpcResp, err := l.svcCtx.DictRpc.GetDictByCode(l.ctx, &adminpb.GetDictByCodeRequest{Code: req.Code})
	if err != nil {
		return nil, err
	}
	return &types.GetDictByCodeResp{Dict: mapDictInfo(rpcResp.Dict)}, nil
}
