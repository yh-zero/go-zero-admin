package dictionary

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSysDictionaryInfoListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetSysDictionaryInfoListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSysDictionaryInfoListLogic {
	return &GetSysDictionaryInfoListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetSysDictionaryInfoListLogic) GetSysDictionaryInfoList(req *types.GetSysDictionaryInfoListRequest) (resp *types.GetSysDictionaryInfoListResponse, err error) {
	if req.PageNo == 0 {
		req.PageNo = l.svcCtx.Config.Page.PageNo
	}
	if req.PageSize == 0 {
		req.PageSize = l.svcCtx.Config.Page.PageSize
	}
	var pbSysDictionaryInfo pb.SysDictionaryInfo
	var pbPageRequest pb.PageRequest
	_ = copier.Copy(&pbSysDictionaryInfo, req.SysDictionaryInfo)
	_ = copier.Copy(&pbPageRequest, req.PageRequest)

	sysDictionaryInfoList, err := l.svcCtx.AppletDictionaryRPC.GetSysDictionaryInfoList(l.ctx, &pb.GetSysDictionaryInfoListRequest{
		SysDictionaryInfo: &pbSysDictionaryInfo,
		PageRequest:       &pbPageRequest,
	})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletDictionaryRPC.GetSysDictionaryInfoList err:%v", err)
		return nil, err
	}
	var typesSysDictionaryInfoList []types.SysDictionaryInfo
	_ = copier.Copy(&typesSysDictionaryInfoList, sysDictionaryInfoList.SysDictionaryInfoList)

	return &types.GetSysDictionaryInfoListResponse{
		List: typesSysDictionaryInfoList,
		PageResponse: types.PageResponse{
			Total:    sysDictionaryInfoList.Total,
			PageNo:   req.PageNo,
			PageSize: req.PageSize,
		},
	}, nil
}
