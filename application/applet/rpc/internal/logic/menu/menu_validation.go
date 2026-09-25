package menulogic

import (
	"go-zero-admin/pkg/result/xerr"
	"strings"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	base "go-zero-admin/pkg/model"
	"gorm.io/gorm"
)

func validateMenu(db *gorm.DB, menu *pb.SysBaseMenu) error {
	if menu == nil || menu.Meta == nil || strings.TrimSpace(menu.Name) == "" || strings.TrimSpace(menu.Path) == "" || strings.TrimSpace(menu.Meta.Title) == "" {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单name、path和标题不能为空")
	}
	if strings.ContainsAny(menu.Path, "?#:\\") || strings.HasPrefix(menu.Path, "//") {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单路径无效")
	}
	for _, part := range strings.Split(menu.Path, "/") {
		if part == "." || part == ".." {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单路径不能包含.或..段")
		}
	}
	reserved := map[string]bool{"AccessError": true, "Authentication": true, "BusinessSessionHome": true, "FallbackNotFound": true, "Login": true, "Root": true}
	if reserved[menu.Name] {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单name与系统路由冲突")
	}
	if err := accessutil.Unique(db, &model.SysBaseMenu{}, "id <> ? AND (name = ? OR path = ?)", menu.ID, menu.Name, menu.Path); err != nil {
		return err
	}
	var all []model.SysBaseMenu
	if err := db.Find(&all).Error; err != nil {
		return err
	}
	parents := map[int64]int64{}
	for _, value := range all {
		parents[value.ID] = value.ParentId
	}
	if err := accessutil.ValidateParent(menu.ID, menu.ParentId, parents); err != nil {
		return err
	}
	if err := validateResolvedPaths(all, menu); err != nil {
		return err
	}
	names := map[string]bool{}
	for _, button := range menu.MenuBtn {
		if button == nil || strings.TrimSpace(button.Name) == "" || strings.Contains(button.Name, ":") || names[button.Name] {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "按钮name不能为空、重复或包含冒号")
		}
		names[button.Name] = true
		if button.ID > 0 {
			var old model.SysBaseMenuBtn
			if err := accessutil.RequireID(db, &old, button.ID); err != nil {
				return err
			}
			if old.SysBaseMenuID != menu.ID {
				return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "按钮不属于当前菜单")
			}
		}
	}
	for _, parameter := range menu.Parameters {
		if parameter == nil || parameter.Key == "" || (parameter.Type != "query" && parameter.Type != "params") {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单参数须包含key且type为query或params")
		}
		if parameter.ID > 0 {
			var old model.SysBaseMenuParameter
			if err := accessutil.RequireID(db, &old, parameter.ID); err != nil {
				return err
			}
			if old.SysBaseMenuID != menu.ID {
				return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "参数不属于当前菜单")
			}
		}
	}
	return nil
}

// Check final URLs, including descendants moved by a parent edit.
func validateResolvedPaths(all []model.SysBaseMenu, value *pb.SysBaseMenu) error {
	byID := map[int64]model.SysBaseMenu{}
	for _, menu := range all {
		byID[menu.ID] = menu
	}
	id := value.ID
	if id == 0 {
		id = -1
	}
	byID[id] = model.SysBaseMenu{ParentId: value.ParentId, Path: value.Path}
	resolved := map[int64]string{}
	visiting := map[int64]bool{}
	var resolve func(int64) (string, error)
	resolve = func(id int64) (string, error) {
		if id == 0 {
			return "", nil
		}
		if path, ok := resolved[id]; ok {
			return path, nil
		}
		menu, ok := byID[id]
		if !ok || visiting[id] {
			return "", xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单父节点不存在或层级循环")
		}
		visiting[id] = true
		path := menu.Path
		if !strings.HasPrefix(path, "/") {
			parent, err := resolve(menu.ParentId)
			if err != nil {
				return "", err
			}
			path = parent + "/" + path
		}
		for strings.Contains(path, "//") {
			path = strings.ReplaceAll(path, "//", "/")
		}
		path = strings.TrimSuffix(path, "/")
		if path == "" || strings.HasPrefix(path, "/auth") || strings.HasPrefix(path, "/_session") {
			return "", xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单路径与系统路由冲突")
		}
		visiting[id] = false
		resolved[id] = path
		return path, nil
	}
	paths := map[string]bool{}
	for id := range byID {
		path, err := resolve(id)
		if err != nil {
			return err
		}
		if paths[path] {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单完整路径重复")
		}
		paths[path] = true
	}
	return nil
}

func menuModel(menu *pb.SysBaseMenu) model.SysBaseMenu {
	return model.SysBaseMenu{
		MODEL_BASE: base.MODEL_BASE{ID: menu.ID}, ParentId: menu.ParentId, Path: menu.Path, Name: menu.Name, Hidden: menu.Hidden, Component: menu.Component, Sort: menu.Sort,
		Meta: model.Meta{ActiveName: menu.Meta.ActiveName, KeepAlive: menu.Meta.KeepAlive, DefaultMenu: menu.Meta.DefaultMenu, Title: menu.Meta.Title, Icon: menu.Meta.Icon, CloseTab: menu.Meta.CloseTab},
	}
}

// Keep retained button IDs stable so editing a menu does not lose existing grants.
func saveMenuRelations(tx *gorm.DB, id int64, menu *pb.SysBaseMenu) error {
	keep := make([]int64, 0, len(menu.MenuBtn))
	for _, button := range menu.MenuBtn {
		if button.ID > 0 {
			keep = append(keep, button.ID)
		}
	}
	removed := tx.Where("sys_menu_id = ?", id)
	if len(keep) > 0 {
		removed = removed.Where("sys_base_menu_btn_id NOT IN ?", keep)
	}
	if err := removed.Delete(&model.SysAuthorityBtn{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("sys_base_menu_id = ?", id).Delete(&model.SysBaseMenuParameter{}).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("sys_base_menu_id = ?", id).Delete(&model.SysBaseMenuBtn{}).Error; err != nil {
		return err
	}
	for _, parameter := range menu.Parameters {
		value := model.SysBaseMenuParameter{MODEL_BASE: base.MODEL_BASE{ID: parameter.ID}, SysBaseMenuID: id, Type: parameter.Type, Key: parameter.Key, Value: parameter.Value}
		if err := tx.Create(&value).Error; err != nil {
			return err
		}
	}
	for _, button := range menu.MenuBtn {
		value := model.SysBaseMenuBtn{MODEL_BASE: base.MODEL_BASE{ID: button.ID}, SysBaseMenuID: id, Name: button.Name, Desc: button.Desc}
		if err := tx.Create(&value).Error; err != nil {
			return err
		}
	}
	return nil
}
