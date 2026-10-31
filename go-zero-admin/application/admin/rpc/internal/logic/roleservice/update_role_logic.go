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
	err := l.svcCtx.SysRoleModel.UpdateWithMenus(l.ctx, &model.SysRole{
		Id: uint64(in.RoleId), Name: in.Name, Code: in.Code, Description: in.Description,
		DataScope: in.DataScope, Sort: int64(in.Sort), Status: int64(in.Status),
	}, in.MenuIds)
	if err != nil {
		return nil, roleMutationError(err)
	}
	// 状态变更会影响有效策略，保存后广播刷新。
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
