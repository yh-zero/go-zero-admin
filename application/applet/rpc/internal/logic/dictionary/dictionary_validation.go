package dictionarylogic

import (
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"strings"
)

func validateDictionary(db *gorm.DB, value *pb.SysDictionary) error {
	if value == nil || strings.TrimSpace(value.Name) == "" || strings.TrimSpace(value.Type) == "" {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "字典名称和类型不能为空")
	}
	if err := accessutil.Status(value.Status); err != nil {
		return err
	}
	return accessutil.Unique(db, &model.SysDictionary{}, "id <> ? AND type = ?", value.ID, value.Type)
}

func validateItem(db *gorm.DB, value *pb.SysDictionaryInfo) error {
	if value == nil || strings.TrimSpace(value.Label) == "" {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "字典项名称不能为空")
	}
	if err := accessutil.Status(value.Status); err != nil {
		return err
	}
	var dictionary model.SysDictionary
	if err := accessutil.RequireID(db, &dictionary, value.SysDictionaryID); err != nil {
		return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "所属字典不存在")
	}
	return accessutil.Unique(db, &model.SysDictionaryInfo{}, "id <> ? AND sys_dictionary_id = ? AND value = ?", value.ID, value.SysDictionaryID, value.Value)
}
