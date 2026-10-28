package datascope

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	base "go-zero-admin/pkg/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type ownedRecord struct {
	ID           int64
	OwnerID      int64
	DepartmentID int64
}

func TestScopeChecksDeviceSession(t *testing.T) {
	db, actor := scopeDB(t)
	actor.SessionID = "valid-device"
	if _, err := Apply(context.Background(), db, actor, "owner_id", "department_id"); err == nil {
		t.Fatal("unknown device accepted")
	}
	device := model.SysDeviceSession{ID: actor.SessionID, UserID: actor.UserID, AuthorityID: actor.AuthorityId, SessionVersion: actor.SessionVersion, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	if err := db.Create(&device).Error; err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db, actor); len(got) != 2 {
		t.Fatal("active device cannot query", got)
	}
	if err := db.Model(&device).Update("revoked_at", time.Now().UTC()).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), db, actor, "owner_id", "department_id"); err == nil {
		t.Fatal("revoked device queried records")
	}
}

func scopeDB(t *testing.T) (*gorm.DB, *pb.SessionRequest) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.SysUser{}, &model.SysAuthority{}, &model.SysUserAuthority{}, &model.SysDeviceSession{}, &model.SysDepartment{}, &model.SysUserDepartment{}, &model.SysRoleDataScope{}, &model.SysRoleScopeDepartment{}, &ownedRecord{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{1, 2} {
		if err := db.Create(&model.SysAuthority{AuthorityId: id, AuthorityName: "role"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []model.SysUser{{MODEL_BASE: base.MODEL_BASE{ID: 10}, Username: "member", AuthorityId: 2, Enable: 1, SessionVersion: 1}, {MODEL_BASE: base.MODEL_BASE{ID: 30}, Username: "admin", AuthorityId: 1, Enable: 1, SessionVersion: 1}} {
		if err := db.Omit("Authority", "Authorities").Create(&u).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.SysUserAuthority{SysUserId: u.ID, SysAuthorityAuthorityId: u.AuthorityId}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range []model.SysDepartment{{MODEL_BASE: base.MODEL_BASE{ID: 1}, Name: "root", Code: "root", Status: 1}, {MODEL_BASE: base.MODEL_BASE{ID: 2}, ParentID: 1, Name: "child", Code: "child", Status: 1}, {MODEL_BASE: base.MODEL_BASE{ID: 3}, Name: "other", Code: "other", Status: 1}} {
		if err := db.Create(&d).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.SysUserDepartment{UserID: 10, DepartmentID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]ownedRecord{{1, 10, 1}, {2, 20, 1}, {3, 20, 2}, {4, 10, 3}, {5, 20, 0}}).Error; err != nil {
		t.Fatal(err)
	}
	return db, &pb.SessionRequest{UserID: 10, AuthorityId: 2, SessionVersion: 1}
}
func ids(t *testing.T, db *gorm.DB, actor *pb.SessionRequest) []int64 {
	t.Helper()
	query, err := Apply(context.Background(), db.Model(&ownedRecord{}), actor, "owner_id", "department_id")
	if err != nil {
		t.Fatal(err)
	}
	var out []int64
	if err := query.Order("id ASC").Pluck("id", &out).Error; err != nil {
		t.Fatal(err)
	}
	return out
}
func TestEveryDataScopeAndQueryClauses(t *testing.T) {
	db, actor := scopeDB(t)
	if got := ids(t, db, actor); !reflect.DeepEqual(got, []int64{1, 4}) {
		t.Fatal("unconfigured role must only read owned records", got)
	}
	for scope, want := range map[string][]int64{"all": {1, 2, 3, 4, 5}, "self": {1, 4}, "department": {1, 2}, "department_and_children": {1, 2, 3}, "custom": {4}} {
		if err := db.Where("authority_id = ?", 2).Delete(&model.SysRoleDataScope{}).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.SysRoleDataScope{AuthorityID: 2, Scope: scope}).Error; err != nil {
			t.Fatal(err)
		}
		if scope == "custom" {
			if err := db.Create(&model.SysRoleScopeDepartment{AuthorityID: 2, DepartmentID: 3}).Error; err != nil {
				t.Fatal(err)
			}
		}
		if got := ids(t, db, actor); !reflect.DeepEqual(got, want) {
			t.Fatalf("scope %s got %v want %v", scope, got, want)
		}
	}
	if err := db.Model(&model.SysRoleDataScope{}).Where("authority_id = ?", 2).Update("scope", "self").Error; err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db.Where("id > ?", 1), actor); !reflect.DeepEqual(got, []int64{4}) {
		t.Fatal("scope lost the caller's existing query clauses", got)
	}
	admin := &pb.SessionRequest{UserID: 30, AuthorityId: 1, SessionVersion: 1}
	if got := ids(t, db, admin); len(got) != 5 {
		t.Fatal("admin data scope is restricted", got)
	}
	if err := db.Model(&model.SysRoleDataScope{}).Where("authority_id = ?", 2).Update("scope", "department").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("user_id = ?", 10).Delete(&model.SysUserDepartment{}).Error; err != nil {
		t.Fatal(err)
	}
	if got := ids(t, db, actor); len(got) != 0 {
		t.Fatal("unassigned user received department records", got)
	}
}

func TestScopeRejectsStaleSessionAndConfigurationFailures(t *testing.T) {
	db, actor := scopeDB(t)
	stale := &pb.SessionRequest{UserID: actor.UserID, AuthorityId: actor.AuthorityId, SessionVersion: actor.SessionVersion + 1, SessionID: actor.SessionID}
	if _, err := Apply(context.Background(), db, stale, "owner_id", "department_id"); err == nil {
		t.Fatal("stale session accepted")
	}
	if _, err := Apply(context.Background(), db, actor, "owner_id OR 1=1", "department_id"); err == nil {
		t.Fatal("SQL identifier injection accepted")
	}
	if err := db.Create(&model.SysRoleDataScope{AuthorityID: 2, Scope: "unknown"}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), db, actor, "owner_id", "department_id"); err == nil {
		t.Fatal("invalid configured scope treated as permission")
	}
	if err := db.Model(&model.SysRoleDataScope{}).Where("authority_id = ?", 2).Update("scope", "custom").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SysRoleScopeDepartment{AuthorityID: 2, DepartmentID: 999}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), db, actor, "owner_id", "department_id"); err == nil {
		t.Fatal("missing custom department ignored")
	}
	if err := db.Migrator().DropTable(&model.SysRoleDataScope{}); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(context.Background(), db, actor, "owner_id", "department_id"); err == nil {
		t.Fatal("database failure treated as permission")
	}
}
