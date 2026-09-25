package accessutil

import (
	"errors"
	"go-zero-admin/pkg/result/xerr"
	"strings"

	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"
)

func RequireRole(db *gorm.DB, id int64) error {
	if id <= 0 {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "角色ID无效")
	}
	var role model.SysAuthority
	if err := db.Where("authority_id = ? AND deleted_at IS NULL", id).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "角色不存在")
		}
		return err
	}
	return nil
}

func RequireID(db *gorm.DB, value interface{}, id int64) error {
	if id <= 0 {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "ID无效")
	}
	if err := db.First(value, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "记录不存在")
		}
		return err
	}
	return nil
}

func UniqueIDs(ids []int64) ([]int64, error) {
	result := make([]int64, 0, len(ids))
	seen := map[int64]bool{}
	for _, id := range ids {
		if id <= 0 {
			return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "ID必须大于0")
		}
		if !seen[id] {
			result = append(result, id)
			seen[id] = true
		}
	}
	return result, nil
}

// ValidateParent walks ancestors, rejecting both self-parenting and existing cycles.
func ValidateParent(id, parent int64, parents map[int64]int64) error {
	if parent < 0 {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "父节点ID无效")
	}
	seen := map[int64]bool{}
	for parent != 0 {
		if parent == id || seen[parent] {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "父节点不能是自身或子孙节点")
		}
		next, ok := parents[parent]
		if !ok {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "父节点不存在")
		}
		seen[parent] = true
		parent = next
	}
	return nil
}

func Status(value int64) error {
	if value != 1 && value != 2 {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "状态只允许1（启用）或2（禁用）")
	}
	return nil
}

func Page(page, size int64) (int, int, error) {
	if page < 1 || size < 1 || size > 500 {
		return 0, 0, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "分页参数无效，pageSize须在1至500之间")
	}
	return int((page - 1) * size), int(size), nil
}

func API(path, method string) (string, string, error) {
	path = strings.TrimSpace(path)
	method = strings.ToUpper(strings.TrimSpace(method))
	if !strings.HasPrefix(path, "/v1/") || strings.ContainsAny(path, "?# \\") {
		return "", "", xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "接口路径必须以/v1/开头，且不能包含查询参数")
	}
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS":
	default:
		return "", "", xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "HTTP方法无效")
	}
	return path, method, nil
}

func Unique(db *gorm.DB, entity interface{}, condition string, args ...interface{}) error {
	var count int64
	if err := db.Model(entity).Where(condition, args...).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "存在重复记录，请检查名称、路径或值")
	}
	return nil
}
