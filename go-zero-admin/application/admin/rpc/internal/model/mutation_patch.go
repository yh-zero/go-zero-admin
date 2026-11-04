package model

import (
	"errors"
	"fmt"
	"strings"
)

// CommittedMutationError 表示数据库已提交，失败仅发生在提交后的缓存失效。
// 调用方仍须刷新安全策略，不能把缓存错误误判为事务回滚。
type CommittedMutationError struct{ Err error }

func (e *CommittedMutationError) Error() string { return e.Err.Error() }
func (e *CommittedMutationError) Unwrap() error { return e.Err }
func MutationCommitted(err error) bool {
	var committed *CommittedMutationError
	return err == nil || errors.As(err, &committed)
}
func committedCacheError(err error) error {
	if err != nil {
		return &CommittedMutationError{Err: err}
	}
	return nil
}

func CreateStatus(status *int32) (int64, error) {
	if status == nil {
		return 1, nil
	}
	if *status != 1 && *status != 2 {
		return 0, fmt.Errorf("%w: 状态只能为1或2", ErrInvalidMutation)
	}
	return int64(*status), nil
}

type RolePatch struct {
	Name, Code, Description, DataScope *string
	Sort, Status                       *int32
	MenuIds                            []int64
	UpdateMenus                        bool
}

func applyRolePatch(role *SysRole, p RolePatch) error {
	if p.Code != nil && *p.Code != role.Code {
		return fmt.Errorf("%w: 角色标识创建后不能修改", ErrInvalidMutation)
	}
	if p.Name != nil {
		if strings.TrimSpace(*p.Name) == "" {
			return ErrInvalidMutation
		}
		role.Name = *p.Name
	}
	if p.Description != nil {
		role.Description = *p.Description
	}
	if p.DataScope != nil {
		switch *p.DataScope {
		case "ALL", "DEPT_AND_CHILDREN", "DEPT", "SELF":
		default:
			return ErrInvalidMutation
		}
		role.DataScope = *p.DataScope
	}
	if p.Sort != nil {
		role.Sort = int64(*p.Sort)
	}
	if p.Status != nil {
		status, err := CreateStatus(p.Status)
		if err != nil {
			return err
		}
		role.Status = status
	}
	if role.Code == "super_admin" && role.Status != 1 {
		return fmt.Errorf("%w: 超级管理员角色不能停用", ErrInvalidMutation)
	}
	return nil
}

type MenuPatch struct {
	ParentId                                     *int64
	Title, Name, Path, Component, PermCode, Icon *string
	Type, Sort, Status                           *int32
	Visible                                      *bool
}

func applyMenuPatch(menu *SysMenu, p MenuPatch) error {
	if p.ParentId != nil {
		if *p.ParentId < 0 {
			return ErrInvalidMutation
		}
		menu.ParentId = uint64(*p.ParentId)
	}
	if p.Title != nil {
		if strings.TrimSpace(*p.Title) == "" {
			return ErrInvalidMutation
		}
		menu.Title = *p.Title
	}
	if p.Name != nil {
		menu.Name = *p.Name
	}
	if p.Path != nil {
		menu.Path = *p.Path
	}
	if p.Component != nil {
		menu.Component = *p.Component
	}
	if p.PermCode != nil {
		menu.PermCode = *p.PermCode
	}
	if p.Icon != nil {
		menu.Icon = *p.Icon
	}
	if p.Sort != nil {
		menu.Sort = int64(*p.Sort)
	}
	if p.Visible != nil {
		menu.Visible = 0
		if *p.Visible {
			menu.Visible = 1
		}
	}
	if p.Type != nil {
		if *p.Type < 1 || *p.Type > 3 {
			return ErrInvalidMutation
		}
		menu.Type = int64(*p.Type)
	}
	if p.Status != nil {
		status, err := CreateStatus(p.Status)
		if err != nil {
			return err
		}
		menu.Status = status
	}
	return nil
}

type DictPatch struct {
	Name, Code, Items, Description *string
	Status                         *int32
}

func applyDictPatch(dict *SysDict, p DictPatch) error {
	if p.Name != nil {
		if strings.TrimSpace(*p.Name) == "" {
			return ErrInvalidMutation
		}
		dict.Name = *p.Name
	}
	if p.Code != nil {
		if strings.TrimSpace(*p.Code) == "" {
			return ErrInvalidMutation
		}
		dict.Code = *p.Code
	}
	if p.Items != nil {
		dict.Items = *p.Items
	}
	if p.Description != nil {
		dict.Description = *p.Description
	}
	if p.Status != nil {
		status, err := CreateStatus(p.Status)
		if err != nil {
			return err
		}
		dict.Status = status
	}
	return nil
}
