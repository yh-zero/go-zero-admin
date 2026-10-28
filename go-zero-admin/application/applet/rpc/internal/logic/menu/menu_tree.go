package menulogic

import "go-zero-admin/application/applet/rpc/internal/model"

func baseMenuTree(all []model.SysBaseMenu, parent int64) []model.SysBaseMenu {
	children := make(map[int64][]int, len(all))
	for i := range all {
		children[all[i].ParentId] = append(children[all[i].ParentId], i)
	}
	var build func(int64) []model.SysBaseMenu
	build = func(parentID int64) []model.SysBaseMenu {
		indices := children[parentID]
		result := make([]model.SysBaseMenu, 0, len(indices))
		for _, i := range indices {
			menu := all[i]
			menu.Children = build(menu.ID)
			result = append(result, menu)
		}
		return result
	}
	return build(parent)
}
