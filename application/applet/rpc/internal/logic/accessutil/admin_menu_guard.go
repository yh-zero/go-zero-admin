package accessutil

import (
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

type adminMenuAccess struct {
	Visible bool
	Buttons map[string]bool
}

// Protect the registered recovery page and its permission-editor buttons. Old
// installations with missing UI grants may still add them gradually.
func adminMenuState(tx *gorm.DB) (adminMenuAccess, error) {
	state := adminMenuAccess{Buttons: map[string]bool{}}
	exists, err := adminRoleExists(tx)
	if err != nil || !exists {
		return state, err
	}
	var menus []model.SysBaseMenu
	if err := tx.Find(&menus).Error; err != nil {
		return state, err
	}
	var ids []int64
	if err := tx.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ?", AdminAuthorityID).Pluck("sys_base_menu_id", &ids).Error; err != nil {
		return state, err
	}
	assigned := map[int64]bool{}
	for _, id := range ids {
		assigned[id] = true
	}
	byID := map[int64]model.SysBaseMenu{}
	var recovery model.SysBaseMenu
	for _, menu := range menus {
		byID[menu.ID] = menu
		if menu.Name == "authority" && menu.Component == "views/superAdmin/authority/authority.vue" {
			recovery = menu
		}
	}
	if recovery.ID == 0 {
		return state, nil
	}
	visited := map[int64]bool{}
	for id := recovery.ID; id != 0; {
		menu, ok := byID[id]
		if !ok || !assigned[id] || menu.Hidden || visited[id] {
			return state, nil
		}
		visited[id] = true
		id = menu.ParentId
	}
	// A menu with visible routed children renders a directory rather than its page.
	for _, menu := range menus {
		if menu.ParentId == recovery.ID && assigned[menu.ID] {
			return state, nil
		}
	}
	state.Visible = true
	var buttons []model.SysBaseMenuBtn
	err = tx.Where("sys_base_menu_id = ? AND name IN ?", recovery.ID, []string{"menus", "buttons", "apis"}).
		Where("EXISTS (SELECT 1 FROM sys_authority_btns b WHERE b.authority_id = ? AND b.sys_menu_id = ? AND b.sys_base_menu_btn_id = sys_base_menu_btns.id)", AdminAuthorityID, recovery.ID).Find(&buttons).Error
	if err != nil {
		return state, err
	}
	for _, button := range buttons {
		state.Buttons[button.Name] = true
	}
	return state, nil
}

// Serialize menu and button changes with role/user/policy updates, then compare
// recoverable capabilities before committing. This also covers editing the page
// component, hiding its ancestor, and removing a button definition.
func AdminMenuTransaction(db *gorm.DB, change func(*gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := LockAdminGuard(tx); err != nil {
			return err
		}
		before, err := adminMenuState(tx)
		if err != nil {
			return err
		}
		if err := change(tx); err != nil {
			return err
		}
		after, err := adminMenuState(tx)
		if err != nil {
			return err
		}
		if before.Visible && !after.Visible {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "管理员必须保留可访问的角色管理菜单")
		}
		for name := range before.Buttons {
			if !after.Buttons[name] {
				return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "管理员必须保留角色管理授权按钮："+name)
			}
		}
		return nil
	})
}
