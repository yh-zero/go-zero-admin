package model

import (
	"gorm.io/gorm"
	"strconv"
)

func (b *SysBaseMenuBtn) BeforeCreate(_ *gorm.DB) error {
	if b.PermissionKey == "" {
		b.PermissionKey = "menu." + strconv.FormatInt(b.SysBaseMenuID, 10) + ".button." + b.Name
	}
	return nil
}
