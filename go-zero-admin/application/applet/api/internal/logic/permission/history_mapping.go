package permission

import (
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
)

func historyGrant(row *pb.PermissionGrantSet) types.PermissionGrantSet {
	out := types.PermissionGrantSet{MenuIds: []int64{}, MenuBtnIds: []int64{}, Policies: []types.CasbinInfo{}}
	if row == nil {
		return out
	}
	out.MenuIds = append(out.MenuIds, row.MenuIds...)
	out.MenuBtnIds = append(out.MenuBtnIds, row.MenuBtnIds...)
	for _, p := range row.Policies {
		if p != nil {
			out.Policies = append(out.Policies, types.CasbinInfo{Path: p.Path, Method: p.Method})
		}
	}
	if row.DataScope != nil {
		d := row.DataScope
		out.DataScope = types.RoleDataScope{AuthorityId: d.AuthorityId, Scope: d.Scope, DepartmentIds: append([]int64{}, d.DepartmentIds...), Revision: d.Revision}
	}
	return out
}
func historyChange(row *pb.PermissionChange) types.PermissionChange {
	if row == nil {
		return types.PermissionChange{}
	}
	return types.PermissionChange{ID: row.ID, AuthorityId: row.AuthorityId, Kind: row.Kind, BeforeRevision: row.BeforeRevision, AfterRevision: row.AfterRevision, Before: historyGrant(row.Before), After: historyGrant(row.After), ActorId: row.ActorId, ActorName: row.ActorName, TraceId: row.TraceId, CreatedAt: row.CreatedAt}
}
