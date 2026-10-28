package accessutil_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/audit"
	base "go-zero-admin/pkg/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func historyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	conn, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	conn.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = conn.Close() })
	if err = db.AutoMigrate(&model.SysAuthority{}, &model.SysBaseMenu{}, &model.SysBaseMenuBtn{}, &model.SysAuthorityMenu{}, &model.SysAuthorityBtn{}, &gormadapter.CasbinRule{}, &model.SysRoleDataScope{}, &model.SysRoleScopeDepartment{}, &model.SysPermissionVersion{}, &model.SysPermissionChange{}); err != nil {
		t.Fatal(err)
	}
	root := int64(0)
	if err = db.Create(&model.SysAuthority{AuthorityId: 88, AuthorityName: "reviewed", ParentId: &root}).Error; err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.SysPermissionVersion{ID: 1, Revision: 1}).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPermissionHistoryCapturesFourControlledKinds(t *testing.T) {
	db := historyDB(t)
	ctx := audit.WithRequest(context.Background(), audit.Request{TraceID: "trace-1", UserAgent: "secret-token-user-agent"})
	audit.SetActor(ctx, audit.Actor{ID: 9, Name: "operator"})
	err := db.Transaction(func(tx *gorm.DB) error {
		before, err := accessutil.CapturePermissionState(tx)
		if err != nil {
			return err
		}
		for _, row := range []any{
			&model.SysAuthorityMenu{AuthorityId: "88", MenuId: "7"},
			&model.SysAuthorityBtn{AuthorityId: 88, SysMenuID: 7, SysBaseMenuBtnID: 11},
			&gormadapter.CasbinRule{Ptype: "p", V0: "88", V1: "/v1/example", V2: "GET"},
			&model.SysRoleDataScope{AuthorityID: 88, Scope: "custom"},
			&model.SysRoleScopeDepartment{AuthorityID: 88, DepartmentID: 4},
		} {
			if err = tx.Create(row).Error; err != nil {
				return err
			}
		}
		return accessutil.CommitPermissionChanges(ctx, tx, before)
	})
	if err != nil {
		t.Fatal(err)
	}
	var changes []model.SysPermissionChange
	if err = db.Order("id").Find(&changes).Error; err != nil {
		t.Fatal(err)
	}
	if len(changes) != 4 {
		t.Fatalf("want four controlled changes: %#v", changes)
	}
	kinds := map[string]bool{}
	for _, c := range changes {
		kinds[c.Kind] = true
		if c.AuthorityID != 88 || c.BeforeRevision != 1 || c.AfterRevision != 2 || c.ActorID != 9 || c.ActorName != "operator" || c.TraceID != "trace-1" {
			t.Fatalf("invalid actor/version: %#v", c)
		}
		if strings.Contains(c.Before+c.After, "secret-token") || strings.Contains(c.Before+c.After, "password") {
			t.Fatalf("sensitive request entered history: %#v", c)
		}
	}
	for _, kind := range []string{"menu", "button", "api", "dataScope"} {
		if !kinds[kind] {
			t.Fatalf("missing %s", kind)
		}
	}
	revision, err := accessutil.PermissionRevision(db)
	if err != nil || revision != "2" {
		t.Fatalf("revision=%s err=%v", revision, err)
	}
}

func TestPermissionHistoryFailureRollsBackGrantsAndRevision(t *testing.T) {
	db := historyDB(t)
	failure := errors.New("history storage unavailable")
	_ = db.Callback().Create().Before("gorm:create").Register("history_fail", func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_permission_changes" {
			tx.AddError(failure)
		}
	})
	err := db.Transaction(func(tx *gorm.DB) error {
		before, err := accessutil.CapturePermissionState(tx)
		if err != nil {
			return err
		}
		if err = tx.Create(&model.SysAuthorityMenu{AuthorityId: "88", MenuId: "7"}).Error; err != nil {
			return err
		}
		return accessutil.CommitPermissionChanges(context.Background(), tx, before)
	})
	if !errors.Is(err, failure) {
		t.Fatalf("want injected history failure, got %v", err)
	}
	var count int64
	if err = db.Model(&model.SysAuthorityMenu{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("grant partially committed: %d %v", count, err)
	}
	revision, err := accessutil.PermissionRevision(db)
	if err != nil || revision != "1" {
		t.Fatalf("revision partially committed: %s %v", revision, err)
	}
}

func TestPermissionHistoryCanonicalOrderingAndResourceOnlyRevision(t *testing.T) {
	db := historyDB(t)
	if err := db.Create([]model.SysAuthorityMenu{{AuthorityId: "88", MenuId: "9"}, {AuthorityId: "88", MenuId: "7"}, {AuthorityId: "88", MenuId: "7"}}).Error; err != nil {
		t.Fatal(err)
	}
	err := db.Transaction(func(tx *gorm.DB) error {
		before, err := accessutil.CapturePermissionState(tx)
		if err != nil {
			return err
		}
		if err = tx.Where("sys_authority_authority_id = ?", 88).Delete(&model.SysAuthorityMenu{}).Error; err != nil {
			return err
		}
		if err = tx.Create([]model.SysAuthorityMenu{{AuthorityId: "88", MenuId: "7"}, {AuthorityId: "88", MenuId: "9"}}).Error; err != nil {
			return err
		}
		if err = tx.Create(&model.SysBaseMenu{MODEL_BASE: base.MODEL_BASE{ID: 15}, Name: "resource"}).Error; err != nil {
			return err
		}
		return accessutil.CommitPermissionChanges(context.Background(), tx, before)
	})
	if err != nil {
		t.Fatal(err)
	}
	var count int64
	if err = db.Model(&model.SysPermissionChange{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("ordering created false history: %d %v", count, err)
	}
	revision, err := accessutil.PermissionRevision(db)
	if err != nil || revision != "2" {
		t.Fatalf("resource revision=%s %v", revision, err)
	}
}
