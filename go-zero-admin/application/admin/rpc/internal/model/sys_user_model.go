package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysUserModel = (*customSysUserModel)(nil)

type (
	// SysUserModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysUserModel.
	SysUserModel interface {
		sysUserModel
		// ListUserPage 分页查有效用户
		ListUserPage(ctx context.Context, f UserListFilter, page, pageSize int64) ([]*UserRow, int64, error)
		FindScopedUser(ctx context.Context, userId int64, actor UserActor) (*SysUser, error)
		UpdateLastLogin(ctx context.Context, userId uint64, ip string) error
		WithLoginUser(ctx context.Context, userId uint64, expectedHash string, issue func(context.Context, *SysUser) error) error
		UpdatePassword(ctx context.Context, userId uint64, actor UserActor, expectedHash, newHash string, revoke func(context.Context, int64) error) error
		CreateWithRoles(ctx context.Context, user *SysUser, actor UserActor, roleIds []int64) error
		UpdatePartial(ctx context.Context, userId int64, actor UserActor, patch UserPatch) error
		ListUserRoleIds(ctx context.Context, userIds []int64) (map[int64][]int64, error)
		IsSuperAdmin(ctx context.Context, userId int64) (bool, error)
		DeleteWithRoles(ctx context.Context, userId int64, actor UserActor) error
		ReconcileAvatar(ctx context.Context, userId int64, reconcile func(context.Context, int64) error) error
	}

	customSysUserModel struct {
		*defaultSysUserModel
	}
)

var (
	ErrInvalidMutation         = errors.New("参数不合法")
	ErrRoleAssignmentForbidden = errors.New("仅超级管理员可以分配角色")
	ErrLastSuperAdmin          = errors.New("不能移除或停用最后一个超级管理员")
	ErrUserDisabled            = errors.New("用户已停用")
	ErrPasswordChanged         = errors.New("密码已变化")
	ErrDataScopeDenied         = errors.New("没有目标用户的数据权限")
	ErrUsernameExists          = errors.New("用户名已存在")
)

// OwnAccount 仅由个人资料/自助改密服务设置，不从 HTTP 或客户端 Actor 接收。
type UserActor struct {
	UserId, DeptId int64
	OwnAccount     bool
}

type userDataScope struct {
	actor                              UserActor
	all, self, department, descendants bool
	departmentPath                     string
}

type scopeQuery func(context.Context, any, string, ...any) error

func resolveUserScope(ctx context.Context, query scopeQuery, actor UserActor) (userDataScope, error) {
	scope := userDataScope{actor: actor}
	if actor.UserId <= 0 || actor.DeptId < 0 {
		return scope, ErrDataScopeDenied
	}
	if actor.OwnAccount {
		var rows []struct {
			Id int64 `db:"id"`
		}
		if err := query(ctx, &rows, "SELECT id FROM sys_user WHERE id=? AND status=1 AND deleted_at IS NULL", actor.UserId); err != nil {
			return scope, err
		}
		if len(rows) != 1 {
			return scope, ErrDataScopeDenied
		}
		scope.self = true
		return scope, nil
	}
	var rows []struct {
		Code           string `db:"code"`
		DataScope      string `db:"data_scope"`
		Domain         string `db:"domain"`
		DepartmentPath string `db:"department_path"`
	}
	err := query(ctx, &rows, `SELECT r.code,r.data_scope,g.v2 AS domain,COALESCE(d.path,'') AS department_path
FROM casbin_rule g JOIN sys_role r ON r.code=g.v1 AND r.status=1 AND r.deleted_at IS NULL
JOIN sys_user a ON a.id=? AND a.status=1 AND a.deleted_at IS NULL
LEFT JOIN sys_dept d ON d.id=? AND d.status=1 AND d.deleted_at IS NULL
WHERE g.ptype='g' AND g.v0=CONCAT('user:',a.id) AND (g.v2=? OR g.v2='*') AND (?=0 OR d.id IS NOT NULL)`, actor.UserId, actor.DeptId, fmt.Sprintf("dept:%d", actor.DeptId), actor.DeptId)
	if err != nil {
		return scope, err
	}
	for _, role := range rows {
		if role.Code == "super_admin" && role.Domain == "*" {
			scope.all = true
		}
		switch strings.ToUpper(strings.TrimSpace(role.DataScope)) {
		case "ALL":
			scope.all = true
		case "SELF":
			scope.self = true
		case "DEPT":
			scope.department = true
		case "DEPT_AND_CHILDREN":
			scope.department = true
			if role.DepartmentPath != "" {
				scope.descendants, scope.departmentPath = true, role.DepartmentPath
			}
		}
	}
	if !scope.all && !scope.self && !scope.department && !scope.descendants {
		return scope, ErrDataScopeDenied
	}
	return scope, nil
}

