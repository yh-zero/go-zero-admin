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

type GetRoleDataScopeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 读取角色数据范围
func NewGetRoleDataScopeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRoleDataScopeLogic {
	return &GetRoleDataScopeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetRoleDataScopeLogic) GetRoleDataScope(req *types.GetRoleDataScopeRequest) (resp *types.RoleDataScope, err error) {
	result, err := l.svcCtx.AppletOrganizationRPC.GetRoleDataScope(l.ctx, &pb.GetRoleDataScopeRequest{AuthorityId: req.AuthorityId})
	if err != nil {
		return nil, err
	}
	return &types.RoleDataScope{AuthorityId: result.DataScope.AuthorityId, Scope: result.DataScope.Scope, DepartmentIds: append([]int64{}, result.DataScope.DepartmentIds...)}, nil
}
