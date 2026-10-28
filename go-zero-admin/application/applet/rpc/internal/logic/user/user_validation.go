package userlogic

import (
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

func userError(message string) error { return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, message) }
func validateUserAuthorities(tx *gorm.DB, ids []int64, defaultID int64) ([]int64, error) {
	unique := make([]int64, 0, len(ids))
	seen := map[int64]bool{}
	for _, id := range ids {
		if id <= 0 {
			return nil, userError("角色ID无效")
		}
		if !seen[id] {
			unique = append(unique, id)
			seen[id] = true
		}
	}
	if len(unique) == 0 {
		return nil, userError("用户至少需要一个角色")
	}
	if !seen[defaultID] {
		return nil, userError("默认角色必须属于已选角色")
	}
	var count int64
	if err := tx.Model(&model.SysAuthority{}).Where("authority_id IN ? AND deleted_at IS NULL", unique).Count(&count).Error; err != nil {
		return nil, err
	}
	if count != int64(len(unique)) {
		return nil, userError("所选角色不存在")
	}
	return unique, nil
}
func replaceUserAuthorities(tx *gorm.DB, userID int64, ids []int64) error {
	if err := tx.Where("sys_user_id = ?", userID).Delete(&model.SysUserAuthority{}).Error; err != nil {
		return err
	}
	rows := make([]model.SysUserAuthority, 0, len(ids))
	for _, id := range ids {
		rows = append(rows, model.SysUserAuthority{SysUserId: userID, SysAuthorityAuthorityId: id})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.Create(&rows).Error
}
