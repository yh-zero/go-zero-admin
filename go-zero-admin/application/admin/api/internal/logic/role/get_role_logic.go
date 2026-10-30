package role

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRoleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRoleLogic {
	return &GetRoleLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetRoleLogic) GetRole(req *types.GetRoleReq) (resp *types.GetRoleResp, err error) {
	rpcResp, err := l.svcCtx.RoleRpc.GetRole(l.ctx, &adminpb.GetRoleRequest{RoleId: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.GetRoleResp{Role: mapRoleInfo(rpcResp.Role)}, nil
}

func mapRoleInfoR(r *adminpb.RoleInfo) *types.RoleInfo {
	return &types.RoleInfo{
		Id: r.Id, Name: r.Name, Code: r.Code, Description: r.Description,
		DataScope: r.DataScope, Sort: r.Sort, Status: r.Status,
		MenuIds: r.MenuIds, UserCount: r.UserCount,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}
