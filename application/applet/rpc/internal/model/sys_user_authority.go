package model

// SysUserAuthority 是 sysUser 和 sysAuthority 的连接表
type SysUserAuthority struct {
	SysUserId               int64 `gorm:"column:sys_user_id"`
	SysAuthorityAuthorityId int64 `gorm:"column:sys_authority_authority_id"`
}

func (s *SysUserAuthority) TableName() string {
	return "sys_user_authority"
}
