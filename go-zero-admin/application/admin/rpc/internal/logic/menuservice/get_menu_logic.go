package menuservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuLogic {
	return &GetMenuLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetMenuLogic) GetMenu(in *adminpb.GetMenuRequest) (*adminpb.GetMenuResponse, error) {
	var menu adminpb.MenuInfo
	err := l.svcCtx.DB.QueryRowCtx(l.ctx, &menu,
		"SELECT id, parent_id, title, name, type, path, component, perm_code, icon, sort, visible, status, created_at FROM sys_menu WHERE id=? AND deleted_at IS NULL",
		in.MenuId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.MENU_NOT_FOUND)
	}
	return &adminpb.GetMenuResponse{Menu: &menu}, nil
}
