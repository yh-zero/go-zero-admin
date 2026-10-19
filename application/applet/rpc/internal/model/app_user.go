package model

import "time"

type AppUser struct {
	ID                 int64  `gorm:"primaryKey;autoIncrement"`
	Nickname           string `gorm:"size:64;not null"`
	RegistrationIP     string `gorm:"size:64;not null;default:''"`
	RegistrationRegion string `gorm:"size:255;not null;default:''"`
	Status             string `gorm:"size:16;not null;default:pending;index:idx_app_user_status"`
	SessionVersion     int64  `gorm:"not null;default:1"`
	ReviewReason       string `gorm:"size:500;not null;default:''"`
	ReviewedBy         int64  `gorm:"not null;default:0"`
	ReviewedAt         *time.Time
	CreatedAt          time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt          time.Time `gorm:"not null;autoUpdateTime"`
}

func (AppUser) TableName() string { return "app_users" }

type AppUserIdentity struct {
	VerifiedAt   *time.Time
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	UserID       int64     `gorm:"not null;uniqueIndex:uk_app_identity_user"`
	Kind         string    `gorm:"size:16;not null"`
	Identifier   string    `gorm:"size:254;not null;uniqueIndex:uk_app_identity_identifier"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `gorm:"not null;autoCreateTime"`
}

func (AppUserIdentity) TableName() string { return "app_user_identities" }
