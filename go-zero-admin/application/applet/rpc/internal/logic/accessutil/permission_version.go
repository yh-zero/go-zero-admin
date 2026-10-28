package accessutil

import (
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
)

func LockPermissionVersion(tx *gorm.DB) error {
	var row model.SysPermissionVersion
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", 1).Error
}
func PermissionRevision(tx *gorm.DB) (string, error) {
	var row model.SysPermissionVersion
	if err := tx.First(&row, "id = ?", 1).Error; err != nil {
		return "", err
	}
	return strconv.FormatUint(row.Revision, 10), nil
}
func RequirePermissionRevision(tx *gorm.DB, expected string) error {
	n, err := strconv.ParseUint(expected, 10, 64)
	if err != nil || n == 0 || strconv.FormatUint(n, 10) != expected {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "缺少或无效的权限版本，请重新加载后确认")
	}
	current, err := PermissionRevision(tx)
	if err != nil {
		return err
	}
	if expected != current {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "权限已变更，请重新加载后确认")
	}
	return nil
}

// Holding the same locks as writers makes every resource and selection query
// in fn belong to one version, including under MySQL READ COMMITTED.
func PermissionRead(db *gorm.DB, fn func(*gorm.DB) error) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := LockAdminGuard(tx); err != nil {
			return err
		}
		if err := LockPermissionVersion(tx); err != nil {
			return err
		}
		return fn(tx)
	})
}
func PermissionTransaction(db *gorm.DB, fn func(*gorm.DB) error) error {
	return PermissionRead(db, func(tx *gorm.DB) error {
		before, err := CapturePermissionState(tx)
		if err != nil {
			return err
		}
		if err := fn(tx); err != nil {
			return err
		}
		return CommitPermissionChanges(db.Statement.Context, tx, before)
	})
}
