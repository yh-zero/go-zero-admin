package model

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysDeptModel = (*customSysDeptModel)(nil)

type (
	// SysDeptModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysDeptModel.
	SysDeptModel interface {
		sysDeptModel
		// ListDeptTree 全量查有效部门（树用），status 为 0 时不过滤
		ListDeptTree(ctx context.Context, status int64) ([]*DeptRow, error)
		// CollectSubtreeIds 收集子树全部 id（含 root，带防环 visited 集合）；root 不存在或已删时返回 sqlx.ErrNotFound
		CollectSubtreeIds(ctx context.Context, rootId uint64) ([]int64, error)
		// SoftDelete 软删并清 id 缓存；目标不存在或已删时返回 sqlx.ErrNotFound
		SoftDelete(ctx context.Context, id uint64) error
		CreateDepartment(ctx context.Context, dept *DeptRow, actor UserActor) error
		UpdateDepartment(ctx context.Context, id int64, patch DeptPatch, actor UserActor) error
		DeleteDepartment(ctx context.Context, id int64) error
		ListMyDepartments(ctx context.Context, userId int64, all bool) ([]*DeptRow, error)
		ListDepartmentMemberCounts(ctx context.Context, actor UserActor) (map[int64]int64, error)
	}

	customSysDeptModel struct {
		*defaultSysDeptModel
	}
)

// DeptRow 部门行（时间列 UNIX_TIMESTAMP 转 int64）
type DeptRow struct {
	Id           int64  `db:"id"`
	ParentId     int64  `db:"parent_id"`
	Name         string `db:"name"`
	Leader       string `db:"leader"`
	LeaderUserId int64  `db:"leader_user_id"`
	Phone        string `db:"phone"`
	Email        string `db:"email"`
	Sort         int32  `db:"sort"`
	Status       int32  `db:"status"`
	Path         string `db:"path"`
	CreatedAt    int64  `db:"created_at"`
}

type DeptPatch struct {
	ParentId                   *int64
	LeaderUserId               *int64
	Name, Leader, Phone, Email *string
	Sort, Status               *int32
}

func departmentPath(nodes map[int64]*DeptRow, id int64) (string, error) {
	seen := map[int64]bool{}
	ids := []int64{}
	for id != 0 {
		if seen[id] {
			return "", fmt.Errorf("%w: 不能挂载到自身或子部门", ErrInvalidMutation)
		}
		seen[id] = true
		node, ok := nodes[id]
		if !ok {
			return "", fmt.Errorf("%w: 父部门不存在", ErrInvalidMutation)
		}
		ids = append(ids, id)
		id = node.ParentId
	}
	var path strings.Builder
	path.WriteByte('/')
	for i := len(ids) - 1; i >= 0; i-- {
		fmt.Fprintf(&path, "%d/", ids[i])
	}
	if path.Len() > 512 {
		return "", fmt.Errorf("%w: 部门层级过深", ErrInvalidMutation)
	}
	return path.String(), nil
}

func deptNodes(rows []*DeptRow) map[int64]*DeptRow {
	nodes := make(map[int64]*DeptRow, len(rows))
	for _, r := range rows {
		nodes[r.Id] = r
	}
	return nodes
}

func (m *customSysDeptModel) CreateDepartment(ctx context.Context, dept *DeptRow, actor UserActor) error {
	if dept.ParentId < 0 || dept.LeaderUserId < 0 || strings.TrimSpace(dept.Name) == "" || (dept.Status != 1 && dept.Status != 2) || dept.LeaderUserId > 0 && strings.TrimSpace(dept.Leader) != "" {
		return ErrInvalidMutation
	}
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		rows, err := lockActiveDepartments(ctx, s)
		if err != nil {
			return err
		}
		nodes := deptNodes(rows)
		if dept.ParentId > 0 {
			parent := nodes[dept.ParentId]
			if parent == nil || parent.Status != 1 {
				return fmt.Errorf("%w: 父部门不存在或已停用", ErrInvalidMutation)
			}
		}
		if dept.LeaderUserId > 0 {
			if err := validateDepartmentLeader(ctx, s, dept.LeaderUserId, actor); err != nil {
				return err
			}
			dept.Leader = ""
		}
		result, err := s.ExecCtx(ctx, "INSERT INTO sys_dept (parent_id,name,leader,leader_user_id,phone,email,sort,status,path) VALUES (?,?,?,?,?,?,?,?,?)", dept.ParentId, dept.Name, dept.Leader, dept.LeaderUserId, dept.Phone, dept.Email, dept.Sort, dept.Status, "")
		if err != nil {
			return err
		}
		dept.Id, err = result.LastInsertId()
		if err != nil {
			return err
		}
		nodes[dept.Id] = dept
		dept.Path, err = departmentPath(nodes, dept.Id)
		if err != nil {
			return err
		}
		_, err = s.ExecCtx(ctx, "UPDATE sys_dept SET path=? WHERE id=?", dept.Path, dept.Id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminSysDeptIdPrefix, dept.Id))
}

