package api

import (
	"context"
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

func (l *GetApiListLogic) GetApiList(req *types.GetApiListRequest) (resp *types.GetApiListResponse, err error) {
	if req.PageNo == 0 {
		req.PageNo = l.svcCtx.Config.Page.PageNo
	}
	if req.PageSize == 0 {
		req.PageSize = l.svcCtx.Config.Page.PageSize
	}

	var pbPageRequest pb.PageRequest
	_ = copier.Copy(&pbPageRequest, req.PageRequest)

	var pbSysApi pb.SysApi
	_ = copier.Copy(&pbSysApi, req.SysApi)

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
