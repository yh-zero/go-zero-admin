package api

import (
	"context"
	"fmt"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
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

// 前端请求 /v1/sys/api/getApiList?pageNo=11111&=222222
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

	var pbPageRequest pb.PageRequest
	_ = copier.Copy(&pbPageRequest, req.PageRequest)

	var pbSysApi pb.SysApi
	_ = copier.Copy(&pbSysApi, req.SysApi)

	var pbGetApiListReq pb.GetApiListRequest
	pbGetApiListReq.PageRequest = &pbPageRequest
	pbGetApiListReq.SysApi = &pbSysApi

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
