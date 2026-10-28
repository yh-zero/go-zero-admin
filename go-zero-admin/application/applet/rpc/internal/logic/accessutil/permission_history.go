package accessutil

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/audit"
	"gorm.io/gorm"
)

type PermissionPolicy struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}
type PermissionDataScope struct {
	Scope         string  `json:"scope"`
	DepartmentIDs []int64 `json:"departmentIds"`
}
type PermissionGrantDocument struct {
	MenuIDs    []int64              `json:"menuIds,omitempty"`
	MenuBtnIDs []int64              `json:"menuBtnIds,omitempty"`
	Policies   []PermissionPolicy   `json:"policies,omitempty"`
	DataScope  *PermissionDataScope `json:"dataScope,omitempty"`
}
type RolePermissionState struct {
	MenuIDs    []int64
	MenuBtnIDs []int64
	Policies   []PermissionPolicy
	DataScope  PermissionDataScope
}
type PermissionState struct {
	Revision uint64
	Roles    map[int64]RolePermissionState
}

func canonicalIDs(ids []int64) []int64 { slices.Sort(ids); return slices.Compact(ids) }

// Call only in the transaction that holds the common administrator/version
// locks. Capturing grants separately from revision would produce stale edits.
func CapturePermissionState(tx *gorm.DB) (PermissionState, error) {
	rev, err := PermissionRevision(tx)
	if err != nil {
		return PermissionState{}, err
	}
	revision, err := strconv.ParseUint(rev, 10, 64)
	if err != nil {
		return PermissionState{}, err
	}
	state := PermissionState{Revision: revision, Roles: map[int64]RolePermissionState{}}
	var roles []model.SysAuthority
	if err = tx.Where("deleted_at IS NULL").Find(&roles).Error; err != nil {
		return state, err
	}
	for _, r := range roles {
		scope := "self"
		if r.AuthorityId == AdminAuthorityID {
			scope = "all"
		}
		state.Roles[r.AuthorityId] = RolePermissionState{DataScope: PermissionDataScope{Scope: scope}}
	}
	var menus []model.SysAuthorityMenu
	if err = tx.Find(&menus).Error; err != nil {
		return state, err
	}
	for _, row := range menus {
		id, err := strconv.ParseInt(row.AuthorityId, 10, 64)
		if err != nil {
			return state, fmt.Errorf("invalid stored role grant: %w", err)
		}
		role, ok := state.Roles[id]
		if !ok {
			continue
		}
		menu, err := strconv.ParseInt(row.MenuId, 10, 64)
		if err != nil || menu <= 0 {
			return state, fmt.Errorf("invalid stored menu grant")
		}
		role.MenuIDs = append(role.MenuIDs, menu)
		state.Roles[id] = role
	}
	var buttons []model.SysAuthorityBtn
	if err = tx.Find(&buttons).Error; err != nil {
		return state, err
	}
	for _, row := range buttons {
		if role, ok := state.Roles[row.AuthorityId]; ok {
			role.MenuBtnIDs = append(role.MenuBtnIDs, row.SysBaseMenuBtnID)
			state.Roles[row.AuthorityId] = role
		}
	}
	var policies []gormadapter.CasbinRule
	if err = tx.Where("ptype = ?", "p").Find(&policies).Error; err != nil {
		return state, err
	}
	for _, row := range policies {
		id, err := strconv.ParseInt(row.V0, 10, 64)
		if err != nil {
			continue
		}
		if role, ok := state.Roles[id]; ok {
			role.Policies = append(role.Policies, PermissionPolicy{Path: row.V1, Method: row.V2})
			state.Roles[id] = role
		}
	}
	var scopes []model.SysRoleDataScope
	if err = tx.Find(&scopes).Error; err != nil {
		return state, err
	}
	for _, row := range scopes {
		if role, ok := state.Roles[row.AuthorityID]; ok && row.AuthorityID != AdminAuthorityID {
			role.DataScope.Scope = row.Scope
			state.Roles[row.AuthorityID] = role
		}
	}
	var departments []model.SysRoleScopeDepartment
	if err = tx.Find(&departments).Error; err != nil {
		return state, err
	}
	for _, row := range departments {
		if role, ok := state.Roles[row.AuthorityID]; ok {
			role.DataScope.DepartmentIDs = append(role.DataScope.DepartmentIDs, row.DepartmentID)
			state.Roles[row.AuthorityID] = role
		}
	}
	for id, role := range state.Roles {
		role.MenuIDs = canonicalIDs(role.MenuIDs)
		role.MenuBtnIDs = canonicalIDs(role.MenuBtnIDs)
		role.DataScope.DepartmentIDs = canonicalIDs(role.DataScope.DepartmentIDs)
		slices.SortFunc(role.Policies, func(a, b PermissionPolicy) int {
			if a.Path != b.Path {
				if a.Path < b.Path {
					return -1
				}
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
		role.Policies = slices.Compact(role.Policies)
		state.Roles[id] = role
	}
	return state, nil
}

func GrantDocument(role RolePermissionState, kind string) PermissionGrantDocument {
	switch kind {
	case "menu":
		return PermissionGrantDocument{MenuIDs: role.MenuIDs}
	case "button":
		return PermissionGrantDocument{MenuBtnIDs: role.MenuBtnIDs}
	case "api":
		return PermissionGrantDocument{Policies: role.Policies}
	case "dataScope":
		return PermissionGrantDocument{DataScope: &role.DataScope}
	}
	return PermissionGrantDocument{}
}

func historyText(s string) string {
	r := []rune(s)
	if len(r) > 64 {
		return string(r[:64])
	}
	return s
}

// Revision and complete controlled before/after documents commit with the
// grants. An audit insert failure aborts the enclosing business transaction.
func CommitPermissionChanges(ctx context.Context, tx *gorm.DB, before PermissionState) error {
	if before.Revision == 0 || before.Revision == math.MaxUint64 {
		return fmt.Errorf("permission revision exhausted or invalid")
	}
	after, err := CapturePermissionState(tx)
	if err != nil {
		return err
	}
	if after.Revision != before.Revision {
		return fmt.Errorf("permission revision changed inside transaction")
	}
	ids := make([]int64, 0, len(before.Roles)+len(after.Roles))
	for id := range before.Roles {
		ids = append(ids, id)
	}
	for id := range after.Roles {
		ids = append(ids, id)
	}
	ids = canonicalIDs(ids)
	actor, request := audit.ActorFromContext(ctx), audit.RequestFromContext(ctx)
	rows := make([]model.SysPermissionChange, 0)
	for _, id := range ids {

		oldRole, oldExists := before.Roles[id]
		newRole, newExists := after.Roles[id]
		// A missing role has no grants and the ordinary default data scope.
		// Keep deletion/creation documents valid rather than recording an
		// empty scope that cannot be decoded by history consumers.
		defaultScope := "self"
		if id == AdminAuthorityID {
			defaultScope = "all"
		}
		if !oldExists {
			oldRole.DataScope.Scope = defaultScope
		}
		if !newExists {
			newRole.DataScope.Scope = defaultScope
		}
		for _, kind := range []string{"menu", "button", "api", "dataScope"} {
			oldJSON, err := json.Marshal(GrantDocument(oldRole, kind))
			if err != nil {
				return err
			}
			newJSON, err := json.Marshal(GrantDocument(newRole, kind))
			if err != nil {
				return err
			}
			if string(oldJSON) == string(newJSON) {
				continue
			}
			// Never store a truncated document that could later appear rollbackable.
			if len(oldJSON) > 1<<20 || len(newJSON) > 1<<20 {
				return fmt.Errorf("permission change exceeds complete history document limit")
			}
			rows = append(rows, model.SysPermissionChange{AuthorityID: id, Kind: kind, BeforeRevision: before.Revision, AfterRevision: before.Revision + 1, Before: string(oldJSON), After: string(newJSON), ActorID: actor.ID, ActorName: historyText(actor.Name), TraceID: historyText(request.TraceID), CreatedAt: time.Now().UTC()})
		}
	}
	updated := tx.Model(&model.SysPermissionVersion{}).Where("id = ? AND revision = ?", 1, before.Revision).UpdateColumn("revision", before.Revision+1)
	if updated.Error != nil {
		return updated.Error
	}
	if updated.RowsAffected != 1 {
		return fmt.Errorf("permission revision update conflict")
	}
	if len(rows) > 0 {
		return tx.CreateInBatches(&rows, 128).Error
	}
	return nil
}
