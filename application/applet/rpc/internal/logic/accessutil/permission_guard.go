package accessutil

import (
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

func ValidateProtectedPermissionChange(tx *gorm.DB, change func(*gorm.DB) error) error {
	menu, err := adminMenuState(tx)
	if err != nil {
		return err
	}
	policies, err := adminRecoveryPolicyState(tx)
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
	if menu.Visible && !after.Visible {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "管理员必须保留可访问的角色管理菜单")
	}
	for name := range menu.Buttons {
		if !after.Buttons[name] {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "管理员必须保留角色管理授权按钮："+name)
		}
	}
	return preserveAdminRecoveryPolicies(tx, policies)
}
