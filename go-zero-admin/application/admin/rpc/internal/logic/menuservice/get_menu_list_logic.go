package menuservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetMenuListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMenuListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuListLogic {
	return &GetMenuListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetMenuListLogic) GetMenuList(in *adminpb.GetMenuListRequest) (*adminpb.GetMenuListResponse, error) {
	rows, total, err := l.svcCtx.SysMenuModel.ListMenuPage(l.ctx, in.ParentId, int64(in.Type), in.GetPage().GetPage(), in.GetPage().GetPageSize())
	if err != nil {
		l.Errorf("list menus error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.GetMenuListResponse{List: menuRowsToPb(rows), Page: &adminpb.PageResponse{Total: total}}, nil
}
