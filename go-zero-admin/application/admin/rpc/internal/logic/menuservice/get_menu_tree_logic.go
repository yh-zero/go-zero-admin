package menuservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetMenuTreeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMenuTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuTreeLogic {
	return &GetMenuTreeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetMenuTreeLogic) GetMenuTree(in *adminpb.GetMenuTreeRequest) (*adminpb.GetMenuTreeResponse, error) {
	where := "deleted_at IS NULL"
	var args []interface{}
	if in.Type > 0 {
		where += " AND type=?"
		args = append(args, in.Type)
	}
	if in.Status > 0 {
		where += " AND status=?"
		args = append(args, in.Status)
	}
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
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &menuRows,
		"SELECT id, parent_id, title, name, type, path, component, perm_code, icon, sort, visible, status, created_at FROM sys_menu WHERE "+where+" ORDER BY sort, id",
		args...)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.MenuInfo, 0, len(menuRows))
	for _, r := range menuRows {
		list = append(list, &adminpb.MenuInfo{Id: r.Id, ParentId: r.ParentId, Title: r.Title, Name: r.Name, Type: r.Type, Path: r.Path, Component: r.Component, PermCode: r.PermCode, Icon: r.Icon, Sort: r.Sort, Visible: r.Visible, Status: r.Status, CreatedAt: r.CreatedAt})
	}
	return &adminpb.GetMenuTreeResponse{List: list}, nil
}
