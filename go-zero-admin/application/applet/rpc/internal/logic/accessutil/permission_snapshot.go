package accessutil

import (
	"errors"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"strconv"
)

// LoadPermissionEdit must only be called inside PermissionRead/PermissionTransaction.
func LoadPermissionEdit(tx *gorm.DB, authorityID int64, kind string) (*pb.GetPermissionEditResponse, error) {
	switch kind {
	case "menu", "button", "api", "dataScope":
	default:
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "授权类型无效")
	}
	if err := RequireRole(tx, authorityID); err != nil {
		return nil, err
	}
	rev, err := PermissionRevision(tx)
	if err != nil {
		return nil, err
	}
	out := &pb.GetPermissionEditResponse{Revision: rev, AuthorityId: authorityID, Kind: kind, Menus: []*pb.SysBaseMenu{}, Apis: []*pb.SysApi{}, Departments: []*pb.Department{}, MenuIds: []int64{}, MenuBtnIds: []int64{}, Policies: []*pb.CasbinInfo{}, DataScope: &pb.RoleDataScope{AuthorityId: authorityID, Scope: "self", DepartmentIds: []int64{}, Revision: rev}}
	var menus []model.SysBaseMenu
	if err := tx.Order("sort,id").Preload("MenuBtn").Preload("Parameters").Find(&menus).Error; err != nil {
		return nil, err
	}
	nodes := map[int64]*pb.SysBaseMenu{}
	for _, m := range menus {
		n := &pb.SysBaseMenu{}
		if err := copier.Copy(n, m); err != nil {
			return nil, err
		}
		n.Children = []*pb.SysBaseMenu{}
		nodes[m.ID] = n
	}
	for _, m := range menus {
		n := nodes[m.ID]
		if m.ParentId == 0 {
			out.Menus = append(out.Menus, n)
		} else if p := nodes[m.ParentId]; p != nil {
			p.Children = append(p.Children, n)
		}
	}
	var apis []model.SysApi
	if err := tx.Order("api_group,path,method,id").Find(&apis).Error; err != nil {
		return nil, err
	}
	if err := copier.Copy(&out.Apis, apis); err != nil {
		return nil, err
	}
	var departments []model.SysDepartment
	if err := tx.Order("sort,id").Find(&departments).Error; err != nil {
		return nil, err
	}
	deps := map[int64]*pb.Department{}
	for _, d := range departments {
		deps[d.ID] = &pb.Department{ID: d.ID, ParentId: d.ParentID, Name: d.Name, Code: d.Code, Sort: d.Sort, Status: d.Status, Leader: d.Leader, Children: []*pb.Department{}}
	}
	for _, d := range departments {
		n := deps[d.ID]
		if d.ParentID == 0 {
			out.Departments = append(out.Departments, n)
		} else if p := deps[d.ParentID]; p != nil {
			p.Children = append(p.Children, n)
		}
	}
	if err := tx.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ?", authorityID).Order("sys_base_menu_id").Pluck("sys_base_menu_id", &out.MenuIds).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&model.SysAuthorityBtn{}).Where("authority_id = ?", authorityID).Order("sys_base_menu_btn_id").Pluck("sys_base_menu_btn_id", &out.MenuBtnIds).Error; err != nil {
		return nil, err
	}
	var rules []gormadapter.CasbinRule
	if err := tx.Where("ptype = ? AND v0 = ?", "p", strconv.FormatInt(authorityID, 10)).Order("v1,v2").Find(&rules).Error; err != nil {
		return nil, err
	}
	for _, r := range rules {
		out.Policies = append(out.Policies, &pb.CasbinInfo{Path: r.V1, Method: r.V2})
	}
	var scope model.SysRoleDataScope
	err = tx.First(&scope, "authority_id = ?", authorityID).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		out.DataScope.Scope = scope.Scope
	}
	if authorityID == AdminAuthorityID {
		out.DataScope.Scope = "all"
	}
	if err := tx.Model(&model.SysRoleScopeDepartment{}).Where("authority_id = ?", authorityID).Order("department_id").Pluck("department_id", &out.DataScope.DepartmentIds).Error; err != nil {
		return nil, err
	}
	return out, nil
}
