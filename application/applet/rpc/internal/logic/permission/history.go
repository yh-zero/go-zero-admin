package permissionlogic

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strconv"
	"time"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func historyError(message string) error { return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, message) }
func validKind(kind string) bool {
	return kind == "menu" || kind == "button" || kind == "api" || kind == "dataScope"
}

func decodeGrant(raw, kind string) (accessutil.PermissionGrantDocument, error) {
	var doc accessutil.PermissionGrantDocument
	if !validKind(kind) || len(raw) == 0 || len(raw) > 1<<20 || bytes.Equal(bytes.TrimSpace([]byte(raw)), []byte("null")) {
		return doc, historyError("授权历史明细缺失或无效")
	}
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&doc); err != nil {
		return doc, historyError("授权历史明细不完整，无法回滚")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return doc, historyError("授权历史包含额外明细")
	}
	if kind != "menu" && len(doc.MenuIDs) > 0 || kind != "button" && len(doc.MenuBtnIDs) > 0 || kind != "api" && len(doc.Policies) > 0 || kind != "dataScope" && doc.DataScope != nil {
		return doc, historyError("授权历史类型与明细不一致")
	}
	var err error
	doc.MenuIDs, err = accessutil.UniqueIDs(doc.MenuIDs)
	if err != nil {
		return doc, err
	}
	slices.Sort(doc.MenuIDs)
	if len(doc.MenuIDs) == 0 {
		doc.MenuIDs = nil
	}
	doc.MenuBtnIDs, err = accessutil.UniqueIDs(doc.MenuBtnIDs)
	if err != nil {
		return doc, err
	}
	slices.Sort(doc.MenuBtnIDs)
	if len(doc.MenuBtnIDs) == 0 {
		doc.MenuBtnIDs = nil
	}
	slices.SortFunc(doc.Policies, func(a, b accessutil.PermissionPolicy) int {
		if a.Path < b.Path {
			return -1
		}
		if a.Path > b.Path {
			return 1
		}
		if a.Method < b.Method {
			return -1
		}
		if a.Method > b.Method {
			return 1
		}
		return 0
	})
	doc.Policies = slices.Compact(doc.Policies)
	if len(doc.Policies) == 0 {
		doc.Policies = nil
	}
	if kind == "dataScope" {
		if doc.DataScope == nil {
			return doc, historyError("数据范围历史缺少明细")
		}
		switch doc.DataScope.Scope {
		case "all", "self", "department", "department_and_children", "custom":
		default:
			return doc, historyError("数据范围历史无效")
		}
		doc.DataScope.DepartmentIDs, err = accessutil.UniqueIDs(doc.DataScope.DepartmentIDs)
		if err != nil {
			return doc, err
		}
		slices.Sort(doc.DataScope.DepartmentIDs)
		if len(doc.DataScope.DepartmentIDs) == 0 {
			doc.DataScope.DepartmentIDs = nil
		}
		if (doc.DataScope.Scope == "custom") != (len(doc.DataScope.DepartmentIDs) > 0) {
			return doc, historyError("数据范围历史部门不完整")
		}
	}
	return doc, nil
}

func grantPB(doc accessutil.PermissionGrantDocument, roleID int64) *pb.PermissionGrantSet {
	out := &pb.PermissionGrantSet{MenuIds: append([]int64{}, doc.MenuIDs...), MenuBtnIds: append([]int64{}, doc.MenuBtnIDs...), Policies: []*pb.CasbinInfo{}}
	for _, p := range doc.Policies {
		out.Policies = append(out.Policies, &pb.CasbinInfo{Path: p.Path, Method: p.Method})
	}
	if doc.DataScope != nil {
		out.DataScope = &pb.RoleDataScope{AuthorityId: roleID, Scope: doc.DataScope.Scope, DepartmentIds: append([]int64{}, doc.DataScope.DepartmentIDs...)}
	}
	return out
}

func changePB(row model.SysPermissionChange) (*pb.PermissionChange, error) {
	before, err := decodeGrant(row.Before, row.Kind)
	if err != nil {
		return nil, err
	}
	after, err := decodeGrant(row.After, row.Kind)
	if err != nil {
		return nil, err
	}
	return &pb.PermissionChange{ID: row.ID, AuthorityId: row.AuthorityID, Kind: row.Kind, BeforeRevision: strconv.FormatUint(row.BeforeRevision, 10), AfterRevision: strconv.FormatUint(row.AfterRevision, 10), Before: grantPB(before, row.AuthorityID), After: grantPB(after, row.AuthorityID), ActorId: row.ActorID, ActorName: row.ActorName, TraceId: row.TraceID, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339Nano)}, nil
}

