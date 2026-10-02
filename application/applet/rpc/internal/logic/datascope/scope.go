// Package datascope applies the active role's data range to an owned-resource query.
package datascope

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

var identifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)?$`)

// Apply never treats a database or configuration error as permission to read.
// Column names are internal schema identifiers, validated before constructing SQL.
func Apply(ctx context.Context, db *gorm.DB, actor *pb.SessionRequest, ownerColumn, departmentColumn string) (*gorm.DB, error) {
	if db == nil {
		return nil, errors.New("data scope database is required")
	}
	if db.Error != nil {
		return nil, db.Error
	}
	if !identifier.MatchString(ownerColumn) || !identifier.MatchString(departmentColumn) {
		return nil, errors.New("invalid data scope column")
	}
	if actor == nil || actor.UserID <= 0 || actor.SessionVersion <= 0 || actor.AuthorityId <= 0 {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	source := db.Session(&gorm.Session{NewDB: true, Context: ctx})
	var user model.SysUser
	userQuery := source.Model(&model.SysUser{}).
		Where("sys_users.id = ? AND sys_users.enable = 1 AND sys_users.session_version = ? AND sys_users.authority_id = ?", actor.UserID, actor.SessionVersion, actor.AuthorityId).
		Where("EXISTS (SELECT 1 FROM sys_user_authority ua JOIN sys_authorities a ON a.authority_id = ua.sys_authority_authority_id AND a.deleted_at IS NULL WHERE ua.sys_user_id = sys_users.id AND ua.sys_authority_authority_id = sys_users.authority_id)")
	if actor.SessionID != "" {
		userQuery = userQuery.Where("EXISTS (SELECT 1 FROM sys_device_sessions ds WHERE ds.id = ? AND ds.user_id = sys_users.id AND ds.authority_id = sys_users.authority_id AND ds.session_version = sys_users.session_version AND ds.revoked_at IS NULL AND ds.expires_at > ?)", actor.SessionID, time.Now().UTC())
	}
	err := userQuery.First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	if err != nil {
		return nil, err
	}
	query := db.WithContext(ctx)
	if actor.AuthorityId == accessutil.AdminAuthorityID {
		return query, nil
	}
	scope := "self"
	var rule model.SysRoleDataScope
	err = source.First(&rule, "authority_id = ?", actor.AuthorityId).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		scope = rule.Scope
	}
	switch scope {
	case "all":
		return query, nil
	case "self":
		return query.Where(ownerColumn+" = ?", actor.UserID), nil
	case "department", "department_and_children":
		var membership model.SysUserDepartment
		err := source.First(&membership, "user_id = ?", actor.UserID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return query.Where("1 = 0"), nil
		}
		if err != nil {
			return nil, err
		}
		var department model.SysDepartment
		if err := source.First(&department, membership.DepartmentID).Error; err != nil {
			return nil, err
		}
		ids := []int64{membership.DepartmentID}
		if scope == "department_and_children" {
			var rows []model.SysDepartment
			if err := source.Find(&rows).Error; err != nil {
				return nil, err
			}
			children := map[int64][]int64{}
			for _, row := range rows {
				children[row.ParentID] = append(children[row.ParentID], row.ID)
			}
			seen := map[int64]bool{membership.DepartmentID: true}
			for i := 0; i < len(ids); i++ {
				for _, id := range children[ids[i]] {
					if seen[id] {
						return nil, errors.New("department hierarchy contains a cycle")
					}
					seen[id] = true
					ids = append(ids, id)
				}
			}
		}
		return query.Where(departmentColumn+" IN ?", ids), nil
	case "custom":
		var ids []int64
		if err := source.Model(&model.SysRoleScopeDepartment{}).Where("authority_id = ?", actor.AuthorityId).Pluck("department_id", &ids).Error; err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			return query.Where("1 = 0"), nil
		}
		var count int64
		if err := source.Model(&model.SysDepartment{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
			return nil, err
		}
		if count != int64(len(ids)) {
			return nil, errors.New("custom data scope references missing departments")
		}
		return query.Where(departmentColumn+" IN ?", ids), nil
	default:
		return nil, errors.New("unsupported role data scope: " + strings.TrimSpace(scope))
	}
}
