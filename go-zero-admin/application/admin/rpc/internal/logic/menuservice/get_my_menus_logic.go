package menuservicelogic

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetMyMenusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyMenusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyMenusLogic {
	return &GetMyMenusLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetMyMenusLogic) GetMyMenus(in *adminpb.GetMyMenusRequest) (*adminpb.GetMyMenusResponse, error) {
	// 查用户的角色（casbin g 规则 v0=user:N）
	var roleCodes []string
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &roleCodes,
		"SELECT v1 FROM casbin_rule WHERE ptype='g' AND v0=?", fmt.Sprintf("user:%d", in.Actor.UserId))
	if err != nil || len(roleCodes) == 0 {
		return &adminpb.GetMyMenusResponse{List: []*adminpb.MenuInfo{}}, nil
	}
	// 角色的菜单 IDs
	type menuRow struct {
		Id        int64  `db:"id"`
		ParentId  int64  `db:"parent_id"`
		Title     string `db:"title"`
		Name      string `db:"name"`
		Type      int32  `db:"type"`
		Path      string `db:"path"`
		Component string `db:"component"`
		PermCode  string `db:"perm_code"`
		Icon      string `db:"icon"`
		Sort      int32  `db:"sort"`
		Visible   bool   `db:"visible"`
		Status    int32  `db:"status"`
		CreatedAt int64  `db:"created_at"`
	}
	var menuRows []menuRow
	err = l.svcCtx.DB.QueryRowsCtx(l.ctx, &menuRows,
		"SELECT DISTINCT m.id, m.parent_id, m.title, m.name, m.type, m.path, m.component, m.perm_code, m.icon, m.sort, m.visible, m.status, m.created_at "+
			"FROM sys_menu m JOIN sys_role_menu rm ON m.id=rm.menu_id JOIN sys_role r ON rm.role_id=r.id "+
			"WHERE r.code IN (?) AND m.deleted_at IS NULL AND r.deleted_at IS NULL ORDER BY m.sort, m.id",
		roleCodes)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.MenuInfo, 0, len(menuRows))
	for _, r := range menuRows {
		list = append(list, &adminpb.MenuInfo{Id: r.Id, ParentId: r.ParentId, Title: r.Title, Name: r.Name, Type: r.Type, Path: r.Path, Component: r.Component, PermCode: r.PermCode, Icon: r.Icon, Sort: r.Sort, Visible: r.Visible, Status: r.Status, CreatedAt: r.CreatedAt})
	}
	return &adminpb.GetMyMenusResponse{List: list}, nil
}