func rollbackVersion(row model.SysPermissionChange, current string) string {
	payload, _ := json.Marshal(struct {
		ID, Role            int64
		Kind, Before, After string
		BeforeRev, AfterRev uint64
		Current             string
	}{row.ID, row.AuthorityID, row.Kind, row.Before, row.After, row.BeforeRevision, row.AfterRevision, current})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func checkRollback(tx *gorm.DB, row model.SysPermissionChange) (accessutil.PermissionGrantDocument, accessutil.PermissionGrantDocument, error) {
	before, err := decodeGrant(row.Before, row.Kind)
	if err != nil {
		return before, accessutil.PermissionGrantDocument{}, err
	}
	after, err := decodeGrant(row.After, row.Kind)
	if err != nil {
		return before, after, err
	}
	if row.BeforeRevision == 0 || row.AfterRevision != row.BeforeRevision+1 {
		return before, after, historyError("授权历史版本不完整")
	}
	revision, err := accessutil.PermissionRevision(tx)
	if err != nil {
		return before, after, err
	}
	if revision != strconv.FormatUint(row.AfterRevision, 10) {
		return before, after, historyError("历史之后已有权限或资源修改，不能自动回滚")
	}
	if err = accessutil.RequireRole(tx, row.AuthorityID); err != nil {
		return before, after, err
	}
	current, err := accessutil.CapturePermissionState(tx)
	if err != nil {
		return before, after, err
	}
	a, _ := json.Marshal(accessutil.GrantDocument(current.Roles[row.AuthorityID], row.Kind))
	b, _ := json.Marshal(after)
	if !bytes.Equal(a, b) {
		return before, after, historyError("当前授权与历史结果不一致，不能自动回滚")
	}
	return before, after, nil
}

func loadChange(tx *gorm.DB, id int64) (model.SysPermissionChange, error) {
	var row model.SysPermissionChange
	if id <= 0 {
		return row, historyError("历史ID无效")
	}
	err := tx.First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = historyError("授权历史不存在")
	}
	return row, err
}