func (s userDataScope) clause() (string, []any) {
	if s.all {
		return "1=1", nil
	}
	conditions, args := []string{}, []any{}
	if s.self {
		conditions, args = append(conditions, "id=?"), append(args, s.actor.UserId)
	}
	if s.department {
		conditions, args = append(conditions, "COALESCE(dept_id,0)=?"), append(args, s.actor.DeptId)
	}
	if s.descendants {
		conditions = append(conditions, "dept_id IN (SELECT id FROM sys_dept WHERE deleted_at IS NULL AND status=1 AND path LIKE ?)")
		args = append(args, s.departmentPath+"%")
	}
	return "(" + strings.Join(conditions, " OR ") + ")", args
}

func (s userDataScope) check(ctx context.Context, query scopeQuery, userId, deptId int64) error {
	if s.all || s.self && userId > 0 && userId == s.actor.UserId || s.department && deptId == s.actor.DeptId {
		return nil
	}
	if s.descendants && deptId > 0 {
		var rows []struct {
			Id int64 `db:"id"`
		}
		if err := query(ctx, &rows, "SELECT id FROM sys_dept WHERE id=? AND deleted_at IS NULL AND status=1 AND path LIKE ?", deptId, s.departmentPath+"%"); err != nil {
			return err
		}
		if len(rows) == 1 {
			return nil
		}
	}
	return ErrDataScopeDenied
}

func (m *customSysUserModel) FindScopedUser(ctx context.Context, userId int64, actor UserActor) (*SysUser, error) {
	var user SysUser
	err := m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		scope, err := resolveUserScope(ctx, session.QueryRowsCtx, actor)
		if err != nil {
			return err
		}
		if err := session.QueryRowCtx(ctx, &user, "SELECT "+sysUserRows+" FROM sys_user WHERE id=? AND deleted_at IS NULL", userId); err != nil {
			return err
		}
		return scope.check(ctx, session.QueryRowsCtx, userId, user.DeptId.Int64)
	})
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UserPatch 的指针保留更新字段存在性，不用零值推断是否更新。
type UserPatch struct {
	Nickname, RealName, Email, Phone, Remark *string
	DeptId, AvatarFileId                     *int64
	Gender, Status                           *int32
	UpdateRoles                              bool
	RoleIds                                  []int64
	PrepareAvatar                            AvatarPrepare
	PrepareStatus                            func(context.Context, int64) error
	OwnAccount                               bool
}

type AvatarPrepare func(ctx context.Context, userId, oldFileId, newFileId int64) error

func validateUserPatch(p UserPatch) error {
	if p.DeptId != nil && *p.DeptId < 0 || p.AvatarFileId != nil && *p.AvatarFileId < 0 ||
		p.Gender != nil && (*p.Gender < 0 || *p.Gender > 2) ||
		p.Status != nil && *p.Status != 1 && *p.Status != 2 {
		return ErrInvalidMutation
	}
	for _, id := range p.RoleIds {
		if id <= 0 {
			return ErrInvalidMutation
		}
	}
	return nil
}

func isSuperAdmin(ctx context.Context, s sqlx.Session, userId int64) (bool, error) {
	var count struct {
		Total int64 `db:"total"`
	}
	err := s.QueryRowCtx(ctx, &count, "SELECT COUNT(*) AS total FROM casbin_rule g JOIN sys_role r ON r.code=g.v1 JOIN sys_user u ON g.v0=CONCAT('user:',u.id) WHERE g.ptype='g' AND g.v2='*' AND r.code='super_admin' AND r.status=1 AND r.deleted_at IS NULL AND u.status=1 AND u.deleted_at IS NULL AND u.id=?", userId)
	return count.Total > 0, err
}

