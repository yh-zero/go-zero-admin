package model

import "time"

// AppMembershipPlan is the versioned source copied into every independent grant.
type AppMembershipPlan struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	Code         string    `gorm:"size:32;not null;uniqueIndex:uk_app_plan_version,priority:1"`
	Name         string    `gorm:"size:64;not null"`
	Version      int64     `gorm:"not null;uniqueIndex:uk_app_plan_version,priority:2"`
	Entitlements string    `gorm:"type:json;not null"`
	Enabled      bool      `gorm:"not null"`
	CreatedAt    time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt    time.Time `gorm:"not null;autoUpdateTime"`
}

func (AppMembershipPlan) TableName() string { return "app_membership_plans" }

// AppMembershipGrant retains its entitlement snapshot even if its plan changes.
// Revocation only affects this grant; it never shifts later renewal periods.
type AppMembershipGrant struct {
	ID           int64      `gorm:"primaryKey;autoIncrement"`
	UserID       int64      `gorm:"not null;index:idx_app_grant_user_end,priority:1"`
	PlanID       int64      `gorm:"not null"`
	PlanCode     string     `gorm:"size:32;not null"`
	PlanVersion  int64      `gorm:"not null"`
	Entitlements string     `gorm:"type:json;not null"`
	RequestID    string     `gorm:"size:64;not null;uniqueIndex:uk_app_grant_request"`
	Days         int64      `gorm:"not null"`
	StartsAt     time.Time  `gorm:"not null"`
	EndsAt       time.Time  `gorm:"not null;index:idx_app_grant_user_end,priority:2"`
	RevokedAt    *time.Time `gorm:"index:idx_app_grant_revoked"`
	GrantedBy    int64      `gorm:"not null"`
	RevokedBy    int64      `gorm:"not null;default:0"`
	Reason       string     `gorm:"size:500;not null"`
	RevokeReason string     `gorm:"size:500;not null;default:''"`
	CreatedAt    time.Time  `gorm:"not null;autoCreateTime"`
}

func (AppMembershipGrant) TableName() string { return "app_membership_grants" }