// Mutate only the recorded inverse delta. Validation finishes before any row is
// changed; the enclosing protected transaction owns version/history commits.
func restoreGrant(tx *gorm.DB, roleID int64, kind string, target, current accessutil.PermissionGrantDocument) error {
	switch kind {
	case "menu":
		var definitions []model.SysBaseMenu
		if err := tx.Find(&definitions).Error; err != nil {
			return err
		}
		byID := map[int64]model.SysBaseMenu{}
		for _, m := range definitions {
			byID[m.ID] = m
		}
		selected := idSet(target.MenuIDs)
		for _, id := range target.MenuIDs {
			seen := map[int64]bool{}
			for id != 0 {
				m, ok := byID[id]
				if !ok || seen[id] || !selected[id] {
					return historyError("回滚菜单不存在、层级循环或缺少父节点")
				}
				seen[id] = true
				id = m.ParentId
			}
		}
		added, removed := idDelta(target.MenuIDs, current.MenuIDs)
		if len(removed) > 0 {
			if err := tx.Where("sys_authority_authority_id = ? AND sys_base_menu_id IN ?", roleID, removed).Delete(&model.SysAuthorityMenu{}).Error; err != nil {
				return err
			}
		}
		for _, id := range added {
			if err := tx.Create(&model.SysAuthorityMenu{AuthorityId: strconv.FormatInt(roleID, 10), MenuId: strconv.FormatInt(id, 10)}).Error; err != nil {
				return err
			}
		}
		prune := tx.Where("authority_id = ?", roleID)
		if len(target.MenuIDs) > 0 {
			prune = prune.Where("sys_menu_id NOT IN ?", target.MenuIDs)
		}
		if err := prune.Delete(&model.SysAuthorityBtn{}).Error; err != nil {
			return err
		}
		var role model.SysAuthority
		if err := tx.First(&role, "authority_id = ?", roleID).Error; err != nil {
			return err
		}
		keepHome := role.DefaultRouter == ""
		for _, id := range target.MenuIDs {
			if byID[id].Name == role.DefaultRouter {
				keepHome = true
			}
		}
		if !keepHome {
			return tx.Model(&role).Update("default_router", "").Error
		}
	case "button":
		var definitions []model.SysBaseMenuBtn
		if len(target.MenuBtnIDs) > 0 {
			if err := tx.Where("id IN ?", target.MenuBtnIDs).Find(&definitions).Error; err != nil {
				return err
			}
		}
		if len(definitions) != len(target.MenuBtnIDs) {
			return historyError("回滚按钮已删除")
		}
		var menuIDs []int64
		if err := tx.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ?", roleID).Pluck("sys_base_menu_id", &menuIDs).Error; err != nil {
			return err
		}
		allowed := idSet(menuIDs)
		byID := map[int64]model.SysBaseMenuBtn{}
		for _, d := range definitions {
			if !allowed[d.SysBaseMenuID] {
				return historyError("回滚按钮所属菜单未授权")
			}
			var m model.SysBaseMenu
			if err := accessutil.RequireID(tx, &m, d.SysBaseMenuID); err != nil {
				return err
			}
			byID[d.ID] = d
		}
		added, removed := idDelta(target.MenuBtnIDs, current.MenuBtnIDs)
		if len(removed) > 0 {
			if err := tx.Where("authority_id = ? AND sys_base_menu_btn_id IN ?", roleID, removed).Delete(&model.SysAuthorityBtn{}).Error; err != nil {
				return err
			}
		}
		for _, id := range added {
			d := byID[id]
			if err := tx.Create(&model.SysAuthorityBtn{AuthorityId: roleID, SysMenuID: d.SysBaseMenuID, SysBaseMenuBtnID: id}).Error; err != nil {
				return err
			}
		}
	case "api":
		var definitions []model.SysApi
		if err := tx.Find(&definitions).Error; err != nil {
			return err
		}
		registered := map[accessutil.PermissionPolicy]bool{}
		for _, a := range definitions {
			registered[accessutil.PermissionPolicy{Path: a.Path, Method: a.Method}] = true
		}
		for _, p := range target.Policies {
			path, method, err := accessutil.API(p.Path, p.Method)
			if err != nil {
				return err
			}
			if path != p.Path || method != p.Method || !registered[p] {
				return historyError("回滚API资源不存在或无效")
			}
		}
		old, next := map[accessutil.PermissionPolicy]bool{}, map[accessutil.PermissionPolicy]bool{}
		for _, p := range current.Policies {
			old[p] = true
		}
		for _, p := range target.Policies {
			next[p] = true
		}
		role := strconv.FormatInt(roleID, 10)
		for _, p := range current.Policies {
			if !next[p] {
				if err := tx.Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", role, p.Path, p.Method).Delete(&gormadapter.CasbinRule{}).Error; err != nil {
					return err
				}
			}
		}
		for _, p := range target.Policies {
			if !old[p] {
				if err := tx.Create(&gormadapter.CasbinRule{Ptype: "p", V0: role, V1: p.Path, V2: p.Method}).Error; err != nil {
					return err
				}
			}
		}
	case "dataScope":
		if target.DataScope == nil || current.DataScope == nil {
			return historyError("数据范围明细缺失")
		}
		if roleID == accessutil.AdminAuthorityID && target.DataScope.Scope != "all" {
			return historyError("内置管理员的数据范围必须为all")
		}
		retained := idSet(current.DataScope.DepartmentIDs)
		if current.DataScope.Scope != "custom" {
			retained = map[int64]bool{}
		}
		for _, id := range target.DataScope.DepartmentIDs {
			original := id
			seen := map[int64]bool{}
			for id != 0 {
				if seen[id] {
					return historyError("回滚部门层级循环")
				}
				seen[id] = true
				var d model.SysDepartment
				if err := accessutil.RequireID(tx, &d, id); err != nil {
					return err
				}
				if !retained[original] && d.Status != 1 {
					return historyError("回滚新增部门或上级部门已停用")
				}
				id = d.ParentID
			}
		}
		if !reflect.DeepEqual(target.DataScope, current.DataScope) {
			row := model.SysRoleDataScope{AuthorityID: roleID, Scope: target.DataScope.Scope}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "authority_id"}}, DoUpdates: clause.AssignmentColumns([]string{"scope", "updated_at"})}).Create(&row).Error; err != nil {
				return err
			}
		}
		added, removed := idDelta(target.DataScope.DepartmentIDs, current.DataScope.DepartmentIDs)
		if len(removed) > 0 {
			if err := tx.Where("authority_id = ? AND department_id IN ?", roleID, removed).Delete(&model.SysRoleScopeDepartment{}).Error; err != nil {
				return err
			}
		}
		for _, id := range added {
			if err := tx.Create(&model.SysRoleScopeDepartment{AuthorityID: roleID, DepartmentID: id}).Error; err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported permission history kind")
	}
	return nil
}
func idSet(ids []int64) map[int64]bool {
	out := map[int64]bool{}
	for _, id := range ids {
		out[id] = true
	}
	return out
}
func idDelta(target, current []int64) (added, removed []int64) {
	old, next := idSet(current), idSet(target)
	for _, id := range target {
		if !old[id] {
			added = append(added, id)
		}
	}
	for _, id := range current {
		if !next[id] {
			removed = append(removed, id)
		}
	}
	return
}
