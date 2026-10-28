package model

type SysAuthorityBtn struct {
	AuthorityId      int64          `gorm:"comment:角色ID"`
	SysMenuID        int64          `gorm:"comment:菜单ID"`
	SysBaseMenuBtnID int64          `gorm:"comment:菜单按钮ID"`
	SysBaseMenuBtn   SysBaseMenuBtn ` gorm:"comment:按钮详情"`
}
