package accessutil

import (
	"errors"
	"regexp"
	"strconv"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/result/xerr"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdminAuthorityID is the built-in administrator role seeded by the database.
// It is a recovery role, not a username: administrators may rename their accounts.
const AdminAuthorityID int64 = 1

func adminRoleExists(tx *gorm.DB) (bool, error) {
	var count int64
	err := tx.Model(&model.SysAuthority{}).Where("authority_id = ? AND deleted_at IS NULL", AdminAuthorityID).Count(&count).Error
	return count > 0, err
}

// LockAdminGuard must be acquired before changing users or permission rules.
// Locking the same row serializes these transactions across RPC instances.
func LockAdminGuard(tx *gorm.DB) error {
	var role model.SysAuthority
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("authority_id = ? AND deleted_at IS NULL", AdminAuthorityID).First(&role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	} // Isolated fixtures without the built-in role.
	return err
}

func EnsureUsableAdministrator(tx *gorm.DB) error {
	exists, err := adminRoleExists(tx)
	if err != nil || !exists {
		return err
	}
	var count int64
	err = tx.Model(&model.SysUser{}).
		Where("enable = 1 AND authority_id = ?", AdminAuthorityID).
		Where("EXISTS (SELECT 1 FROM sys_user_authority ua WHERE ua.sys_user_id = sys_users.id AND ua.sys_authority_authority_id = ?)", AdminAuthorityID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "至少保留一个启用的管理员账号，不能移除最后管理员的角色或停用该账号")
	}
	return nil
}

var adminRecoveryAPIs = []struct{ Path, Method string }{
	{"/v1/sys/menu/getMenu", "GET"},
	{"/v1/sys/authority/getAuthorityList", "GET"},
	{"/v1/sys/api/getApiList", "GET"},
	{"/v1/sys/api/getAllApiList", "GET"},
	{"/v1/sys/casbin/getPathByAuthorityId", "GET"},
	{"/v1/sys/casbin/updateCasbinDataByApiIds", "PUT"},
}

// Snapshot individual recovery permissions so a legacy database can be repaired
// incrementally without unrelated writes being blocked by already missing grants.
func adminRecoveryPolicyState(tx *gorm.DB) (map[string]bool, error) {
	state := map[string]bool{}
	exists, err := adminRoleExists(tx)
	if err != nil || !exists {
		return state, err
	}
	var rules []gormadapter.CasbinRule
	if err := tx.Where("ptype = ? AND v0 = ?", "p", strconv.FormatInt(AdminAuthorityID, 10)).Find(&rules).Error; err != nil {
		return nil, err
	}
	for _, required := range adminRecoveryAPIs {
		key := required.Method + " " + required.Path
		state[key] = false
		for _, rule := range rules {
			if rule.V2 != required.Method {
				continue
			}
			// Invalid legacy patterns are not usable permissions. Permit their repair.
			matches, err := regexp.MatchString(policyPathPattern(rule.V1), required.Path)
			if err == nil && matches {
				state[key] = true
				break
			}
		}
	}
	return state, nil
}

func preserveAdminRecoveryPolicies(tx *gorm.DB, before map[string]bool) error {
	after, err := adminRecoveryPolicyState(tx)
	if err != nil {
		return err
	}
	for key, allowed := range before {
		if allowed && !after[key] {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "管理员必须保留权限恢复入口："+key)
		}
	}
	return nil
}

func EnsureAdminRecoveryPolicies(tx *gorm.DB) error {
	state, err := adminRecoveryPolicyState(tx)
	if err != nil {
		return err
	}
	for key, allowed := range state {
		if !allowed {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "管理员必须保留权限恢复入口："+key)
		}
	}
	return nil
}