func (m *customSysUserModel) IsSuperAdmin(ctx context.Context, userId int64) (bool, error) {
	var count struct {
		Total int64 `db:"total"`
	}
	err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM casbin_rule g JOIN sys_role r ON r.code=g.v1 JOIN sys_user u ON g.v0=CONCAT('user:',u.id) WHERE g.ptype='g' AND g.v2='*' AND r.code='super_admin' AND r.status=1 AND r.deleted_at IS NULL AND u.status=1 AND u.deleted_at IS NULL AND u.id=?", userId)
	return count.Total > 0, err
}

// 所有用户角色变更先锁同一内置角色行，串行检查最后一个管理员约束。
func lockSuperAdminRole(ctx context.Context, s sqlx.Session) error {
	var ids []struct {
		Id int64 `db:"id"`
	}
	return s.QueryRowsCtx(ctx, &ids, "SELECT id FROM sys_role WHERE code='super_admin' ORDER BY id FOR UPDATE")
}

func validateUserDepartment(ctx context.Context, s sqlx.Session, id int64) error {
	if id == 0 {
		return nil
	}
	var count struct {
		Total int64 `db:"total"`
	}
	if err := s.QueryRowCtx(ctx, &count, "SELECT COUNT(*) AS total FROM sys_dept WHERE id=? AND status=1 AND deleted_at IS NULL", id); err != nil {
		return err
	}
	if count.Total != 1 {
		return fmt.Errorf("%w: 部门不存在或已停用", ErrInvalidMutation)
	}
	return nil
}

func replaceUserRoles(ctx context.Context, s sqlx.Session, userId, actorId, deptId int64, roleIds []int64) error {
	allowed, err := isSuperAdmin(ctx, s, actorId)
	if err != nil {
		return err
	}
	if !allowed {
		return ErrRoleAssignmentForbidden
	}
	subject := fmt.Sprintf("user:%d", userId)
	var existing []struct {
		Code string `db:"code"`
	}
	if err := s.QueryRowsCtx(ctx, &existing, "SELECT DISTINCT v1 AS code FROM casbin_rule WHERE ptype='g' AND v0=?", subject); err != nil {
		return err
	}
	old := make(map[string]bool, len(existing))
	for _, code := range existing {
		old[code.Code] = true
	}
	seen := make(map[int64]bool, len(roleIds))
	codes := make([]string, 0, len(roleIds))
	for _, id := range roleIds {
		if id <= 0 {
			return ErrInvalidMutation
		}
		if seen[id] {
			continue
		}
		seen[id] = true
		var role struct {
			Code   string `db:"code"`
			Status int64  `db:"status"`
		}
		if err := s.QueryRowCtx(ctx, &role, "SELECT code,status FROM sys_role WHERE id=? AND deleted_at IS NULL FOR UPDATE", id); err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				return fmt.Errorf("%w: 角色不存在", ErrInvalidMutation)
			}
			return err
		}
		if role.Status != 1 && !old[role.Code] {
			return fmt.Errorf("%w: 角色已停用", ErrInvalidMutation)
		}
		codes = append(codes, role.Code)
		if !old[role.Code] {
			domain := fmt.Sprintf("dept:%d", deptId)
			if role.Code == "super_admin" {
				domain = "*"
			}
			if _, err := s.ExecCtx(ctx, "INSERT INTO casbin_rule (ptype,v0,v1,v2) VALUES ('g',?,?,?)", subject, role.Code, domain); err != nil {
				return err
			}
		}
	}
	query := "DELETE FROM casbin_rule WHERE ptype='g' AND v0=?"
	args := []any{subject}
	if len(codes) > 0 {
		query += " AND v1 NOT IN (" + inPlaceholders(len(codes)) + ")"
		for _, code := range codes {
			args = append(args, code)
		}
	}
	_, err = s.ExecCtx(ctx, query, args...)
	return err
}

