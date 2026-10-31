package menuservicelogic

import (
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
)

// menuRowToPb 统一 MenuRow → pb 映射，避免各 logic 重复
func menuRowToPb(r *model.MenuRow) *adminpb.MenuInfo {
	return &adminpb.MenuInfo{
		Id: r.Id, ParentId: r.ParentId, Title: r.Title, Name: r.Name, Type: r.Type,
		Path: r.Path, Component: r.Component, PermCode: r.PermCode, Icon: r.Icon,
		Sort: r.Sort, Visible: r.Visible, Status: r.Status, CreatedAt: r.CreatedAt,
	}
}

func menuRowsToPb(rows []*model.MenuRow) []*adminpb.MenuInfo {
	list := make([]*adminpb.MenuInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, menuRowToPb(r))
	}
	return list
}
