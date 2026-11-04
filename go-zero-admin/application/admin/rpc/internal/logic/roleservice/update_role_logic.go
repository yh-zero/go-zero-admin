package roleservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
)

type UpdateRoleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRoleLogic {
	return &UpdateRoleLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateRoleLogic) UpdateRole(in *adminpb.UpdateRoleRequest) (*adminpb.NoDataResponse, error) {
	if err := requireRoleActor(l.ctx, l.svcCtx, in.GetActor()); err != nil {
		return nil, err
	}
	err := l.svcCtx.SysRoleModel.UpdatePartial(l.ctx, uint64(in.RoleId), model.RolePatch{
		Name: in.Name, Code: in.Code, Description: in.Description, DataScope: in.DataScope,
		Sort: in.Sort, Status: in.Status, MenuIds: in.MenuIds, UpdateMenus: in.UpdateMenus,
	})
	if model.MutationCommitted(err) {
		l.svcCtx.PublishPolicyChanged(l.ctx)
	}
	if err != nil {
		return nil, roleMutationError(err)
	}
	// 状态变更会影响有效策略，保存后广播刷新。
	return &adminpb.NoDataResponse{}, nil
}
