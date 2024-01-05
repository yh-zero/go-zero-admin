package model

import (
	"gorm.io/gorm"
	"time"
)

type MODEL_BASE struct {
	ID        int64          `gorm:"primarykey"` // 主键ID
	CreatedAt time.Time      // 创建时间
	UpdatedAt time.Time      // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 删除时间
}
