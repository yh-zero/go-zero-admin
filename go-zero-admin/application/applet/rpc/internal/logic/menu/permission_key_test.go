package menulogic

import (
	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
	"strings"
	"testing"
)

func TestStableKeyOnMenuRelations(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.SysBaseMenu{}, &model.SysBaseMenuBtn{}, &model.SysBaseMenuParameter{}, &model.SysAuthorityBtn{}); err != nil {
		t.Fatal(err)
	}
	m := &pb.SysBaseMenu{ID: 7, Name: "custom", MenuBtn: []*pb.SysBaseMenuBtn{{Name: "create"}}}
	if err = saveMenuRelations(db, 7, m); err != nil {
		t.Fatal(err)
	}
	var b model.SysBaseMenuBtn
	if err = db.First(&b).Error; err != nil {
		t.Fatal(err)
	}
	if b.PermissionKey == "" {
		t.Fatal("stable permission key missing")
	}
	m.Name = "renamed"
	m.MenuBtn[0].ID = b.ID
	m.MenuBtn[0].Name = "rename"
	if err = saveMenuRelations(db, 7, m); err != nil {
		t.Fatal(err)
	}
	var after model.SysBaseMenuBtn
	if err = db.First(&after, b.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.PermissionKey != b.PermissionKey {
		t.Fatalf("stable key changed: %s -> %s", b.PermissionKey, after.PermissionKey)
	}
}

func TestLegacyPermissionKeyPreservesFullNameLimits(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err = db.AutoMigrate(&model.SysBaseMenu{}, &model.SysBaseMenuBtn{}, &model.SysBaseMenuParameter{}, &model.SysAuthorityBtn{}); err != nil {
		t.Fatal(err)
	}
	menuName := strings.Repeat("界", 191)
	buttonName := strings.Repeat("键", 191)
	legacyKey := menuName + ":" + buttonName
	m := model.SysBaseMenu{Name: menuName, Path: "/legacy-long-key", Meta: model.Meta{Title: "legacy"}}
	if err = db.Create(&m).Error; err != nil {
		t.Fatal(err)
	}
	b := model.SysBaseMenuBtn{Name: buttonName, PermissionKey: legacyKey, SysBaseMenuID: m.ID}
	if err = db.Create(&b).Error; err != nil {
		t.Fatal(err)
	}
	input := &pb.SysBaseMenu{ID: m.ID, Name: m.Name, Path: m.Path, Meta: &pb.Meta{Title: "legacy"}, MenuBtn: []*pb.SysBaseMenuBtn{{ID: b.ID, Name: b.Name, PermissionKey: b.PermissionKey}}}
	if err = validateMenu(db, input); err != nil {
		t.Fatalf("valid historical varchar(191) names/key must remain editable: %v", err)
	}
	if err = saveMenuRelations(db, m.ID, input); err != nil {
		t.Fatal(err)
	}
	var retained model.SysBaseMenuBtn
	if err = db.First(&retained, b.ID).Error; err != nil {
		t.Fatal(err)
	}
	if retained.PermissionKey != legacyKey || retained.Name != buttonName {
		t.Fatal("historical permission key or button name was truncated")
	}
	input.MenuBtn[0].ID = 0
	input.MenuBtn[0].PermissionKey = ""
	if err = validateMenu(db, input); err != nil {
		t.Fatalf("a new button must support the existing name column length: %v", err)
	}
	input.MenuBtn[0].PermissionKey = strings.Repeat("码", 384)
	if err = validateMenu(db, input); err == nil {
		t.Fatal("permission keys exceeding both historical names plus separator must be rejected")
	}
}
