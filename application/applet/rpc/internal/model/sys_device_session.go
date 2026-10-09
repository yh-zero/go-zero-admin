package model

import "time"

// SysDeviceSession retains revoked rows for administration. JWTs are never stored.
type SysDeviceSession struct {
	ID             string     `gorm:"primaryKey;size:36"`
	UserID         int64      `gorm:"not null;index:idx_device_user,priority:1"`
	AuthorityID    int64      `gorm:"not null"`
	SessionVersion int64      `gorm:"not null"`
	IP             string     `gorm:"size:64;not null"`
	UserAgent      string     `gorm:"size:512;not null"`
	CreatedAt      time.Time  `gorm:"not null;autoCreateTime"`
	ExpiresAt      time.Time  `gorm:"not null;index:idx_device_expiry;index:idx_device_user,priority:2"`
	RevokedAt      *time.Time `gorm:"index:idx_device_revoked"`
}

func (SysDeviceSession) TableName() string { return "sys_device_sessions" }
