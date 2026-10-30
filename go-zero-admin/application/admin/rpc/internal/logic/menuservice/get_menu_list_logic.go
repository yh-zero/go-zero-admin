package menuservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
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
	page := in.GetPage()
	rows, total, err := l.svcCtx.SysMenuModel.FindList(l.ctx, in.ParentId, in.Type, page.GetPage(), page.GetPageSize())
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.GetMenuListResponse{List: menuInfoList(rows), Page: &adminpb.PageResponse{Total: total}}, nil
}

func menuInfoList(rows []model.MenuRow) []*adminpb.MenuInfo {
	list := make([]*adminpb.MenuInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, &adminpb.MenuInfo{Id: row.Id, ParentId: row.ParentId, Title: row.Title,
			Name: row.Name, Type: row.Type, Path: row.Path, Component: row.Component, PermCode: row.PermCode,
			Icon: row.Icon, Sort: row.Sort, Visible: row.Visible, Status: row.Status, CreatedAt: row.CreatedAt})
	}
	return list
}
