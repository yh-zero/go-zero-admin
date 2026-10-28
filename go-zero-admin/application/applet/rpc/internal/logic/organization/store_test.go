package organizationlogic

import (
	"context"
	"fmt"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"testing"

	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	base "go-zero-admin/pkg/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testStore(t *testing.T) store {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.SysPermissionVersion{}, &model.SysPermissionChange{}, &model.SysBaseMenu{}, &model.SysBaseMenuBtn{}, &model.SysBaseMenuParameter{}, &model.SysAuthorityMenu{}, &model.SysAuthorityBtn{}, &model.SysApi{}, &gormadapter.CasbinRule{}, &model.SysAuthority{}, &model.SysUser{}, &model.SysUserAuthority{}, &model.SysDepartment{}, &model.SysPosition{}, &model.SysUserDepartment{}, &model.SysUserPosition{}, &model.SysRoleDataScope{}, &model.SysRoleScopeDepartment{}, &model.SysFileResource{}, &audit.Event{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SysPermissionVersion{ID: 1, Revision: 1}).Error; err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{1, 2} {
		if err := db.Create(&model.SysAuthority{AuthorityId: id, AuthorityName: "role"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Omit("Authority", "Authorities").Create(&model.SysUser{MODEL_BASE: base.MODEL_BASE{ID: 10}, Username: "member", AuthorityId: 2, Enable: 1, SessionVersion: 1}).Error; err != nil {
		t.Fatal(err)
	}
	return store{ctx: context.Background(), db: db}
}
func mustDepartment(t *testing.T, s store, parent int64, name, code string) *pb.Department {
	t.Helper()
	d, err := s.saveDepartment(&pb.DepartmentRequest{Department: &pb.Department{ParentId: parent, Name: name, Code: code, Status: 1}}, true)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func mustPosition(t *testing.T, s store, name, code string) *pb.Position {
	t.Helper()
	p, err := s.savePosition(&pb.PositionRequest{Position: &pb.Position{Name: name, Code: code, Status: 1}}, true)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func version(t *testing.T, s store) int64 {
	t.Helper()
	var user model.SysUser
	if err := s.db.First(&user, 10).Error; err != nil {
		t.Fatal(err)
	}
	return user.SessionVersion
}

func TestDepartmentTreeValidationAndDeletion(t *testing.T) {
	s := testStore(t)
	root := mustDepartment(t, s, 0, "总部", "root")
	child := mustDepartment(t, s, root.ID, "研发部", "engineering")
	leaf := mustDepartment(t, s, child.ID, "研发一组", "team")
	filtered, err := s.departments(&pb.OrganizationListRequest{Keyword: "一组"})
	if err != nil || len(filtered.List) != 1 || len(filtered.List[0].Children) != 1 || filtered.List[0].Children[0].Children[0].ID != leaf.ID {
		t.Fatal("filter lost ancestor tree", filtered, err)
	}
	root.ParentId = leaf.ID
	if _, err := s.saveDepartment(&pb.DepartmentRequest{Department: root}, false); err == nil {
		t.Fatal("department cycle accepted")
	}
	root.ParentId = 0
	if _, err := s.saveDepartment(&pb.DepartmentRequest{Department: &pb.Department{ParentId: 0, Name: "different", Code: " ROOT ", Status: 1}}, true); err == nil {
		t.Fatal("case-insensitive duplicate department code accepted")
	}
	if err := s.deleteDepartment(&pb.OrganizationIDRequest{ID: root.ID}); err == nil {
		t.Fatal("department with children deleted")
	}
	child.Status = 2
	if _, err := s.saveDepartment(&pb.DepartmentRequest{Department: child}, false); err == nil {
		t.Fatal("department with enabled child disabled")
	}
	leaf.Status = 2
	if _, err := s.saveDepartment(&pb.DepartmentRequest{Department: leaf}, false); err != nil {
		t.Fatal(err)
	}
	child.Status = 2
	if _, err := s.saveDepartment(&pb.DepartmentRequest{Department: child}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.saveDepartment(&pb.DepartmentRequest{Department: &pb.Department{ParentId: child.ID, Name: "new", Code: "new", Status: 1}}, true); err == nil {
		t.Fatal("enabled child assigned to disabled parent")
	}
	if err := s.deleteDepartment(&pb.OrganizationIDRequest{ID: leaf.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.deleteDepartment(&pb.OrganizationIDRequest{ID: child.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.deleteDepartment(&pb.OrganizationIDRequest{ID: root.ID}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.saveDepartment(&pb.DepartmentRequest{Department: &pb.Department{Name: "reuse", Code: "root", Status: 1}}, true); err != nil {
		t.Fatal("deleted code could not be reused", err)
	}
}

func TestMembershipIsAtomicAndRevokesOnlyOnChanges(t *testing.T) {
	s := testStore(t)
	a := mustDepartment(t, s, 0, "A", "a")
	b := mustDepartment(t, s, 0, "B", "b")
	p := mustPosition(t, s, "开发", "dev")
	q := mustPosition(t, s, "测试", "qa")
	input := &pb.MembershipRequest{UserID: 10, DepartmentId: a.ID, PositionIds: []int64{p.ID, p.ID}}
	if err := s.updateMembership(input); err != nil {
		t.Fatal(err)
	}
	if version(t, s) != 2 {
		t.Fatal("membership update kept old session")
	}
	if err := s.updateMembership(input); err != nil {
		t.Fatal(err)
	}
	if version(t, s) != 2 {
		t.Fatal("unchanged membership revoked session")
	}
	if err := s.deleteDepartment(&pb.OrganizationIDRequest{ID: a.ID}); err == nil {
		t.Fatal("assigned department deleted")
	}
	if err := s.deletePosition(&pb.OrganizationIDRequest{ID: p.ID}); err == nil {
		t.Fatal("assigned position deleted")
	}
	b.Status = 2
	if _, err := s.saveDepartment(&pb.DepartmentRequest{Department: b}, false); err != nil {
		t.Fatal(err)
	}
	if err := s.updateMembership(&pb.MembershipRequest{UserID: 10, DepartmentId: b.ID}); err == nil {
		t.Fatal("disabled department assigned")
	}
	if version(t, s) != 2 {
		t.Fatal("rejected membership revoked session")
	}
	b.Status = 1
	if _, err := s.saveDepartment(&pb.DepartmentRequest{Department: b}, false); err != nil {
		t.Fatal(err)
	}
	if err := s.db.Exec(fmt.Sprintf("CREATE TRIGGER reject_position BEFORE INSERT ON sys_user_positions WHEN NEW.position_id = %d BEGIN SELECT RAISE(FAIL, 'membership rejected'); END", q.ID)).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.updateMembership(&pb.MembershipRequest{UserID: 10, DepartmentId: b.ID, PositionIds: []int64{q.ID}}); err == nil {
		t.Fatal("expected position insert failure")
	}
	stored, err := s.membership(&pb.OrganizationIDRequest{ID: 10})
	if err != nil || stored.DepartmentId != a.ID || len(stored.PositionIds) != 1 || stored.PositionIds[0] != p.ID || version(t, s) != 2 {
		t.Fatal("failed membership update was not rolled back", stored, err)
	}
	if err := s.updateMembership(&pb.MembershipRequest{UserID: 10}); err != nil {
		t.Fatal(err)
	}
	if version(t, s) != 3 {
		t.Fatal("membership removal kept session")
	}
	if err := s.deletePosition(&pb.OrganizationIDRequest{ID: p.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.deleteDepartment(&pb.OrganizationIDRequest{ID: a.ID}); err != nil {
		t.Fatal(err)
	}
}

func TestRoleDataScopeValidationAndDepartmentReferences(t *testing.T) {
	s := testStore(t)
	d := mustDepartment(t, s, 0, "研发", "dev")
	defaultScope, err := s.dataScope(&pb.GetRoleDataScopeRequest{AuthorityId: 2})
	if err != nil || defaultScope.DataScope.Scope != "self" {
		t.Fatal("default data scope must be self", err)
	}
	for _, input := range []*pb.RoleDataScope{{AuthorityId: 2, Scope: "unknown"}, {AuthorityId: 2, Scope: "custom"}, {AuthorityId: 2, Scope: "all", DepartmentIds: []int64{d.ID}}, {AuthorityId: 1, Scope: "self"}, {AuthorityId: 999, Scope: "all"}} {
		if err := s.updateDataScope(&pb.RoleDataScopeRequest{DataScope: input}); err == nil {
			t.Fatal("invalid data scope accepted", input)
		}
	}
	if err := s.updateDataScope(&pb.RoleDataScopeRequest{DataScope: &pb.RoleDataScope{ExpectedRevision: permissionRevision(t, s), AuthorityId: 2, Scope: "custom", DepartmentIds: []int64{d.ID, d.ID}}}); err != nil {
		t.Fatal(err)
	}
	got, err := s.dataScope(&pb.GetRoleDataScopeRequest{AuthorityId: 2})
	if err != nil || got.DataScope.Scope != "custom" || len(got.DataScope.DepartmentIds) != 1 {
		t.Fatal("custom scope not stored", got, err)
	}
	if err := s.deleteDepartment(&pb.OrganizationIDRequest{ID: d.ID}); err == nil {
		t.Fatal("custom-scope department deleted")
	}
	if err := s.updateDataScope(&pb.RoleDataScopeRequest{DataScope: &pb.RoleDataScope{ExpectedRevision: permissionRevision(t, s), AuthorityId: 2, Scope: "self"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.deleteDepartment(&pb.OrganizationIDRequest{ID: d.ID}); err != nil {
		t.Fatal(err)
	}
}

func TestAuditFailureRollsBackOrganizationWrite(t *testing.T) {
	s := testStore(t)
	mustDepartment(t, s, 0, "existing", "existing")
	var events int64
	if err := s.db.Model(&audit.Event{}).Count(&events).Error; err != nil || events != 1 {
		t.Fatal("organization change did not record audit", events, err)
	}
	if err := s.db.Migrator().DropTable(&audit.Event{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.saveDepartment(&pb.DepartmentRequest{Department: &pb.Department{Name: "blocked", Code: "blocked", Status: 1}}, true); err == nil {
		t.Fatal("audit persistence failure did not fail the transaction")
	}
	var count int64
	if err := s.db.Model(&model.SysDepartment{}).Where("code = ?", "blocked").Count(&count).Error; err != nil || count != 0 {
		t.Fatal("unaudited department was committed", count, err)
	}
}

func TestDepartmentDeletionProtectsLiveFileOwnership(t *testing.T) {
	s := testStore(t)
	department := mustDepartment(t, s, 0, "Files", "files")
	file := model.SysFileResource{ObjectKey: "department-file", Name: "file.png", Mime: "image/png", Size: 10, OwnerID: 10, DepartmentID: department.ID, Visibility: "private", Status: "active"}
	if err := s.db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	for _, status := range []string{"active", "deleting"} {
		if err := s.db.Model(&file).Update("status", status).Error; err != nil {
			t.Fatal(err)
		}
		if err := s.deleteDepartment(&pb.OrganizationIDRequest{ID: department.ID}); err == nil {
			t.Fatal("department with live file snapshot deleted", status)
		}
	}
	if err := s.db.Model(&file).Update("status", "deleted").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.deleteDepartment(&pb.OrganizationIDRequest{ID: department.ID}); err != nil {
		t.Fatal("archived file should not prevent department deletion", err)
	}
}

func permissionRevision(t *testing.T, s store) string {
	t.Helper()
	rev, err := accessutil.PermissionRevision(s.db)
	if err != nil {
		t.Fatal(err)
	}
	return rev
}
