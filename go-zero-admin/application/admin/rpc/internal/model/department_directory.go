package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type DepartmentDirectory struct {
	DeptName string
	List     []*DepartmentDirectoryMember
	Total    int64
}

type DepartmentDirectoryMember struct {
	Id       int64  `db:"id"`
	Name     string `db:"name"`
	IsLeader bool   `db:"is_leader"`
}

const directoryDisplayName = `COALESCE(NULLIF(TRIM(real_name),''),NULLIF(TRIM(nickname),''),'未设置姓名')`

var directoryLikeEscaper = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")

func (m *customSysDeptModel) ListDepartmentDirectory(ctx context.Context, actor UserActor, keyword string, page, pageSize int64) (*DepartmentDirectory, error) {
	if actor.UserId <= 0 || actor.DeptId <= 0 {
		return nil, ErrDataScopeDenied
	}
	if !utf8.ValidString(keyword) || utf8.RuneCountInString(keyword) > 64 || page < 0 || page > 100000 || pageSize < 0 || pageSize > 100 {
		return nil, ErrInvalidMutation
	}
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	keyword = strings.TrimSpace(keyword)
	offset := (page - 1) * pageSize
	result := &DepartmentDirectory{List: make([]*DepartmentDirectoryMember, 0)}
	err := m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		// 在同一读事务中校验成员身份、部门状态并读取成员，避免把旧身份与更新后的数据混用。
		all, err := isSuperAdmin(ctx, session, actor.UserId)
		if err != nil {
			return err
		}
		var dept struct {
			Name         string `db:"name"`
			LeaderUserId int64  `db:"leader_user_id"`
		}
		query := `SELECT d.name,d.leader_user_id FROM sys_dept d
JOIN sys_user viewer ON viewer.id=? AND viewer.status=1 AND viewer.deleted_at IS NULL
WHERE d.id=? AND d.status=1 AND d.deleted_at IS NULL`
		args := []any{actor.UserId, actor.DeptId}
		if !all {
			query += departmentMembershipCondition
			args = append(args, actor.UserId, fmt.Sprintf("user:%d", actor.UserId))
		}
		if err := session.QueryRowCtx(ctx, &dept, query, args...); err != nil {
			if errors.Is(err, sqlx.ErrNotFound) {
				return ErrDataScopeDenied
			}
			return err
		}
		result.DeptName = dept.Name

		where := " FROM sys_user WHERE dept_id=? AND status=1 AND deleted_at IS NULL"
		args = []any{actor.DeptId}
		if keyword != "" {
			where += " AND " + directoryDisplayName + " LIKE ? ESCAPE '!'"
			args = append(args, "%"+directoryLikeEscaper.Replace(keyword)+"%")
		}
		if err := session.QueryRowCtx(ctx, &result.Total, "SELECT COUNT(*)"+where, args...); err != nil {
			return err
		}
		if result.Total <= offset {
			return nil
		}
		query = "SELECT id," + directoryDisplayName + " AS name,id=? AS is_leader" + where + " ORDER BY is_leader DESC,id LIMIT ? OFFSET ?"
		memberArgs := append([]any{dept.LeaderUserId}, args...)
		return session.QueryRowsCtx(ctx, &result.List, query, append(memberArgs, pageSize, offset)...)
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
