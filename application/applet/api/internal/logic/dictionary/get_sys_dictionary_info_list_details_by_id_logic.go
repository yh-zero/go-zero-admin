package dictionary

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSysDictionaryInfoListDetailsByIdLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSysDictionaryInfoListDetailsByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSysDictionaryInfoListDetailsByIdLogic {
	return &GetSysDictionaryInfoListDetailsByIdLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSysDictionaryInfoListDetailsByIdLogic) GetSysDictionaryInfoListDetailsById(req *types.GetSysDictionaryInfoListDetailsByIdRequest) (resp *types.GetSysDictionaryInfoListDetailsByIdResponse, err error) {
	sysDictionaryInfoListDetailsById, err := l.svcCtx.AppletDictionaryRPC.GetSysDictionaryInfoListDetailsById(l.ctx, &pb.GetSysDictionaryInfoListDetailsByIdRequest{
		ID: req.ID,
	})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletDictionaryRPC.GetSysDictionaryInfoListDetailsById err:%v", err)
		return nil, err
	}
	var typesSysDictionaryInfo types.SysDictionaryInfo
	_ = copier.Copy(&typesSysDictionaryInfo, sysDictionaryInfoListDetailsById.SysDictionaryInfo)
	return &types.GetSysDictionaryInfoListDetailsByIdResponse{
		SysDictionaryInfo: typesSysDictionaryInfo,
	}, nil
}
