package model

import "time"

type SysFileResource struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	ObjectKey    string `gorm:"size:256;not null;uniqueIndex:uk_file_object_key"`
	Name         string `gorm:"size:255;not null"`
	Mime         string `gorm:"size:128;not null"`
	Size         int64  `gorm:"not null"`
	OwnerID      int64  `gorm:"not null;index:idx_file_owner_status,priority:1"`
	DepartmentID int64  `gorm:"not null;default:0;index:idx_file_department_status,priority:1"`
	Visibility   string `gorm:"size:16;not null;default:public"`
	Status       string `gorm:"size:16;not null;default:active;index:idx_file_owner_status,priority:2;index:idx_file_department_status,priority:2"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (SysFileResource) TableName() string { return "sys_file_resources" }

type SysFileReference struct {
	ID         int64  `gorm:"primaryKey;autoIncrement"`
	FileID     int64  `gorm:"not null;uniqueIndex:uk_file_reference,priority:1"`
	ObjectType string `gorm:"size:64;not null;uniqueIndex:uk_file_reference,priority:2"`
	ObjectID   string `gorm:"size:128;not null;uniqueIndex:uk_file_reference,priority:3"`
	CreatedAt  time.Time
}

func (SysFileReference) TableName() string { return "sys_file_references" }
