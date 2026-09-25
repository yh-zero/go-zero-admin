package menulogic

import "go-zero-admin/application/applet/rpc/internal/model"

func baseMenuTree(all []model.SysBaseMenu, parent int64) []model.SysBaseMenu {
	result := make([]model.SysBaseMenu, 0)
	for _, menu := range all {
		if menu.ParentId == parent {
			menu.Children = baseMenuTree(all, menu.ID)
			result = append(result, menu)
		}
	}
	return result
}
