package role

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRoleListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRoleListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRoleListLogic {
	return &GetRoleListLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetRoleListLogic) GetRoleList(req *types.GetRoleListReq) (resp *types.GetRoleListResp, err error) {
	rpcResp, err := l.svcCtx.RoleRpc.GetRoleList(l.ctx, &adminpb.GetRoleListRequest{
		Page: &adminpb.PageRequest{Page: req.Page, PageSize: req.PageSize},
		Name: req.Name, Code: req.Code, Status: req.Status,
	})
	if err != nil {
		return nil, err
	}
	list := make([]*types.RoleInfo, 0, len(rpcResp.List))
	for _, r := range rpcResp.List {
		list = append(list, mapRoleInfo(r))
	}
	return &types.GetRoleListResp{List: list, Total: rpcResp.Page.Total}, nil
}

func mapRoleInfo(r *adminpb.RoleInfo) *types.RoleInfo {
	return &types.RoleInfo{
		Id: r.Id, Name: r.Name, Code: r.Code, Description: r.Description,
		DataScope: r.DataScope, Sort: r.Sort, Status: r.Status,
		MenuIds: r.MenuIds, UserCount: r.UserCount,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}