func preserveLastSuperAdmin(ctx context.Context, s sqlx.Session, userId int64, wasSuper bool, status int64) error {
	if !wasSuper {
		return nil
	}
	stillSuper, err := isSuperAdmin(ctx, s, userId)
	if err != nil {
		return err
	}
	if stillSuper && status == 1 {
		return nil
	}
	var count struct {
		Total int64 `db:"total"`
	}
	err = s.QueryRowCtx(ctx, &count, "SELECT COUNT(DISTINCT u.id) AS total FROM casbin_rule g JOIN sys_role r ON r.code=g.v1 JOIN sys_user u ON g.v0=CONCAT('user:',u.id) WHERE g.ptype='g' AND g.v2='*' AND r.code='super_admin' AND r.status=1 AND r.deleted_at IS NULL AND u.status=1 AND u.deleted_at IS NULL AND u.id<>?", userId)
	if err != nil {
		return err
	}
	if count.Total == 0 {
		return ErrLastSuperAdmin
	}
	return nil
}

func (m *customSysUserModel) CreateWithRoles(ctx context.Context, user *SysUser, actor UserActor, roleIds []int64) error {
	status, gender := int32(user.Status), int32(user.Gender)
	deptId, avatarId := user.DeptId.Int64, user.AvatarFileId.Int64
	if avatarId != 0 {
		return fmt.Errorf("%w: 请先创建用户，再编辑头像", ErrInvalidMutation)
	}
	if err := validateUserPatch(UserPatch{Status: &status, Gender: &gender, DeptId: &deptId, AvatarFileId: &avatarId, RoleIds: roleIds}); err != nil {
		return err
	}
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if _, err := lockActiveDepartments(ctx, s); err != nil {
			return err
		}
		if err := lockSuperAdminRole(ctx, s); err != nil {
			return err
		}
		if err := validateUserDepartment(ctx, s, deptId); err != nil {
			return err
		}
		scope, err := resolveUserScope(ctx, s.QueryRowsCtx, actor)
		if err != nil {
			return err
		}
		if err := scope.check(ctx, s.QueryRowsCtx, 0, deptId); err != nil {
			return err
		}
		var existing struct {
			Id uint64 `db:"id"`
		}
		err = s.QueryRowCtx(ctx, &existing, "SELECT id FROM sys_user WHERE username=? FOR UPDATE", user.Username)
		if err == nil {
			return ErrUsernameExists
		}
		if !errors.Is(err, sqlx.ErrNotFound) {
			return err
		}
		result, err := s.ExecCtx(ctx, "INSERT INTO sys_user (username,password_hash,nickname,real_name,email,phone,dept_id,avatar_file_id,gender,status,remark) VALUES (?,?,?,?,?,?,?,?,?,?,?)", user.Username, user.PasswordHash, user.Nickname, user.RealName, user.Email, user.Phone, user.DeptId, user.AvatarFileId, user.Gender, user.Status, user.Remark)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		user.Id = uint64(id)
		if len(roleIds) > 0 {
			if err := replaceUserRoles(ctx, s, id, actor.UserId, deptId, roleIds); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysUserIdPrefix, user.Id), cacheGozeroAdminNewSysUserUsernamePrefix+user.Username)
}

