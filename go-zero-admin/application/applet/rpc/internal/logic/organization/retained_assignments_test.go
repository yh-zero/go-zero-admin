package organizationlogic

import (
	"testing"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
)

func disableDepartment(t *testing.T, s store, d *pb.Department) {
	t.Helper()
	d.Status = 2
	if _, err := s.saveDepartment(&pb.DepartmentRequest{Department: d}, false); err != nil {
		t.Fatal(err)
	}
}

func disablePosition(t *testing.T, s store, p *pb.Position) {
	t.Helper()
	p.Status = 2
	if _, err := s.savePosition(&pb.PositionRequest{Position: p}, false); err != nil {
		t.Fatal(err)
	}
}

func requireMembership(t *testing.T, s store, department int64, positions []int64, sessionVersion int64) {
	t.Helper()
	got, err := s.membership(&pb.OrganizationIDRequest{ID: 10})
	if err != nil || got.DepartmentId != department || !sameIDs(got.PositionIds, positions) || version(t, s) != sessionVersion {
		t.Fatal("unexpected stored membership/session version", got, version(t, s), err)
	}
}

func sameIDs(first, second []int64) bool {
	if len(first) != len(second) {
		return false
	}
	for i, id := range first {
		if second[i] != id {
			return false
		}
	}
	return true
}

func TestMembershipRetainsDisabledAssignmentsAndRejectsNewOnes(t *testing.T) {
	s := testStore(t)
	a := mustDepartment(t, s, 0, "A", "a")
	b := mustDepartment(t, s, 0, "B", "b")
	p := mustPosition(t, s, "Original", "original")
	q := mustPosition(t, s, "Additional", "additional")
	r := mustPosition(t, s, "Unavailable", "unavailable")
	input := &pb.MembershipRequest{UserID: 10, DepartmentId: a.ID, PositionIds: []int64{p.ID}}
	if err := s.updateMembership(input); err != nil {
		t.Fatal(err)
	}
	disableDepartment(t, s, a)
	disablePosition(t, s, p)
	disablePosition(t, s, r)
	if err := s.updateMembership(input); err != nil {
		t.Fatal("unchanged disabled assignments must remain valid", err)
	}
	requireMembership(t, s, a.ID, []int64{p.ID}, 2)
	input.DepartmentId = b.ID
	if err := s.updateMembership(input); err != nil {
		t.Fatal("disabled original position prevented department change", err)
	}
	requireMembership(t, s, b.ID, []int64{p.ID}, 3)
	if err := s.updateMembership(&pb.MembershipRequest{UserID: 10, DepartmentId: a.ID, PositionIds: []int64{p.ID}}); err == nil {
		t.Fatal("reassigning the disabled previous department was accepted")
	}
	input.PositionIds = []int64{p.ID, q.ID}
	if err := s.updateMembership(input); err != nil {
		t.Fatal("retaining disabled original position while adding enabled position failed", err)
	}
	requireMembership(t, s, b.ID, []int64{p.ID, q.ID}, 4)
	disableDepartment(t, s, b)
	input.PositionIds = []int64{p.ID}
	if err := s.updateMembership(input); err != nil {
		t.Fatal("disabled original department prevented position removal", err)
	}
	requireMembership(t, s, b.ID, []int64{p.ID}, 5)
	input.PositionIds = []int64{p.ID, r.ID}
	if err := s.updateMembership(input); err == nil {
		t.Fatal("new disabled position accepted")
	}
	requireMembership(t, s, b.ID, []int64{p.ID}, 5)
}

func TestCustomScopeRetainsDisabledDepartmentsWithoutReassignmentBypass(t *testing.T) {
	s := testStore(t)
	a := mustDepartment(t, s, 0, "A", "a")
	b := mustDepartment(t, s, 0, "B", "b")
	c := mustDepartment(t, s, 0, "C", "c")
	set := func(ids ...int64) error {
		return s.updateDataScope(&pb.RoleDataScopeRequest{DataScope: &pb.RoleDataScope{ExpectedRevision: permissionRevision(t, s), AuthorityId: 2, Scope: "custom", DepartmentIds: ids}})
	}
	if err := set(a.ID); err != nil {
		t.Fatal(err)
	}
	disableDepartment(t, s, a)
	disableDepartment(t, s, c)
	if err := set(a.ID, b.ID); err != nil {
		t.Fatal("retaining disabled custom department while adding enabled department failed", err)
	}
	if err := set(a.ID, c.ID); err == nil {
		t.Fatal("new disabled custom department accepted")
	}
	got, err := s.dataScope(&pb.GetRoleDataScopeRequest{AuthorityId: 2})
	if err != nil || !sameIDs(got.DataScope.DepartmentIds, []int64{a.ID, b.ID}) {
		t.Fatal("failed custom scope write changed stored scope", got, err)
	}
	if err := set(b.ID); err != nil {
		t.Fatal(err)
	}
	if err := set(a.ID, b.ID); err == nil {
		t.Fatal("removed disabled custom department could be reassigned")
	}
	disableDepartment(t, s, b)
	// Simulate a stale relation left by historical data. Only the effective
	// previous custom scope is eligible for retention.
	if err := s.db.Model(&model.SysRoleDataScope{}).Where("authority_id = ?", 2).Update("scope", "self").Error; err != nil {
		t.Fatal(err)
	}
	if err := set(b.ID); err == nil {
		t.Fatal("stale relation from non-custom scope bypassed disabled department validation")
	}
	got, err = s.dataScope(&pb.GetRoleDataScopeRequest{AuthorityId: 2})
	if err != nil || got.DataScope.Scope != "self" {
		t.Fatal("rejected scope switch changed effective scope", got, err)
	}
}

