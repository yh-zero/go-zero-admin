package roleservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetRoleListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRoleListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRoleListLogic {
	return &GetRoleListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetRoleListLogic) GetRoleList(in *adminpb.GetRoleListRequest) (*adminpb.GetRoleListResponse, error) {
	rows, total, err := l.svcCtx.SysRoleModel.ListRolePage(l.ctx, model.RoleListFilter{
		Name: in.Name, Code: in.Code, Status: int64(in.Status),
	}, in.GetPage().GetPage(), in.GetPage().GetPageSize())
	if err != nil {
		l.Errorf("query roles error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	// 统计字段：user_count 来自 casbin g 规则，menu_ids 来自 sys_role_menu
	codes := make([]string, 0, len(rows))
	roleIds := make([]int64, 0, len(rows))
	for _, r := range rows {
		codes = append(codes, r.Code)
		roleIds = append(roleIds, r.Id)
	}
	userCounts, err := l.svcCtx.SysRoleModel.CountUsersByRoleCodes(l.ctx, codes)
	if err != nil {
		l.Errorf("count role users error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	// 批量取菜单授权，避免逐角色 N+1
	menuIdsMap, err := l.svcCtx.SysRoleModel.ListMenuIdsByRoleIds(l.ctx, roleIds)
	if err != nil {
		l.Errorf("query role menus error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.RoleInfo, 0, len(rows))
	for _, r := range rows {
		menuIds := menuIdsMap[r.Id]
		if menuIds == nil {
			menuIds = []int64{}
		}
		list = append(list, &adminpb.RoleInfo{
			Id: r.Id, Name: r.Name, Code: r.Code, Description: r.Description,
			DataScope: r.DataScope, Sort: r.Sort, Status: r.Status,
			MenuIds: menuIds, UserCount: userCounts[r.Code],
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	return &adminpb.GetRoleListResponse{List: list, Page: &adminpb.PageResponse{Total: total}}, nil
}
