package api

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetAllApiListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAllApiListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAllApiListLogic {
	return &GetAllApiListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAllApiListLogic) GetAllApiList(req *types.GetAllApiListRequest) (resp *types.GetAllApiListResponse, err error) {
	apiList, err := l.svcCtx.AppletAPIRPC.GetAllApiList(l.ctx, &pb.NoDataResponse{})
	if err != nil {
		logx.Errorf("----------- 获取失败! err:%v", err)
		return nil, err
	}
	resp = &types.GetAllApiListResponse{}
	_ = copier.Copy(&resp.ApiList, apiList.ApiList)

	return resp, err
}
