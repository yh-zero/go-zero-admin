package menuservicelogic

import (
	"context"

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
	roleCodes, err := l.svcCtx.SysMenuModel.ListRoleCodesByUser(l.ctx, in.Actor.UserId)
	if err != nil {
		l.Errorf("query user roles error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if len(roleCodes) == 0 {
		return &adminpb.GetMyMenusResponse{List: []*adminpb.MenuInfo{}}, nil
	}
	rows, err := l.svcCtx.SysMenuModel.ListMenusByRoleCodes(l.ctx, roleCodes)
	if err != nil {
		l.Errorf("query role menus error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.GetMyMenusResponse{List: menuRowsToPb(rows)}, nil
}