func (m *customSysDeptModel) UpdateDepartment(ctx context.Context, id int64, p DeptPatch, actor UserActor) error {
	if id <= 0 || p.ParentId != nil && *p.ParentId < 0 || p.LeaderUserId != nil && *p.LeaderUserId < 0 || p.Name != nil && strings.TrimSpace(*p.Name) == "" || p.Status != nil && *p.Status != 1 && *p.Status != 2 {
		return ErrInvalidMutation
	}
	if p.LeaderUserId != nil && p.Leader != nil && strings.TrimSpace(*p.Leader) != "" {
		return fmt.Errorf("%w: 不能同时填写负责人账号和历史负责人文本", ErrInvalidMutation)
	}
	var affected []int64
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		rows, err := lockActiveDepartments(ctx, s)
		if err != nil {
			return err
		}
		nodes := deptNodes(rows)
		node := nodes[id]
		if node == nil {
			return sqlx.ErrNotFound
		}
		if p.Status != nil && *p.Status == 2 && node.Status != 2 {
			if err := ensureDepartmentEmpty(ctx, s, rows, id, "停用"); err != nil {
				return err
			}
		}
		if p.ParentId != nil {
			if *p.ParentId > 0 {
				parent := nodes[*p.ParentId]
				if parent == nil || parent.Status != 1 {
					return fmt.Errorf("%w: 父部门不存在或已停用", ErrInvalidMutation)
				}
			}
			node.ParentId = *p.ParentId
		}
		if p.Status != nil && *p.Status == 1 && node.ParentId > 0 {
			parent := nodes[node.ParentId]
			if parent == nil || parent.Status != 1 {
				return fmt.Errorf("%w: 请先启用上级部门", ErrInvalidMutation)
			}
		}
		if p.Name != nil {
			node.Name = *p.Name
		}
		if p.LeaderUserId != nil {
			if *p.LeaderUserId > 0 {
				if err := validateDepartmentLeader(ctx, s, *p.LeaderUserId, actor); err != nil {
					return err
				}
			}
			node.LeaderUserId, node.Leader = *p.LeaderUserId, ""
		} else if p.Leader != nil {
			if node.LeaderUserId > 0 {
				return fmt.Errorf("%w: 已关联负责人账号，请通过负责人账号修改或清空", ErrInvalidMutation)
			}
			node.Leader = *p.Leader
		}
		if p.Phone != nil {
			node.Phone = *p.Phone
		}
		if p.Email != nil {
			node.Email = *p.Email
		}
		if p.Sort != nil {
			node.Sort = *p.Sort
		}
		if p.Status != nil {
			node.Status = *p.Status
		}
		node.Path, err = departmentPath(nodes, id)
		if err != nil {
			return err
		}
		if _, err = s.ExecCtx(ctx, "UPDATE sys_dept SET parent_id=?,name=?,leader=?,leader_user_id=?,phone=?,email=?,sort=?,status=?,path=? WHERE id=? AND deleted_at IS NULL", node.ParentId, node.Name, node.Leader, node.LeaderUserId, node.Phone, node.Email, node.Sort, node.Status, node.Path, id); err != nil {
			return err
		}
		affected = []int64{id}
		if p.ParentId == nil {
			return nil
		}
		children := make(map[int64][]int64, len(rows))
		for _, r := range rows {
			children[r.ParentId] = append(children[r.ParentId], r.Id)
		}
		for i := 0; i < len(affected); i++ {
			for _, childId := range children[affected[i]] {
				path, err := departmentPath(nodes, childId)
				if err != nil {
					return err
				}
				if _, err = s.ExecCtx(ctx, "UPDATE sys_dept SET path=? WHERE id=? AND deleted_at IS NULL", path, childId); err != nil {
					return err
				}
				affected = append(affected, childId)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(affected))
	for _, affectedId := range affected {
		keys = append(keys, fmt.Sprintf("%s%d", cacheGozeroAdminSysDeptIdPrefix, affectedId))
	}
	return m.DelCacheCtx(ctx, keys...)
}

func (m *customSysDeptModel) DeleteDepartment(ctx context.Context, id int64) error {
	if id <= 0 {
		return ErrInvalidMutation
	}
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		rows, err := lockActiveDepartments(ctx, s)
		if err != nil {
			return err
		}
		if deptNodes(rows)[id] == nil {
			return sqlx.ErrNotFound
		}
		if err = ensureDepartmentEmpty(ctx, s, rows, id, "删除"); err != nil {
			return err
		}
		_, err = s.ExecCtx(ctx, "UPDATE sys_dept SET deleted_at=NOW() WHERE id=? AND deleted_at IS NULL", id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminSysDeptIdPrefix, id))
}

