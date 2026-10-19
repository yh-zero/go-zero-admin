package model

import "time"

type AppSession struct {
	ID                  string    `gorm:"primaryKey;size:36"`
	UserID              int64     `gorm:"not null;index:idx_app_session_user"`
	RefreshHash         string    `gorm:"size:64;not null" json:"-"`
	PreviousRefreshHash string    `gorm:"size:64;not null;default:''" json:"-"`
	RefreshGeneration   int64     `gorm:"not null;default:1"`
	SessionVersion      int64     `gorm:"not null"`
	IP                  string    `gorm:"size:64;not null"`
	UserAgent           string    `gorm:"size:512;not null"`
	ExpiresAt           time.Time `gorm:"not null;index:idx_app_session_expiry"`
	RevokedAt           *time.Time
	CreatedAt           time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt           time.Time `gorm:"not null;autoUpdateTime"`
}

func (AppSession) TableName() string { return "app_sessions" }
