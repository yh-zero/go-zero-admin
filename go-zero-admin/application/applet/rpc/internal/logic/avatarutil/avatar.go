// Package avatarutil maintains avatar references inside the caller's user transaction.
package avatarutil

import (
	"context"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/gofrs/uuid/v5"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/objectstorage"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const SysUser = "sys_user_avatar"
const AppUser = "app_user_avatar"

func IsSystemReference(kind string) bool { return kind == SysUser || kind == AppUser }

func invalid(message string) error { return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, message) }

// The caller holds the user row lock. File locks follow in ID order, before any
// reference writes, matching deletion's file-then-reference lock order.
func Sync(ctx context.Context, tx *gorm.DB, storage objectstorage.Config, kind string, ownerID int64, previousURL, nextURL string) error {
	if !IsSystemReference(kind) || ownerID <= 0 {
		return invalid("头像引用对象无效")
	}
	nextKey, err := avatarKey(storage, nextURL)
	if err != nil {
		return err
	}
	previousKey, _ := avatarKey(storage, previousURL)
	keys := []string{}
	if previousKey != "" {
		keys = append(keys, previousKey)
	}
	if nextKey != "" && nextKey != previousKey {
		keys = append(keys, nextKey)
	}
	var files []model.SysFileResource
	if len(keys) > 0 {
		if err := tx.WithContext(ctx).Where("object_key IN ?", keys).Order("id").Clauses(clause.Locking{Strength: "UPDATE"}).Find(&files).Error; err != nil {
			return err
		}
	}
	var nextID int64
	for _, file := range files {
		if file.ObjectKey == nextKey {
			if file.Visibility != "public" || file.Status != "active" {
				return invalid("头像必须使用公开且可用的文件")
			}
			nextID = file.ID
		}
	}
	if nextKey != "" && nextID == 0 {
		return invalid("头像文件不存在或尚未登记")
	}
	objectID := strconv.FormatInt(ownerID, 10)
	if nextID > 0 {
		reference := model.SysFileReference{FileID: nextID, ObjectType: kind, ObjectID: objectID}
		if err := tx.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&reference).Error; err != nil {
			return err
		}
	}
	return tx.WithContext(ctx).Where("object_type = ? AND object_id = ? AND file_id <> ?", kind, objectID, nextID).Delete(&model.SysFileReference{}).Error
}

// Recognize the object path across host aliases too: localhost and 127.0.0.1
// must not let a managed image escape deletion protection. No URL is fetched.
func avatarKey(storage objectstorage.Config, value string) (string, error) {
	u, err := url.Parse(strings.ReplaceAll(strings.TrimSpace(value), "\\", "/"))
	if err != nil {
		return "", invalid("头像地址无效")
	}
	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme != "" && u.Scheme != "http" && u.Scheme != "https" {
		return "", nil
	}
	if u.Opaque != "" {
		return "", invalid("头像地址无效")
	}
	if storage.PrivateBucket != "" && strings.HasPrefix(u.Path, "/"+storage.PrivateBucket+"/") {
		return "", invalid("私有临时地址不能作为头像")
	}
	index := strings.LastIndex(u.Path, "/go-zero-admin/")
	if index < 0 {
		origin, _ := url.Parse(storage.ExternalEndpoint)
		if (origin != nil && origin.Host != "" && strings.EqualFold(u.Host, origin.Host)) || (storage.PublicBucket != "" && strings.HasPrefix(u.Path, "/"+storage.PublicBucket+"/")) {
			return "", invalid("头像文件地址无效")
		}
		return "", nil
	}
	key := u.Path[index+1:]
	extension := path.Ext(key)
	if extension != ".png" && extension != ".jpg" && extension != ".gif" && extension != ".webp" {
		return "", invalid("头像文件类型无效")
	}
	if _, err := uuid.FromString(strings.TrimSuffix(strings.TrimPrefix(key, "go-zero-admin/"), extension)); err != nil {
		return "", invalid("头像文件标识无效")
	}
	if u.User != nil || (storage.PublicBucket != "" && u.Path != "/"+storage.PublicBucket+"/"+key) {
		return "", invalid("头像必须使用公开文件地址")
	}
	return key, nil
}
