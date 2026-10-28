package casbinlogic

import (
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"strconv"
)

// Existing unregistered rules may be preserved by advanced mode, but new rules
// must name a registered API. Validate everything before replacing any grants.
func replaceRolePolicies(tx *gorm.DB, authorityID int64, entries []*pb.CasbinInfo) error {
	if err := accessutil.RequireRole(tx, authorityID); err != nil {
		return err
	}
	role := strconv.FormatInt(authorityID, 10)
	var old []gormadapter.CasbinRule
	if err := tx.Where("ptype = ? AND v0 = ?", "p", role).Find(&old).Error; err != nil {
		return err
	}
	allowed := map[string]bool{}
	for _, rule := range old {
		allowed[rule.V1+"\x00"+rule.V2] = true
	}
	var apis []model.SysApi
	if err := tx.Find(&apis).Error; err != nil {
		return err
	}
	for _, api := range apis {
		allowed[api.Path+"\x00"+api.Method] = true
	}
	rows := make([]gormadapter.CasbinRule, 0, len(entries))
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry == nil {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "接口规则不能为空")
		}
		path, method, err := accessutil.API(entry.Path, entry.Method)
		if err != nil {
			return err
		}
		key := path + "\x00" + method
		if !allowed[key] {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "接口尚未登记，请先在API管理中创建")
		}
		if !seen[key] {
			seen[key] = true
			rows = append(rows, gormadapter.CasbinRule{Ptype: "p", V0: role, V1: path, V2: method})
		}
	}
	if err := tx.Where("ptype = ? AND v0 = ?", "p", role).Delete(&gormadapter.CasbinRule{}).Error; err != nil {
		return err
	}
	if len(rows) > 0 {
		return tx.Create(&rows).Error
	}
	return nil
}