func (m *customSysUserModel) UpdatePartial(ctx context.Context, userId int64, actor UserActor, p UserPatch) error {
	if userId <= 0 || actor.UserId <= 0 {
		return ErrInvalidMutation
	}
	if err := validateUserPatch(p); err != nil {
		return err
	}
	var username string
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if _, err := lockActiveDepartments(ctx, s); err != nil {
			return err
		}
		if err := lockSuperAdminRole(ctx, s); err != nil {
			return err
		}
		var user SysUser
		if err := s.QueryRowCtx(ctx, &user, "SELECT "+sysUserRows+" FROM sys_user WHERE id=? AND deleted_at IS NULL FOR UPDATE", userId); err != nil {
			return err
		}
		scope, err := resolveUserScope(ctx, s.QueryRowsCtx, actor)
		if err != nil {
			return err
		}
		if err := scope.check(ctx, s.QueryRowsCtx, userId, user.DeptId.Int64); err != nil {
			return err
		}
		username = user.Username
		wasSuper, err := isSuperAdmin(ctx, s, userId)
		if err != nil {
			return err
		}
		deptId := user.DeptId.Int64
		if p.DeptId != nil {
			deptId = *p.DeptId
			if deptId != user.DeptId.Int64 {
				allowed, err := isSuperAdmin(ctx, s, actor.UserId)
				if err != nil {
					return err
				}
				if !allowed {
					return ErrRoleAssignmentForbidden
				}
			}
			if err := validateUserDepartment(ctx, s, deptId); err != nil {
				return err
			}
		}
		if p.UpdateRoles {
			if err := validateUserDepartment(ctx, s, deptId); err != nil {
				return err
			}
			if err := replaceUserRoles(ctx, s, userId, actor.UserId, deptId, p.RoleIds); err != nil {
				return err
			}
		}
		if p.DeptId != nil && deptId != user.DeptId.Int64 {
			subject, oldDomain, newDomain := fmt.Sprintf("user:%d", userId), fmt.Sprintf("dept:%d", user.DeptId.Int64), fmt.Sprintf("dept:%d", deptId)
			if _, err := s.ExecCtx(ctx, "INSERT IGNORE INTO casbin_rule (ptype,v0,v1,v2,v3,v4,v5) SELECT ptype,v0,v1,?,v3,v4,v5 FROM casbin_rule WHERE ptype='g' AND v0=? AND v2=?", newDomain, subject, oldDomain); err != nil {
				return err
			}
			if _, err := s.ExecCtx(ctx, "DELETE FROM casbin_rule WHERE ptype='g' AND v0=? AND v2=?", subject, oldDomain); err != nil {
				return err
			}
		}
		status := user.Status
		if p.Status != nil {
			status = int64(*p.Status)
		}
		if err := preserveLastSuperAdmin(ctx, s, userId, wasSuper, status); err != nil {
			return err
		}
		// 重新启用前也撤销旧会话，避免停用期间保持离线的令牌恢复有效。
		if status != user.Status && p.PrepareStatus != nil {
			if err := p.PrepareStatus(ctx, userId); err != nil {
				return err
			}
		}
		sets, args := make([]string, 0, 10), make([]any, 0, 11)
		add := func(column string, value any) { sets = append(sets, column+"=?"); args = append(args, value) }
		if p.Nickname != nil {
			add("nickname", *p.Nickname)
		}
		if p.RealName != nil {
			add("real_name", *p.RealName)
		}
		if p.Email != nil {
			add("email", *p.Email)
		}
		if p.Phone != nil {
			add("phone", *p.Phone)
		}
		if p.Remark != nil {
			add("remark", *p.Remark)
		}
		if p.Gender != nil {
			add("gender", *p.Gender)
		}
		if p.Status != nil {
			add("status", *p.Status)
		}
		if p.DeptId != nil {
			add("dept_id", sql.NullInt64{Int64: deptId, Valid: deptId > 0})
		}
		if p.AvatarFileId != nil {
			if p.PrepareAvatar != nil {
				if err := p.PrepareAvatar(ctx, userId, user.AvatarFileId.Int64, *p.AvatarFileId); err != nil {
					return err
				}
			}
			add("avatar_file_id", sql.NullInt64{Int64: *p.AvatarFileId, Valid: *p.AvatarFileId > 0})
		}
		if len(sets) == 0 {
			return nil
		}
		args = append(args, userId)
		_, err = s.ExecCtx(ctx, "UPDATE sys_user SET "+strings.Join(sets, ",")+" WHERE id=? AND deleted_at IS NULL", args...)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysUserIdPrefix, userId), cacheGozeroAdminNewSysUserUsernamePrefix+username)
}

// 事务结束后再按用户当前真实头像对账，防止较早请求误删较晚请求正在使用的引用。
func (m *customSysUserModel) ReconcileAvatar(ctx context.Context, userId int64, reconcile func(context.Context, int64) error) error {
	return m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		var row struct {
			AvatarFileId int64        `db:"avatar_file_id"`
			DeletedAt    sql.NullTime `db:"deleted_at"`
		}
		err := s.QueryRowCtx(ctx, &row, "SELECT COALESCE(avatar_file_id,0) AS avatar_file_id,deleted_at FROM sys_user WHERE id=? FOR UPDATE", userId)
		if err != nil && !errors.Is(err, sqlx.ErrNotFound) {
			return err
		}
		keep := row.AvatarFileId
		if errors.Is(err, sqlx.ErrNotFound) || row.DeletedAt.Valid {
			keep = 0
		}
		return reconcile(ctx, keep)
	})
}

