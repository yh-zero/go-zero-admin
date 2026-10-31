package model

import (
	"context"
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
		CreateDepartment(ctx context.Context, dept *DeptRow) error
		UpdateDepartment(ctx context.Context, id int64, patch DeptPatch) error
		DeleteDepartment(ctx context.Context, id int64) error
		ListMyDepartments(ctx context.Context, userId int64, all bool) ([]*DeptRow, error)
	}

	customSysDeptModel struct {
		*defaultSysDeptModel
	}
)

// DeptRow 部门行（时间列 UNIX_TIMESTAMP 转 int64）
type DeptRow struct {
	Id        int64  `db:"id"`
	ParentId  int64  `db:"parent_id"`
	Name      string `db:"name"`
	Leader    string `db:"leader"`
	Phone     string `db:"phone"`
	Email     string `db:"email"`
	Sort      int32  `db:"sort"`
	Status    int32  `db:"status"`
	Path      string `db:"path"`
	CreatedAt int64  `db:"created_at"`
}

type DeptPatch struct {
	ParentId                   *int64
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

func (m *customSysDeptModel) CreateDepartment(ctx context.Context, dept *DeptRow) error {
	if dept.ParentId < 0 || strings.TrimSpace(dept.Name) == "" || (dept.Status != 1 && dept.Status != 2) {
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
		result, err := s.ExecCtx(ctx, "INSERT INTO sys_dept (parent_id,name,leader,phone,email,sort,status,path) VALUES (?,?,?,?,?,?,?,?)", dept.ParentId, dept.Name, dept.Leader, dept.Phone, dept.Email, dept.Sort, dept.Status, "")
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
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysDeptIdPrefix, dept.Id))
}

func (m *customSysDeptModel) UpdateDepartment(ctx context.Context, id int64, p DeptPatch) error {
	if id <= 0 || p.ParentId != nil && *p.ParentId < 0 || p.Name != nil && strings.TrimSpace(*p.Name) == "" || p.Status != nil && *p.Status != 1 && *p.Status != 2 {
		return ErrInvalidMutation
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
		if p.ParentId != nil {
			if *p.ParentId > 0 {
				parent := nodes[*p.ParentId]
				if parent == nil || parent.Status != 1 {
					return fmt.Errorf("%w: 父部门不存在或已停用", ErrInvalidMutation)
				}
			}
			node.ParentId = *p.ParentId
		}
		if p.Name != nil {
			node.Name = *p.Name
		}
		if p.Leader != nil {
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
		if _, err = s.ExecCtx(ctx, "UPDATE sys_dept SET parent_id=?,name=?,leader=?,phone=?,email=?,sort=?,status=?,path=? WHERE id=? AND deleted_at IS NULL", node.ParentId, node.Name, node.Leader, node.Phone, node.Email, node.Sort, node.Status, node.Path, id); err != nil {
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
		keys = append(keys, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysDeptIdPrefix, affectedId))
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
		found := false
		for _, r := range rows {
			if r.Id == id {
				found = true
			}
			if r.ParentId == id {
				return fmt.Errorf("%w: 存在子部门，无法删除", ErrInvalidMutation)
			}
		}
		if !found {
			return sqlx.ErrNotFound
		}
		var count struct {
			Total int64 `db:"total"`
		}
		if err = s.QueryRowCtx(ctx, &count, "SELECT COUNT(*) AS total FROM sys_user WHERE dept_id=? AND deleted_at IS NULL", id); err != nil {
			return err
		}
		if count.Total > 0 {
			return fmt.Errorf("%w: 部门下存在用户，无法删除", ErrInvalidMutation)
		}
		if err = s.QueryRowCtx(ctx, &count, "SELECT COUNT(*) AS total FROM casbin_rule WHERE ptype='g' AND v2=?", fmt.Sprintf("dept:%d", id)); err != nil {
			return err
		}
		if count.Total > 0 {
			return fmt.Errorf("%w: 部门仍有成员授权，无法删除", ErrInvalidMutation)
		}
		_, err = s.ExecCtx(ctx, "UPDATE sys_dept SET deleted_at=NOW() WHERE id=? AND deleted_at IS NULL", id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysDeptIdPrefix, id))
}

func (m *customSysDeptModel) ListMyDepartments(ctx context.Context, userId int64, all bool) ([]*DeptRow, error) {
	rows := make([]*DeptRow, 0)
	query := "SELECT d.id,d.parent_id,d.name,d.leader,d.phone,d.email,d.sort,d.status,d.path,UNIX_TIMESTAMP(d.created_at) AS created_at FROM sys_dept d WHERE d.status=1 AND d.deleted_at IS NULL"
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
	err := s.QueryRowsCtx(ctx, &rows, "SELECT id,parent_id,name,leader,phone,email,sort,status,path,UNIX_TIMESTAMP(created_at) AS created_at FROM sys_dept WHERE deleted_at IS NULL ORDER BY id FOR UPDATE")
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
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminNewSysDeptIdPrefix, id))
}

func (m *customSysDeptModel) ListDeptTree(ctx context.Context, status int64) ([]*DeptRow, error) {
	where := "deleted_at IS NULL"
	var args []any
	if status > 0 {
		where += " AND status=?"
		args = append(args, status)
	}
	var rows []*DeptRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id, parent_id, name, leader, phone, email, sort, status, path, UNIX_TIMESTAMP(created_at) AS created_at "+
			"FROM sys_dept WHERE "+where+" ORDER BY sort, id", args...)
	return rows, err
}
