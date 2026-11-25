package roleservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
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
	if err := requireRoleMutationAllowed(l.ctx, l.svcCtx, in.GetActor(), roleMutationInput{RoleID: uint64(in.RoleId), MenuIDs: in.MenuIds}); err != nil {
		return nil, err
	}
	if in.ExpectedMenus == nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	err := l.svcCtx.SysRoleModel.ReplaceMenus(l.ctx, uint64(in.RoleId), in.ExpectedMenus.MenuIds, in.MenuIds)
	if err != nil {
		return nil, roleMutationError(err)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
