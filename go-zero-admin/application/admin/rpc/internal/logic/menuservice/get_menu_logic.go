package menuservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

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
	row, err := l.svcCtx.SysMenuModel.FindMenuRow(l.ctx, uint64(in.MenuId))
	if errors.Is(err, sqlx.ErrNotFound) {
		return nil, xerr.NewErrCode(xerr.MENU_NOT_FOUND)
	}
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.GetMenuResponse{Menu: menuRowToPb(row)}, nil
}