func (m *customSysUserModel) ListUserRoleIds(ctx context.Context, userIds []int64) (map[int64][]int64, error) {
	result := make(map[int64][]int64, len(userIds))
	if len(userIds) == 0 {
		return result, nil
	}
	args := make([]any, 0, len(userIds))
	for _, id := range userIds {
		args = append(args, fmt.Sprintf("user:%d", id))
		result[id] = []int64{}
	}
	var rows []struct {
		UserId int64 `db:"user_id"`
		RoleId int64 `db:"role_id"`
	}
	err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT DISTINCT CAST(SUBSTRING(g.v0,6) AS UNSIGNED) AS user_id,r.id AS role_id FROM casbin_rule g JOIN sys_role r ON r.code=g.v1 WHERE g.ptype='g' AND g.v0 IN ("+inPlaceholders(len(args))+") AND r.deleted_at IS NULL ORDER BY user_id,role_id", args...)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		result[r.UserId] = append(result[r.UserId], r.RoleId)
	}
	return result, nil
}

func (m *customSysUserModel) DeleteWithRoles(ctx context.Context, userId int64, actor UserActor) error {
	if userId <= 0 {
		return ErrInvalidMutation
	}
	var username string
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		if _, err := lockActiveDepartments(ctx, s); err != nil {
			return err
		}
		if err := lockSuperAdminRole(ctx, s); err != nil {
			return err
		}
		var user SysUser
		if err := s.QueryRowCtx(ctx, &user, "SELECT "+sysUserRows+" FROM sys_user WHERE id=? FOR UPDATE", userId); err != nil {
			return err
		}
		scope, err := resolveUserScope(ctx, s.QueryRowsCtx, actor)
		if err != nil {
			return err
		}
		if err := scope.check(ctx, s.QueryRowsCtx, userId, user.DeptId.Int64); err != nil {
			return err
		}
		if user.DeletedAt.Valid {
			return sqlx.ErrNotFound
		}
		username = user.Username
		wasSuper, err := isSuperAdmin(ctx, s, userId)
		if err != nil {
			return err
		}
		if err := preserveLastSuperAdmin(ctx, s, userId, wasSuper, 2); err != nil {
			return err
		}
		if _, err = s.ExecCtx(ctx, "UPDATE sys_user SET deleted_at=NOW() WHERE id=? AND deleted_at IS NULL", userId); err != nil {
			return err
		}
		_, err = s.ExecCtx(ctx, "DELETE FROM casbin_rule WHERE ptype='g' AND v0=?", fmt.Sprintf("user:%d", userId))
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysUserIdPrefix, userId), cacheGozeroAdminNewSysUserUsernamePrefix+username)
}

// UserListFilter 用户列表筛选（空值/0 不过滤）
type UserListFilter struct {
	Actor    UserActor
	Username string
	Nickname string
	DeptId   int64
	Status   int64
}

// 密码变更只写密码列；锁内检查避免旧用户快照恢复已删除用户或覆盖头像引用。
// expectedHash 为空表示管理员重置，否则用于自助改密的并发校验。
func (m *customSysUserModel) UpdatePassword(ctx context.Context, userId uint64, actor UserActor, expectedHash, newHash string, revoke func(context.Context, int64) error) error {
	if userId == 0 || newHash == "" {
		return ErrInvalidMutation
	}
	var username string
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		var user struct {
			Username     string `db:"username"`
			PasswordHash string `db:"password_hash"`
			Status       int64  `db:"status"`
			DeptId       int64  `db:"dept_id"`
		}
		if err := s.QueryRowCtx(ctx, &user, "SELECT username,password_hash,status,COALESCE(dept_id,0) AS dept_id FROM sys_user WHERE id=? AND deleted_at IS NULL FOR UPDATE", userId); err != nil {
			return err
		}
		scope, err := resolveUserScope(ctx, s.QueryRowsCtx, actor)
		if err != nil {
			return err
		}
		if err := scope.check(ctx, s.QueryRowsCtx, int64(userId), user.DeptId); err != nil {
			return err
		}
		if user.Status != 1 {
			return ErrUserDisabled
		}
		if expectedHash != "" && user.PasswordHash != expectedHash {
			return ErrPasswordChanged
		}
		// 与登录签发使用同一行锁；先撤销再提交，避免旧密码登录跨过撤销窗口。
		if err := revoke(ctx, int64(userId)); err != nil {
			return err
		}
		username = user.Username
		_, err = s.ExecCtx(ctx, "UPDATE sys_user SET password_hash=?,updated_at=NOW() WHERE id=? AND deleted_at IS NULL AND status=1", newHash, userId)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysUserIdPrefix, userId), cacheGozeroAdminNewSysUserUsernamePrefix+username)
}

