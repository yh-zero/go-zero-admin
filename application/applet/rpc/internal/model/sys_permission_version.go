package model

// Provisioned by the permission migration, never by service startup.
type SysPermissionVersion struct {
	ID       int64  `gorm:"primaryKey;autoIncrement:false"`
	Revision uint64 `gorm:"not null"`
}

func (SysPermissionVersion) TableName() string { return "sys_permission_versions" }
