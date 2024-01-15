package api

import (
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetApiListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetApiListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetApiListLogic {
	return &GetApiListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetApiListLogic) GetApiList(req *types.GetApiListRequest) (resp *types.GetApiListResponse, err error) {
	fmt.Println("req.PageNo = ", req.PageNo)     // req.PageNo =  0
	fmt.Println("req.PageSize = ", req.PageSize) // req.PageSize =  0
	if req.PageNo == 0 {
		req.PageNo = l.svcCtx.Config.Page.PageNo
	}
	if req.PageSize == 0 {
		req.PageSize = l.svcCtx.Config.Page.PageSize
	}

	fmt.Println("======== GetApiList req=", req)
	fmt.Println("======== GetApiList req.orderKey=", req.OrderKey)

	var pbPageRequest pb.PageRequest
	_ = copier.Copy(&pbPageRequest, req.PageRequest)

	var pbSysApi pb.SysApi
	_ = copier.Copy(&pbSysApi, req.SysApi)
	fmt.Println("======== pbSysApi111=", req.SysApi)

	var pbGetApiListReq pb.GetApiListRequest
	pbGetApiListReq.PageRequest = &pbPageRequest
	pbGetApiListReq.SysApi = &pbSysApi
	pbGetApiListReq.OrderKey = req.OrderKey
	pbGetApiListReq.Desc = req.Desc

	getApiList, err := l.svcCtx.AppletAPIRPC.GetApiList(l.ctx, &pbGetApiListReq)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}
	var typesSysApi []types.SysApi
	_ = copier.Copy(&typesSysApi, getApiList.SysApi)

	return &types.GetApiListResponse{
		List: typesSysApi,
		PageResponse: types.PageResponse{
			Total:    getApiList.Total,
			PageNo:   req.PageNo,
			PageSize: req.PageSize,
		},
	}, nil
}
