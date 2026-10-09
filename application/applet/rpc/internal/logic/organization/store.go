package organizationlogic

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type store struct {
	ctx context.Context
	db  *gorm.DB
}

func orgError(message string) error { return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, message) }

func (s store) record(tx *gorm.DB, action string, id int64) error {
	return audit.Record(s.ctx, tx, audit.Event{Module: "organization", Action: action, Object: strconv.FormatInt(id, 10)})
}
func (s store) change(fn func(*gorm.DB) error) error {
	return accessutil.FriendlyDuplicate(accessutil.PermissionTransaction(s.db.WithContext(s.ctx), func(tx *gorm.DB) error {
		return fn(tx)
	}))
}
func validText(name, code string) error {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(code) == "" || utf8.RuneCountInString(name) > 64 || utf8.RuneCountInString(code) > 64 {
		return orgError("名称和编码不能为空，且不能超过64字符")
	}
	return nil
}
func departmentPB(d model.SysDepartment) *pb.Department {
	return &pb.Department{ID: d.ID, ParentId: d.ParentID, Name: d.Name, Code: d.Code, Sort: d.Sort, Status: d.Status, Leader: d.Leader, Children: []*pb.Department{}}
}
func positionPB(p model.SysPosition) *pb.Position {
	return &pb.Position{ID: p.ID, Name: p.Name, Code: p.Code, Sort: p.Sort, Status: p.Status}
}
func listFilter(in *pb.OrganizationListRequest) error {
	if in == nil {
		return orgError("请求不能为空")
	}
	if in.Status != 0 {
		return accessutil.Status(in.Status)
	}
	return nil
}
func (s store) departments(in *pb.OrganizationListRequest) (*pb.DepartmentListResponse, error) {
	if err := listFilter(in); err != nil {
		return nil, err
	}
	var rows []model.SysDepartment
	if err := s.db.WithContext(s.ctx).Order("sort ASC,id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	byID := map[int64]model.SysDepartment{}
	parents := map[int64]int64{}
	keep := map[int64]bool{}
	for _, r := range rows {
		byID[r.ID] = r
		parents[r.ID] = r.ParentID
	}
	keyword := strings.ToLower(strings.TrimSpace(in.Keyword))
	for _, r := range rows {
		if err := accessutil.ValidateParent(r.ID, r.ParentID, parents); err != nil {
			return nil, err
		}
		if (in.Status == 0 || r.Status == in.Status) && (keyword == "" || strings.Contains(strings.ToLower(r.Name+" "+r.Code), keyword)) {
			for id := r.ID; id != 0; id = byID[id].ParentID {
				keep[id] = true
			}
		}
	}
	nodes := map[int64]*pb.Department{}
	out := &pb.DepartmentListResponse{List: []*pb.Department{}}
	for _, r := range rows {
		if keep[r.ID] {
			nodes[r.ID] = departmentPB(r)
		}
	}
	for _, r := range rows {
		n := nodes[r.ID]
		if n == nil {
			continue
		}
		if r.ParentID == 0 {
			out.List = append(out.List, n)
		} else {
			nodes[r.ParentID].Children = append(nodes[r.ParentID].Children, n)
		}
	}
	return out, nil
}
func validateDepartment(tx *gorm.DB, d *pb.Department) error {
	if d == nil {
		return orgError("部门不能为空")
	}
	d.Name = strings.TrimSpace(d.Name)
	d.Code = strings.TrimSpace(d.Code)
	d.Leader = strings.TrimSpace(d.Leader)
	if err := validText(d.Name, d.Code); err != nil {
		return err
	}
	if utf8.RuneCountInString(d.Leader) > 64 {
		return orgError("负责人不能超过64字符")
	}
	if err := accessutil.Status(d.Status); err != nil {
		return err
	}
	if err := accessutil.Unique(tx, &model.SysDepartment{}, "id <> ? AND (LOWER(code) = ? OR (parent_id = ? AND LOWER(name) = ?))", d.ID, strings.ToLower(d.Code), d.ParentId, strings.ToLower(d.Name)); err != nil {
		return err
	}
	var rows []model.SysDepartment
	if err := tx.Find(&rows).Error; err != nil {
		return err
	}
	parents := map[int64]int64{}
	byID := map[int64]model.SysDepartment{}
	for _, r := range rows {
		parents[r.ID] = r.ParentID
		byID[r.ID] = r
	}
	if err := accessutil.ValidateParent(d.ID, d.ParentId, parents); err != nil {
		return err
	}
	if d.Status == 1 {
		for id := d.ParentId; id != 0; id = byID[id].ParentID {
			if byID[id].Status != 1 {
				return orgError("启用部门的上级部门必须启用")
			}
		}
	}
	if d.Status == 2 && d.ID > 0 {
		for _, r := range rows {
			if r.ParentID == d.ID && r.Status == 1 {
				return orgError("请先停用启用中的子部门")
			}
		}
	}
	return nil
}
func (s store) saveDepartment(in *pb.DepartmentRequest, create bool) (*pb.Department, error) {
	if in == nil || in.Department == nil {
		return nil, orgError("部门不能为空")
	}
	d := in.Department
	var row model.SysDepartment
	if create {
		if d.ID != 0 {
			return nil, orgError("创建部门不允许指定ID")
		}
	} else if d.ID <= 0 {
		return nil, orgError("部门ID无效")
	}
	err := s.change(func(tx *gorm.DB) error {
		if !create {
			if err := accessutil.RequireID(tx, &row, d.ID); err != nil {
				return err
			}
		}
		if err := validateDepartment(tx, d); err != nil {
			return err
		}
		if create {
			row = model.SysDepartment{ParentID: d.ParentId, Name: d.Name, Code: d.Code, Sort: d.Sort, Status: d.Status, Leader: d.Leader}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			return s.record(tx, "createDepartment", row.ID)
		}
		if err := tx.Model(&row).Updates(map[string]any{"parent_id": d.ParentId, "name": d.Name, "code": d.Code, "sort": d.Sort, "status": d.Status, "leader": d.Leader}).Error; err != nil {
			return err
		}
		return s.record(tx, "updateDepartment", row.ID)
	})
	if err != nil {
		return nil, err
	}
	return departmentPB(row), nil
}
func hasReference(tx *gorm.DB, entity any, condition string, args ...any) error {
	var count int64
	if err := tx.Model(entity).Where(condition, args...).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return orgError("记录仍被引用，请先解除关联")
	}
	return nil
}
func (s store) deleteDepartment(in *pb.OrganizationIDRequest) error {
	if in == nil {
		return orgError("部门ID无效")
	}
	return s.change(func(tx *gorm.DB) error {
		var row model.SysDepartment
		if err := accessutil.RequireID(tx, &row, in.ID); err != nil {
			return err
		}
		for _, r := range []struct {
			entity any
			where  string
		}{{&model.SysDepartment{}, "parent_id = ?"}, {&model.SysUserDepartment{}, "department_id = ?"}, {&model.SysRoleScopeDepartment{}, "department_id = ?"}} {
			if err := hasReference(tx, r.entity, r.where, in.ID); err != nil {
				return err
			}
		}
		if tx.Migrator().HasTable("sys_file_resources") {
			var count int64
			if err := tx.Table("sys_file_resources").Where("department_id = ? AND status <> ?", in.ID, "deleted").Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return orgError("部门仍被文件资源引用")
			}
		}
		if err := tx.Unscoped().Delete(&row).Error; err != nil {
			return err
		}
		return s.record(tx, "deleteDepartment", row.ID)
	})
}
func (s store) positions(in *pb.OrganizationListRequest) (*pb.PositionListResponse, error) {
	if err := listFilter(in); err != nil {
		return nil, err
	}
	db := s.db.WithContext(s.ctx)
	if in.Status != 0 {
		db = db.Where("status = ?", in.Status)
	}
	if k := strings.TrimSpace(in.Keyword); k != "" {
		db = db.Where("name LIKE ? OR code LIKE ?", "%"+k+"%", "%"+k+"%")
	}
	var rows []model.SysPosition
	if err := db.Order("sort ASC,id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := &pb.PositionListResponse{List: []*pb.Position{}}
	for _, r := range rows {
		out.List = append(out.List, positionPB(r))
	}
	return out, nil
}
func (s store) savePosition(in *pb.PositionRequest, create bool) (*pb.Position, error) {
	if in == nil || in.Position == nil {
		return nil, orgError("岗位不能为空")
	}
	p := in.Position
	p.Name = strings.TrimSpace(p.Name)
	p.Code = strings.TrimSpace(p.Code)
	if err := validText(p.Name, p.Code); err != nil {
		return nil, err
	}
	if err := accessutil.Status(p.Status); err != nil {
		return nil, err
	}
	if (create && p.ID != 0) || (!create && p.ID <= 0) {
		return nil, orgError("岗位ID无效")
	}
	var row model.SysPosition
	err := s.change(func(tx *gorm.DB) error {
		if !create {
			if err := accessutil.RequireID(tx, &row, p.ID); err != nil {
				return err
			}
		}
		if err := accessutil.Unique(tx, &model.SysPosition{}, "id <> ? AND (LOWER(code) = ? OR LOWER(name) = ?)", p.ID, strings.ToLower(p.Code), strings.ToLower(p.Name)); err != nil {
			return err
		}
		if create {
			row = model.SysPosition{Name: p.Name, Code: p.Code, Sort: p.Sort, Status: p.Status}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			return s.record(tx, "createPosition", row.ID)
		}
		if err := tx.Model(&row).Updates(map[string]any{"name": p.Name, "code": p.Code, "sort": p.Sort, "status": p.Status}).Error; err != nil {
			return err
		}
		return s.record(tx, "updatePosition", row.ID)
	})
	if err != nil {
		return nil, err
	}
	return positionPB(row), nil
}
func (s store) deletePosition(in *pb.OrganizationIDRequest) error {
	if in == nil {
		return orgError("岗位ID无效")
	}
	return s.change(func(tx *gorm.DB) error {
		var row model.SysPosition
		if err := accessutil.RequireID(tx, &row, in.ID); err != nil {
			return err
		}
		if err := hasReference(tx, &model.SysUserPosition{}, "position_id = ?", in.ID); err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&row).Error; err != nil {
			return err
		}
		return s.record(tx, "deletePosition", row.ID)
	})
}
func getMembership(tx *gorm.DB, userID int64) (*pb.MembershipResponse, error) {
	var user model.SysUser
	if err := accessutil.RequireID(tx, &user, userID); err != nil {
		return nil, err
	}
	out := &pb.MembershipResponse{UserID: userID, PositionIds: []int64{}}
	var relation model.SysUserDepartment
	err := tx.First(&relation, "user_id = ?", userID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		out.DepartmentId = relation.DepartmentID
	}
	if err := tx.Model(&model.SysUserPosition{}).Where("user_id = ?", userID).Order("position_id ASC").Pluck("position_id", &out.PositionIds).Error; err != nil {
		return nil, err
	}
	return out, nil
}
func (s store) membership(in *pb.OrganizationIDRequest) (*pb.MembershipResponse, error) {
	if in == nil {
		return nil, orgError("用户ID无效")
	}
	var out *pb.MembershipResponse
	err := s.db.WithContext(s.ctx).Transaction(func(tx *gorm.DB) error { var err error; out, err = getMembership(tx, in.ID); return err })
	return out, err
}