func validateDepartmentLeader(ctx context.Context, s sqlx.Session, userId int64, actor UserActor) error {
	scope, err := resolveUserScope(ctx, s.QueryRowsCtx, actor)
	if err != nil {
		return err
	}
	var user struct {
		Id     int64 `db:"id"`
		DeptId int64 `db:"dept_id"`
	}
	err = s.QueryRowCtx(ctx, &user, "SELECT id,COALESCE(dept_id,0) AS dept_id FROM sys_user WHERE id=? AND status=1 AND deleted_at IS NULL FOR UPDATE", userId)
	if errors.Is(err, sqlx.ErrNotFound) {
		return fmt.Errorf("%w: 负责人不存在或已停用", ErrInvalidMutation)
	}
	if err != nil {
		return err
	}
	return scope.check(ctx, s.QueryRowsCtx, user.Id, user.DeptId)
}

// 用户归属和授权写入也先锁部门树，关联检查与部门状态变更因此保持一致。
func ensureDepartmentEmpty(ctx context.Context, s sqlx.Session, rows []*DeptRow, id int64, operation string) error {
	for _, row := range rows {
		if row.ParentId == id {
			return fmt.Errorf("%w: 存在子部门，请先处理后再%s", ErrInvalidMutation, operation)
		}
	}
	var count struct {
		Total int64 `db:"total"`
	}
	if err := s.QueryRowCtx(ctx, &count, "SELECT COUNT(*) AS total FROM sys_user WHERE dept_id=? AND deleted_at IS NULL", id); err != nil {
		return err
	}
	if count.Total > 0 {
		return fmt.Errorf("%w: 部门下存在用户，请先转移后再%s", ErrInvalidMutation, operation)
	}
	if err := s.QueryRowCtx(ctx, &count, "SELECT COUNT(*) AS total FROM casbin_rule WHERE ptype='g' AND v2=?", fmt.Sprintf("dept:%d", id)); err != nil {
		return err
	}
	if count.Total > 0 {
		return fmt.Errorf("%w: 部门仍有成员授权，请先处理后再%s", ErrInvalidMutation, operation)
	}
	return nil
}

const departmentInfoColumns = `d.id,d.parent_id,d.name,d.leader_user_id,
CASE WHEN d.leader_user_id=0 THEN d.leader
 WHEN leader.id IS NULL OR leader.deleted_at IS NOT NULL THEN '负责人已删除'
 ELSE CONCAT(COALESCE(NULLIF(leader.real_name,''),NULLIF(leader.nickname,''),leader.username),IF(leader.status=1,'','（已停用）')) END AS leader,
d.phone,d.email,d.sort,d.status,d.path,UNIX_TIMESTAMP(d.created_at) AS created_at`

