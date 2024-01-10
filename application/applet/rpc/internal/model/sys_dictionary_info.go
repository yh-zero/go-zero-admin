// 自动生成模板SysDictionaryDetail
package model

import "go-zero-admin/pkg/model"

type SysDictionaryInfo struct {
	model.MODEL_BASE
	Label           string `json:"label" form:"label" gorm:"column:label;comment:展示值"`                                  // 展示值
	Value           int64  `json:"value" form:"value" gorm:"column:value;comment:字典值"`                                  // 字典值
	Extend          string `json:"extend" form:"extend" gorm:"column:extend;comment:扩展值"`                               // 扩展值
	Status          *bool  `json:"status" form:"status" gorm:"column:status;comment:启用状态"`                              // 启用状态
	Sort            int64  `json:"sort" form:"sort" gorm:"column:sort;comment:排序标记"`                                    // 排序标记
	SysDictionaryID int64  `json:"sysDictionaryID" form:"sysDictionaryID" gorm:"column:sys_dictionary_id;comment:关联标记"` // 关联标记
}

func (SysDictionaryInfo) TableName() string {
	return "sys_dictionary_info"
}
