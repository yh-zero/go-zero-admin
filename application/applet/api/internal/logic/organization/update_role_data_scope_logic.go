// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package organization

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateRoleDataScopeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 设置角色数据范围
func NewUpdateRoleDataScopeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRoleDataScopeLogic {
	return &UpdateRoleDataScopeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateRoleDataScopeLogic) UpdateRoleDataScope(req *types.RoleDataScope) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletOrganizationRPC.UpdateRoleDataScope(l.ctx, &pb.RoleDataScopeRequest{DataScope: &pb.RoleDataScope{AuthorityId: req.AuthorityId, Scope: req.Scope, DepartmentIds: req.DepartmentIds}})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "更新成功"}, nil
}
