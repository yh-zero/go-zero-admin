package roleservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
)

type UpdateRoleMenusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateRoleMenusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRoleMenusLogic {
	return &UpdateRoleMenusLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateRoleMenusLogic) UpdateRoleMenus(in *adminpb.UpdateRoleMenusRequest) (*adminpb.NoDataResponse, error) {
	if err := requireRoleActor(l.ctx, l.svcCtx, in.GetActor()); err != nil {
		return nil, err
	}
	err := l.svcCtx.SysRoleModel.ReplaceMenus(l.ctx, uint64(in.RoleId), in.MenuIds)
	if err != nil {
		return nil, roleMutationError(err)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
