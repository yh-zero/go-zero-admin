package model

import (
	"context"

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
	}

	customSysUserModel struct {
		*defaultSysUserModel
	}
)

// UserListFilter 用户列表筛选（空值/0 不过滤）
type UserListFilter struct {
	Username string
	Nickname string
	DeptId   int64
	Status   int64
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
	where := "deleted_at IS NULL"
	var args []any
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
	var total int64
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) FROM sys_user WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var rows []*UserRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id, username, nickname, real_name, email, phone, COALESCE(dept_id,0) AS dept_id, COALESCE(avatar_file_id,0) AS avatar_file_id, "+
			"gender, status, remark, COALESCE(UNIX_TIMESTAMP(last_login_at),0) AS last_login_at, last_login_ip, "+
			"UNIX_TIMESTAMP(created_at) AS created_at, UNIX_TIMESTAMP(updated_at) AS updated_at "+
			"FROM sys_user WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...)
	return rows, total, err
}