// 密码校验可在锁外执行，但签发会话前必须锁行核对快照，和改密、停用串行。
func (m *customSysUserModel) WithLoginUser(ctx context.Context, userId uint64, expectedHash string, issue func(context.Context, *SysUser) error) error {
	return m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		var user SysUser
		if err := s.QueryRowCtx(ctx, &user, "SELECT "+sysUserRows+" FROM sys_user WHERE id=? AND deleted_at IS NULL FOR UPDATE", userId); err != nil {
			return err
		}
		if user.Status != 1 {
			return ErrUserDisabled
		}
		if user.PasswordHash != expectedHash {
			return ErrPasswordChanged
		}
		return issue(ctx, &user)
	})
}

// UpdateLastLogin 只更新登录字段，避免并发软删除后旧用户快照整行回写导致复活。
func (m *customSysUserModel) UpdateLastLogin(ctx context.Context, userId uint64, ip string) error {
	user, err := m.FindOne(ctx, userId)
	if err != nil {
		return err
	}
	idKey := fmt.Sprintf("%s%d", cacheGozeroAdminNewSysUserIdPrefix, userId)
	usernameKey := cacheGozeroAdminNewSysUserUsernamePrefix + user.Username
	_, err = m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, "UPDATE sys_user SET last_login_at=?,last_login_ip=? WHERE id=? AND deleted_at IS NULL AND status=1", time.Now(), ip, userId)
	}, idKey, usernameKey)
	return err
}

// UserRow 用户列表行（可空列 COALESCE、时间列 UNIX_TIMESTAMP，均带 AS 别名防静默零值）
type UserRow struct {
	Id           int64  `db:"id"`
	Username     string `db:"username"`
	Nickname     string `db:"nickname"`
	RealName     string `db:"real_name"`
	Email        string `db:"email"`
	Phone        string `db:"phone"`
	DeptId       int64  `db:"dept_id"`
	AvatarFileId int64  `db:"avatar_file_id"`
	Gender       int32  `db:"gender"`
	Status       int32  `db:"status"`
	Remark       string `db:"remark"`
	LastLoginAt  int64  `db:"last_login_at"`
	LastLoginIp  string `db:"last_login_ip"`
	CreatedAt    int64  `db:"created_at"`
	UpdatedAt    int64  `db:"updated_at"`
}

// NewSysUserModel returns a model for the database table.
func NewSysUserModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysUserModel {
	return &customSysUserModel{
		defaultSysUserModel: newSysUserModel(conn, c, opts...),
	}
}

func (m *customSysUserModel) ListUserPage(ctx context.Context, f UserListFilter, page, pageSize int64) ([]*UserRow, int64, error) {
	scope, err := resolveUserScope(ctx, m.QueryRowsNoCacheCtx, f.Actor)
	if err != nil {
		return nil, 0, err
	}
	condition, args := scope.clause()
	where := "deleted_at IS NULL AND " + condition
	if f.Username != "" {
		where += " AND username LIKE ?"
		args = append(args, "%"+f.Username+"%")
	}
	if f.Nickname != "" {
		where += " AND nickname LIKE ?"
		args = append(args, "%"+f.Nickname+"%")
	}
	if f.DeptId > 0 {
		where += " AND dept_id=?"
		args = append(args, f.DeptId)
	}
	if f.Status > 0 {
		where += " AND status=?"
		args = append(args, f.Status)
	}
	var total struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) AS total FROM sys_user WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var rows []*UserRow
	err = m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id, username, nickname, real_name, email, phone, COALESCE(dept_id,0) AS dept_id, COALESCE(avatar_file_id,0) AS avatar_file_id, "+
			"gender, status, remark, COALESCE(UNIX_TIMESTAMP(last_login_at),0) AS last_login_at, last_login_ip, "+
			"UNIX_TIMESTAMP(created_at) AS created_at, UNIX_TIMESTAMP(updated_at) AS updated_at "+
			"FROM sys_user WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...)
	return rows, total.Total, err
}