func TestRetainedAssignmentsMustStillExist(t *testing.T) {
	for _, missing := range []string{"department", "position", "scope_department"} {
		t.Run(missing, func(t *testing.T) {
			s := testStore(t)
			d := mustDepartment(t, s, 0, "Existing", "existing")
			p := mustPosition(t, s, "Existing", "existing")
			if err := s.updateMembership(&pb.MembershipRequest{UserID: 10, DepartmentId: d.ID, PositionIds: []int64{p.ID}}); err != nil {
				t.Fatal(err)
			}
			if err := s.updateDataScope(&pb.RoleDataScopeRequest{DataScope: &pb.RoleDataScope{ExpectedRevision: permissionRevision(t, s), AuthorityId: 2, Scope: "custom", DepartmentIds: []int64{d.ID}}}); err != nil {
				t.Fatal(err)
			}
			var err error
			if missing == "position" {
				err = s.db.Unscoped().Delete(&model.SysPosition{}, p.ID).Error
			} else {
				err = s.db.Unscoped().Delete(&model.SysDepartment{}, d.ID).Error
			}
			if err != nil {
				t.Fatal(err)
			}
			if missing == "scope_department" {
				err = s.updateDataScope(&pb.RoleDataScopeRequest{DataScope: &pb.RoleDataScope{ExpectedRevision: permissionRevision(t, s), AuthorityId: 2, Scope: "custom", DepartmentIds: []int64{d.ID}}})
			} else {
				err = s.updateMembership(&pb.MembershipRequest{UserID: 10, DepartmentId: d.ID, PositionIds: []int64{p.ID}})
			}
			if err == nil || version(t, s) != 2 {
				t.Fatal("missing retained assignment accepted or session changed", err)
			}
		})
	}
}

func TestRetainedMembershipAuditFailureRollsBackAssignmentsAndSession(t *testing.T) {
	s := testStore(t)
	a := mustDepartment(t, s, 0, "A", "a")
	b := mustDepartment(t, s, 0, "B", "b")
	p := mustPosition(t, s, "Existing", "existing")
	if err := s.updateMembership(&pb.MembershipRequest{UserID: 10, DepartmentId: a.ID, PositionIds: []int64{p.ID}}); err != nil {
		t.Fatal(err)
	}
	disablePosition(t, s, p)
	if err := s.db.Migrator().DropTable(&audit.Event{}); err != nil {
		t.Fatal(err)
	}
	if err := s.updateMembership(&pb.MembershipRequest{UserID: 10, DepartmentId: b.ID, PositionIds: []int64{p.ID}}); err == nil {
		t.Fatal("failed audit allowed retained membership update")
	}
	requireMembership(t, s, a.ID, []int64{p.ID}, 2)
}
func TestStaleDataScopeCannotBroadenAndMissingRevisionRejects(t *testing.T) {
	s := testStore(t)
	old := permissionRevision(t, s)
	if err := s.updateDataScope(&pb.RoleDataScopeRequest{DataScope: &pb.RoleDataScope{AuthorityId: 2, Scope: "self", ExpectedRevision: old}}); err != nil {
		t.Fatal(err)
	}
	latest := permissionRevision(t, s)
	for _, rev := range []string{old, ""} {
		if err := s.updateDataScope(&pb.RoleDataScopeRequest{DataScope: &pb.RoleDataScope{AuthorityId: 2, Scope: "all", ExpectedRevision: rev}}); err == nil {
			t.Fatal("stale/missing revision widened data scope")
		}
	}
	result, err := s.dataScope(&pb.GetRoleDataScopeRequest{AuthorityId: 2})
	if err != nil {
		t.Fatal(err)
	}
	if result.DataScope.Scope != "self" || result.DataScope.Revision != latest {
		t.Fatalf("scope or revision changed: %+v", result)
	}
}
