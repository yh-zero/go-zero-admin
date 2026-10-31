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
	if role.DeletedAt.Valid {
		return nil, xerr.NewErrCode(xerr.ROLE_NOT_FOUND)
	}
	info := &adminpb.RoleInfo{
		Id: int64(role.Id), Name: role.Name, Code: role.Code,
		Description: role.Description, DataScope: role.DataScope,
		Sort: int32(role.Sort), Status: int32(role.Status),
		CreatedAt: role.CreatedAt.Unix(), UpdatedAt: role.UpdatedAt.Unix(),
	}
	// 菜单授权（无效行随菜单软删自动隐藏）
	menuIds, err := l.svcCtx.SysRoleModel.ListMenuIdsByRole(l.ctx, int64(role.Id))
	if err != nil {
		l.Errorf("query role menus error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	info.MenuIds = menuIds
	// 用户统计与列表口径一致（casbin g 规则按 user: 前缀计数）
	userCounts, err := l.svcCtx.SysRoleModel.CountUsersByRoleCodes(l.ctx, []string{role.Code})
	if err != nil {
		l.Errorf("count role users error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	info.UserCount = userCounts[role.Code]
	return &adminpb.GetRoleResponse{Role: info}, nil
}
