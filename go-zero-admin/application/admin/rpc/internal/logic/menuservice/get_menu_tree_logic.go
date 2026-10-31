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
	rows, err := l.svcCtx.SysMenuModel.ListMenuTree(l.ctx, int64(in.Type), int64(in.Status))
	if err != nil {
		l.Errorf("menu tree error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.GetMenuTreeResponse{List: menuRowsToPb(rows)}, nil
}