func (m *customSysDeptModel) ListMyDepartments(ctx context.Context, userId int64, all bool) ([]*DeptRow, error) {
	rows := make([]*DeptRow, 0)
	query := "SELECT " + departmentInfoColumns + " FROM sys_dept d LEFT JOIN sys_user leader ON leader.id=d.leader_user_id WHERE d.status=1 AND d.deleted_at IS NULL"
	var args []any
	if !all {
		query += " AND (d.id=(SELECT u.dept_id FROM sys_user u WHERE u.id=? AND u.status=1 AND u.deleted_at IS NULL) OR EXISTS (SELECT 1 FROM casbin_rule g JOIN sys_role r ON r.code=g.v1 WHERE g.ptype='g' AND g.v0=? AND g.v2=CONCAT('dept:',d.id) AND r.status=1 AND r.deleted_at IS NULL))"
		args = []any{userId, fmt.Sprintf("user:%d", userId)}
	}
	err := m.QueryRowsNoCacheCtx(ctx, &rows, query+" ORDER BY d.sort,d.id", args...)
	return rows, err
}

// NewSysDeptModel returns a model for the database table.
func NewSysDeptModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysDeptModel {
	return &customSysDeptModel{
		defaultSysDeptModel: newSysDeptModel(conn, c, opts...),
	}
}

// 部门结构写入统一按主键锁住现有树；创建、移动、删除和用户归属修改遵循同一顺序。
func lockActiveDepartments(ctx context.Context, s sqlx.Session) ([]*DeptRow, error) {
	var rows []*DeptRow
	err := s.QueryRowsCtx(ctx, &rows, "SELECT id,parent_id,name,leader,leader_user_id,phone,email,sort,status,path,UNIX_TIMESTAMP(created_at) AS created_at FROM sys_dept WHERE deleted_at IS NULL ORDER BY id FOR UPDATE")
	return rows, err
}

func (m *customSysDeptModel) CollectSubtreeIds(ctx context.Context, rootId uint64) ([]int64, error) {
	root, err := m.FindOne(ctx, rootId)
	if err != nil {
		return nil, err
	}
	if root.DeletedAt.Valid {
		return nil, sqlx.ErrNotFound
	}
	visited := map[int64]struct{}{int64(rootId): {}}
	ids := []int64{int64(rootId)}
	frontier := []int64{int64(rootId)}
	for len(frontier) > 0 {
		args := make([]any, 0, len(frontier))
		for _, id := range frontier {
			args = append(args, id)
		}
		var children []int64
		if err := m.QueryRowsNoCacheCtx(ctx, &children,
			"SELECT id FROM sys_dept WHERE parent_id IN ("+inPlaceholders(len(frontier))+") AND deleted_at IS NULL", args...); err != nil {
			return nil, err
		}
		next := make([]int64, 0, len(children))
		for _, id := range children {
			if _, seen := visited[id]; seen {
				continue
			}
			visited[id] = struct{}{}
			next = append(next, id)
		}
		ids = append(ids, next...)
		frontier = next
	}
	return ids, nil
}

func (m *customSysDeptModel) SoftDelete(ctx context.Context, id uint64) error {
	if _, err := m.FindOne(ctx, id); err != nil {
		return err
	}
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		_, err := s.ExecCtx(ctx, "UPDATE sys_dept SET deleted_at=NOW() WHERE id=? AND deleted_at IS NULL", id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminSysDeptIdPrefix, id))
}

func (m *customSysDeptModel) ListDeptTree(ctx context.Context, status int64) ([]*DeptRow, error) {
	where := "d.deleted_at IS NULL"
	var args []any
	if status > 0 {
		where += " AND d.status=?"
		args = append(args, status)
	}
	var rows []*DeptRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT "+departmentInfoColumns+" FROM sys_dept d LEFT JOIN sys_user leader ON leader.id=d.leader_user_id WHERE "+where+" ORDER BY d.sort,d.id", args...)
	return rows, err
}

func (m *customSysDeptModel) ListDepartmentMemberCounts(ctx context.Context, actor UserActor) (map[int64]int64, error) {
	scope, err := resolveUserScope(ctx, m.QueryRowsNoCacheCtx, actor)
	if err != nil {
		return nil, err
	}
	condition, args := scope.clause()
	var rows []struct {
		DeptId int64 `db:"dept_id"`
		Total  int64 `db:"total"`
	}
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT COALESCE(dept_id,0) AS dept_id,COUNT(*) AS total FROM sys_user WHERE deleted_at IS NULL AND "+condition+" GROUP BY COALESCE(dept_id,0)", args...); err != nil {
		return nil, err
	}
	counts := make(map[int64]int64, len(rows))
	for _, row := range rows {
		counts[row.DeptId] = row.Total
	}
	return counts, nil
}