// Retained assignments may be disabled, but their complete ancestry must still
// exist. New assignments require the department and every ancestor to be enabled.
func validateDepartmentAssignment(tx *gorm.DB, id int64, retained bool) error {
	if id <= 0 {
		return orgError("部门ID无效")
	}
	seen := map[int64]bool{}
	for id != 0 {
		if seen[id] {
			return orgError("部门层级循环")
		}
		seen[id] = true
		var row model.SysDepartment
		if err := accessutil.RequireID(tx, &row, id); err != nil {
			return err
		}
		if !retained && row.Status != 1 {
			return orgError("部门或上级部门已停用")
		}
		id = row.ParentID
	}
	return nil
}
func (s store) updateMembership(in *pb.MembershipRequest) error {
	if in == nil || in.UserID <= 0 || in.DepartmentId < 0 {
		return orgError("用户或部门ID无效")
	}
	ids, err := accessutil.UniqueIDs(in.PositionIds)
	if err != nil {
		return err
	}
	slices.Sort(ids)
	return s.change(func(tx *gorm.DB) error {
		var user model.SysUser
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, "id = ?", in.UserID).Error; err != nil {
			return err
		}
		old, err := getMembership(tx, in.UserID)
		if err != nil {
			return err
		}
		if in.DepartmentId != 0 {
			if err := validateDepartmentAssignment(tx, in.DepartmentId, old.DepartmentId == in.DepartmentId); err != nil {
				return err
			}
		}
		if len(ids) > 0 {
			var count int64
			if err := tx.Model(&model.SysPosition{}).Where("id IN ?", ids).
				Where("(status = 1 OR id IN ?)", old.PositionIds).Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(ids)) {
				return orgError("岗位不存在或新分配岗位已停用")
			}
		}
		if old.DepartmentId == in.DepartmentId && slices.Equal(old.PositionIds, ids) {
			return nil
		}
		if err := tx.Where("user_id = ?", in.UserID).Delete(&model.SysUserDepartment{}).Error; err != nil {
			return err
		}
		if in.DepartmentId != 0 {
			if err := tx.Create(&model.SysUserDepartment{UserID: in.UserID, DepartmentID: in.DepartmentId}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("user_id = ?", in.UserID).Delete(&model.SysUserPosition{}).Error; err != nil {
			return err
		}
		for _, id := range ids {
			if err := tx.Create(&model.SysUserPosition{UserID: in.UserID, PositionID: id}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&user).Update("session_version", gorm.Expr("session_version + 1")).Error; err != nil {
			return err
		}
		return s.record(tx, "updateMembership", user.ID)
	})
}
func (s store) dataScope(in *pb.GetRoleDataScopeRequest) (*pb.RoleDataScopeResponse, error) {
	if in == nil {
		return nil, orgError("角色ID无效")
	}
	var out *pb.RoleDataScopeResponse
	err := accessutil.PermissionRead(s.db.WithContext(s.ctx), func(tx *gorm.DB) error {
		e, err := accessutil.LoadPermissionEdit(tx, in.AuthorityId, "dataScope")
		if err != nil {
			return err
		}
		out = &pb.RoleDataScopeResponse{DataScope: e.DataScope}
		return nil
	})
	return out, err
}
func (s store) updateDataScope(in *pb.RoleDataScopeRequest) error {
	if in == nil || in.DataScope == nil {
		return orgError("数据范围不能为空")
	}
	value := in.DataScope
	switch value.Scope {
	case "all", "self", "department", "department_and_children", "custom":
	default:
		return orgError("数据范围无效")
	}
	if value.AuthorityId == accessutil.AdminAuthorityID && value.Scope != "all" {
		return orgError("内置管理员的数据范围必须为all")
	}
	ids, err := accessutil.UniqueIDs(value.DepartmentIds)
	if err != nil {
		return err
	}
	if value.Scope != "custom" && len(ids) > 0 {
		return orgError("仅custom范围允许指定部门")
	}
	if value.Scope == "custom" && len(ids) == 0 {
		return orgError("custom范围至少选择一个部门")
	}
	return s.change(func(tx *gorm.DB) error {
		if err := accessutil.RequirePermissionRevision(tx, value.ExpectedRevision); err != nil {
			return err
		}
		if err := accessutil.RequireRole(tx, value.AuthorityId); err != nil {
			return err
		}
		var old model.SysRoleDataScope
		err := tx.First(&old, "authority_id = ?", value.AuthorityId).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		retained := map[int64]bool{}
		// A stale relation from a different scope is not an existing custom grant.
		if err == nil && old.Scope == "custom" {
			var oldIDs []int64
			if err := tx.Model(&model.SysRoleScopeDepartment{}).Where("authority_id = ?", value.AuthorityId).Pluck("department_id", &oldIDs).Error; err != nil {
				return err
			}
			for _, id := range oldIDs {
				retained[id] = true
			}
		}
		for _, id := range ids {
			if err := validateDepartmentAssignment(tx, id, retained[id]); err != nil {
				return err
			}
		}
		row := model.SysRoleDataScope{AuthorityID: value.AuthorityId, Scope: value.Scope}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "authority_id"}}, DoUpdates: clause.AssignmentColumns([]string{"scope", "updated_at"})}).Create(&row).Error; err != nil {
			return err
		}
		if err := tx.Where("authority_id = ?", value.AuthorityId).Delete(&model.SysRoleScopeDepartment{}).Error; err != nil {
			return err
		}
		for _, id := range ids {
			if err := tx.Create(&model.SysRoleScopeDepartment{AuthorityID: value.AuthorityId, DepartmentID: id}).Error; err != nil {
				return err
			}
		}
		return s.record(tx, "updateRoleDataScope", value.AuthorityId)
	})
}
