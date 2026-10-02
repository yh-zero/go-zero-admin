package model

import (
	"time"

	base "go-zero-admin/pkg/model"
)

type SysDepartment struct {
	base.MODEL_BASE
	ParentID int64  `gorm:"column:parent_id;not null;index"`
	Name     string `gorm:"size:64;not null"`
	Code     string `gorm:"size:64;not null;uniqueIndex"`
	Sort     int64
	Status   int64  `gorm:"not null"`
	Leader   string `gorm:"size:64"`
}

func (SysDepartment) TableName() string { return "sys_departments" }

type SysPosition struct {
	base.MODEL_BASE
	Name   string `gorm:"size:64;not null"`
	Code   string `gorm:"size:64;not null;uniqueIndex"`
	Sort   int64
	Status int64 `gorm:"not null"`
}

func (SysPosition) TableName() string { return "sys_positions" }

type SysUserDepartment struct {
	UserID       int64 `gorm:"column:user_id;primaryKey;autoIncrement:false"`
	DepartmentID int64 `gorm:"column:department_id;not null;index"`
	UpdatedAt    time.Time
}

func (SysUserDepartment) TableName() string { return "sys_user_departments" }

type SysUserPosition struct {
	UserID     int64 `gorm:"column:user_id;primaryKey;autoIncrement:false"`
	PositionID int64 `gorm:"column:position_id;primaryKey;autoIncrement:false;index"`
}

func (SysUserPosition) TableName() string { return "sys_user_positions" }

type SysRoleDataScope struct {
	AuthorityID int64  `gorm:"column:authority_id;primaryKey;autoIncrement:false"`
	Scope       string `gorm:"size:32;not null"`
	UpdatedAt   time.Time
}

func (SysRoleDataScope) TableName() string { return "sys_role_data_scopes" }

type SysRoleScopeDepartment struct {
	AuthorityID  int64 `gorm:"column:authority_id;primaryKey;autoIncrement:false"`
	DepartmentID int64 `gorm:"column:department_id;primaryKey;autoIncrement:false;index"`
}

func (SysRoleScopeDepartment) TableName() string { return "sys_role_scope_departments" }
