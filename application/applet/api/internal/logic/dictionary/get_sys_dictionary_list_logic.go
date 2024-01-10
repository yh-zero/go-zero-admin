package dictionary

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSysDictionaryListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSysDictionaryListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSysDictionaryListLogic {
	return &GetSysDictionaryListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSysDictionaryListLogic) GetSysDictionaryList(req *types.GetSysDictionaryListRequest) (resp *types.GetSysDictionaryListResponse, err error) {
	sysDictionaryList, err := l.svcCtx.AppletDictionaryRPC.GetSysDictionaryList(l.ctx, &pb.NoDataResponse{})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletDictionaryRPC.GetSysDictionaryList err:%v", err)
		return nil, err
	}
	var typesSysDictionaryList []types.SysDictionary
	_ = copier.Copy(&typesSysDictionaryList, sysDictionaryList.SysDictionaryList)

	return &types.GetSysDictionaryListResponse{
		List: typesSysDictionaryList,
	}, err
}
