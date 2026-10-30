package api

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetApiListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetApiListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetApiListLogic {
	return &GetApiListLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetApiListLogic) GetApiList(req *types.GetApiListReq) (resp *types.GetApiListResp, err error) {
	rpcResp, err := l.svcCtx.ApiRpc.GetApiList(l.ctx, &adminpb.GetApiListRequest{
		Page:  &adminpb.PageRequest{Page: req.Page, PageSize: req.PageSize},
		Group: req.Group, Method: req.Method, Path: req.Path,
	})
	if err != nil {
		return nil, err
	}
	list := make([]*types.ApiInfo, 0, len(rpcResp.List))
	for _, a := range rpcResp.List {
		list = append(list, mapApiInfo(a))
	}
	return &types.GetApiListResp{List: list, Total: rpcResp.Page.Total}, nil
}

func mapApiInfo(a *adminpb.ApiInfo) *types.ApiInfo {
	return &types.ApiInfo{Id: a.Id, Group: a.Group, Method: a.Method, Path: a.Path, Title: a.Title, CreatedAt: a.CreatedAt}
}
