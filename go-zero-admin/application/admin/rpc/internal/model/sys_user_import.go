package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var (
	ErrImportDepartment = errors.New("部门不存在或已停用")
	ErrImportDuplicate  = errors.New("同一 CSV 内的账号重复")
	ErrImportConflict   = errors.New("账号已被其他请求创建")
)

// 此值无法通过 bcrypt 校验。管理员须重置密码才能为导入账号建立登录凭据。
const userImportPasswordHash = "!"

type UserImportCandidate struct {
	Row      int64  `json:"row"`
	Username string `json:"username"`
	DeptId   int64  `json:"dept_id"`
	RealName string `json:"-"`
}

func (m *customSysUserModel) ImportUsers(ctx context.Context, actor UserActor, rows []UserImportCandidate, confirmRows []int64) (map[int64]error, error) {
	if len(confirmRows) == 0 {
		return checkUserImport(ctx, m.QueryRowsNoCacheCtx, actor, rows)
	}

	selected := make(map[int64]bool, len(confirmRows))
	for _, row := range confirmRows {
		selected[row] = true
	}
	cacheKeys := make([]string, 0, len(confirmRows)*2)
	var issues map[int64]error
	err := m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		// 与普通新建共用锁顺序。确认时共享同一事务中的数据范围和部门状态。
		if _, err := lockActiveDepartments(ctx, session); err != nil {
			return err
		}
		if err := lockSuperAdminRole(ctx, session); err != nil {
			return err
		}
		var err error
		issues, err = checkUserImport(ctx, session.QueryRowsCtx, actor, rows)
		if err != nil {
			return err
		}
		for _, row := range rows {
			if !selected[row.Row] || issues[row.Row] != nil {
				continue
			}
			user := &SysUser{
				Username: row.Username, RealName: row.RealName, Status: 2, PasswordHash: userImportPasswordHash,
				DeptId: sql.NullInt64{Int64: row.DeptId, Valid: row.DeptId > 0},
			}
			if err := insertUser(ctx, session, user); err != nil {
				if errors.Is(err, ErrUsernameExists) {
					issues[row.Row] = ErrImportConflict
					continue
				}
				return err
			}
			cacheKeys = append(cacheKeys, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysUserIdPrefix, user.Id), cacheGozeroAdminNewSysUserUsernamePrefix+user.Username)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(cacheKeys) > 0 {
		return issues, committedCacheError(m.DelCacheCtx(ctx, cacheKeys...))
	}
	return issues, nil
}

func checkUserImport(ctx context.Context, query scopeQuery, actor UserActor, rows []UserImportCandidate) (map[int64]error, error) {
	scope, err := resolveUserScope(ctx, query, actor)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(rows)
	if err != nil {
		return nil, err
	}
	// 候选账号的比较规则与 sys_user 唯一索引一致，软删除记录仍占用账号。
	condition, args := scope.clause()
	var checked []struct {
		Row        int64 `db:"row_no"`
		Permitted  bool  `db:"permitted"`
		DeptActive bool  `db:"dept_active"`
		Existing   bool  `db:"existing"`
		Repeated   bool  `db:"repeated"`
	}
	err = query(ctx, &checked, `WITH ranked AS (
SELECT row_no, dept_id, 0 AS id, CONVERT(username USING utf8mb4) COLLATE utf8mb4_0900_ai_ci AS username,
COUNT(*) OVER (PARTITION BY CONVERT(username USING utf8mb4) COLLATE utf8mb4_0900_ai_ci) AS same_name
FROM JSON_TABLE(?, '$[*]' COLUMNS(row_no BIGINT PATH '$.row', dept_id BIGINT PATH '$.dept_id', username VARCHAR(64) PATH '$.username')) AS batch
)
SELECT row_no, (`+condition+`) AS permitted,
(dept_id=0 OR EXISTS(SELECT 1 FROM sys_dept d WHERE d.id=r.dept_id AND d.status=1 AND d.deleted_at IS NULL)) AS dept_active,
EXISTS(SELECT 1 FROM sys_user u WHERE u.username=r.username) AS existing,
(same_name>1) AS repeated FROM ranked r`, append([]any{string(encoded)}, args...)...)
	if err != nil {
		return nil, err
	}
	if len(checked) != len(rows) {
		return nil, errors.New("用户导入预检结果行数不完整")
	}
	issues := make(map[int64]error)
	for _, entry := range checked {
		switch {
		case !entry.Permitted:
			issues[entry.Row] = ErrDataScopeDenied
		case !entry.DeptActive:
			issues[entry.Row] = ErrImportDepartment
		case entry.Repeated:
			issues[entry.Row] = ErrImportDuplicate
		case entry.Existing:
			issues[entry.Row] = ErrUsernameExists
		}
	}
	return issues, nil
}
