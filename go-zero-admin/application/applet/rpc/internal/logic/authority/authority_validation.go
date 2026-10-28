package authoritylogic

import (
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"strings"
)

func validateAuthority(db *gorm.DB, value *pb.SysAuthority) error {
	if value == nil || value.AuthorityId <= 0 || strings.TrimSpace(value.AuthorityName) == "" {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "角色ID和名称不能为空")
	}
	var all []model.SysAuthority
	if err := db.Where("deleted_at IS NULL").Find(&all).Error; err != nil {
		return err
	}
	parents := map[int64]int64{}
	for _, role := range all {
		if role.ParentId != nil {
			parents[role.AuthorityId] = *role.ParentId
		} else {
			parents[role.AuthorityId] = 0
		}
	}
	return accessutil.ValidateParent(value.AuthorityId, value.ParentId, parents)
}
