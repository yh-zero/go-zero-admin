package roleservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetRoleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRoleLogic {
	return &GetRoleLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetRoleLogic) GetRole(in *adminpb.GetRoleRequest) (*adminpb.GetRoleResponse, error) {
	role, err := l.svcCtx.SysRoleModel.FindOne(l.ctx, uint64(in.RoleId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.ROLE_NOT_FOUND)
	}
	info := &adminpb.RoleInfo{
		Id: int64(role.Id), Name: role.Name, Code: role.Code,
		Description: role.Description, DataScope: role.DataScope,
		Sort: int32(role.Sort), Status: int32(role.Status),
		CreatedAt: role.CreatedAt.Unix(), UpdatedAt: role.UpdatedAt.Unix(),
	}
	// 查 menu_ids
	var menuIDs []int64
	_ = l.svcCtx.DB.QueryRowsCtx(l.ctx, &menuIDs, "SELECT menu_id FROM sys_role_menu WHERE role_id=?", in.RoleId)
	info.MenuIds = menuIDs
	return &adminpb.GetRoleResponse{Role: info}, nil
}
