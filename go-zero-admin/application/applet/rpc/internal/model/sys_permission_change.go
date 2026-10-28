package model

import "time"

// Only controlled grant documents enter this table. Request bodies, JWTs and
// password hashes are deliberately absent; migration owns table creation.
type SysPermissionChange struct {
	ID             int64     `gorm:"primaryKey"`
	AuthorityID    int64     `gorm:"not null;index:idx_permission_history_role,priority:1"`
	Kind           string    `gorm:"size:16;not null"`
	BeforeRevision uint64    `gorm:"not null"`
	AfterRevision  uint64    `gorm:"not null"`
	Before         string    `gorm:"type:longtext;not null"`
	After          string    `gorm:"type:longtext;not null"`
	ActorID        int64     `gorm:"not null"`
	ActorName      string    `gorm:"size:64;not null"`
	TraceID        string    `gorm:"size:64;not null"`
	CreatedAt      time.Time `gorm:"not null;index:idx_permission_history_role,priority:2"`
}

func (SysPermissionChange) TableName() string { return "sys_permission_changes" }
