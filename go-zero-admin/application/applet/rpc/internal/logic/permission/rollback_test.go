package permissionlogic

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/middlecasbin"
	base "go-zero-admin/pkg/model"
	"go-zero-admin/pkg/orm"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func rollbackService(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%x?mode=memory&cache=shared", sha256.Sum256([]byte(t.Name())))), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err = db.AutoMigrate(&model.SysAuthority{}, &model.SysBaseMenu{}, &model.SysBaseMenuBtn{}, &model.SysAuthorityMenu{}, &model.SysAuthorityBtn{}, &gormadapter.CasbinRule{}, &model.SysRoleDataScope{}, &model.SysRoleScopeDepartment{}, &model.SysDepartment{}, &model.SysApi{}, &model.SysPermissionVersion{}, &model.SysPermissionChange{}, &middlecasbin.PolicyVersion{}, &audit.Event{}); err != nil {
		t.Fatal(err)
	}
	root := int64(0)
	for _, r := range []any{&model.SysAuthority{AuthorityId: 88, AuthorityName: "role", ParentId: &root, DefaultRouter: ""}, &model.SysBaseMenu{MODEL_BASE: base.MODEL_BASE{ID: 7}, Name: "menu"}, &model.SysAuthorityMenu{AuthorityId: "88", MenuId: "7"}, &model.SysPermissionVersion{ID: 1, Revision: 5}, &model.SysPermissionChange{ID: 1, AuthorityID: 88, Kind: "menu", BeforeRevision: 4, AfterRevision: 5, Before: `{}`, After: `{"menuIds":[7]}`}} {
		if err = db.Create(r).Error; err != nil {
			t.Fatal(err)
		}
	}
	return &svc.ServiceContext{DB: &orm.DB{DB: db}}
}

func TestRollbackRejectsSubsequentGrantChanges(t *testing.T) {
	s := rollbackService(t)
	if err := s.DB.Model(&model.SysPermissionVersion{}).Where("id = 1").Update("revision", 6).Error; err != nil {
		t.Fatal(err)
	}
	result, err := NewPreviewPermissionRollbackLogic(context.Background(), s).PreviewPermissionRollback(&pb.PreviewPermissionRollbackRequest{ID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Allowed || result.Reason == "" {
		t.Fatalf("later change was rollbackable: %v", result)
	}
	_, err = NewApplyPermissionRollbackLogic(context.Background(), s).ApplyPermissionRollback(&pb.ApplyPermissionRollbackRequest{ID: 1, ExpectedRevision: "6", Version: result.Version})
	if err == nil {
		t.Fatal("later grants overwritten")
	}
}

func TestRollbackRejectsMissingOrUncontrolledHistory(t *testing.T) {
	for _, bad := range []string{"", `null`, `{"menuIds":[7],"password":"secret"}`, `{"menuIds":[7]`} {
		t.Run(bad, func(t *testing.T) {
			s := rollbackService(t)
			if err := s.DB.Model(&model.SysPermissionChange{}).Where("id = 1").Update("before", bad).Error; err != nil {
				t.Fatal(err)
			}
			result, err := NewPreviewPermissionRollbackLogic(context.Background(), s).PreviewPermissionRollback(&pb.PreviewPermissionRollbackRequest{ID: 1})
			if err == nil && result.Allowed {
				t.Fatal("incomplete history allowed")
			}
		})
	}
}

func TestRollbackRejectsDeletedResource(t *testing.T) {
	s := rollbackService(t)
	if err := s.DB.Model(&model.SysPermissionChange{}).Where("id = 1").Update("before", `{"menuIds":[9]}`).Error; err != nil {
		t.Fatal(err)
	}
	result, err := NewPreviewPermissionRollbackLogic(context.Background(), s).PreviewPermissionRollback(&pb.PreviewPermissionRollbackRequest{ID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Allowed || result.Reason == "" {
		t.Fatal("deleted menu restoration allowed")
	}
}

func TestRollbackInverseAndHistoryCommitAtomically(t *testing.T) {
	s := rollbackService(t)
	ctx := context.Background()
	preview, err := NewPreviewPermissionRollbackLogic(ctx, s).PreviewPermissionRollback(&pb.PreviewPermissionRollbackRequest{ID: 1})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Allowed || preview.Version == "" {
		t.Fatalf("valid inverse unavailable: %v", preview)
	}
	var count int64
	if err = s.DB.Model(&model.SysAuthorityMenu{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("preview mutated grants: %d %v", count, err)
	}
	_, err = NewApplyPermissionRollbackLogic(ctx, s).ApplyPermissionRollback(&pb.ApplyPermissionRollbackRequest{ID: 1, ExpectedRevision: "5", Version: preview.Version})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.DB.Model(&model.SysAuthorityMenu{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("inverse not applied: %d %v", count, err)
	}
	var revision model.SysPermissionVersion
	if err = s.DB.First(&revision, 1).Error; err != nil || revision.Revision != 6 {
		t.Fatalf("revision missing: %#v %v", revision, err)
	}
	if err = s.DB.Model(&model.SysPermissionChange{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("inverse not audited: %d %v", count, err)
	}
}

func TestRollbackRejectsChangedPreviewAndState(t *testing.T) {
	for _, scenario := range []string{"preview", "revision", "grants"} {
		t.Run(scenario, func(t *testing.T) {
			s := rollbackService(t)
			ctx := context.Background()
			preview, err := NewPreviewPermissionRollbackLogic(ctx, s).PreviewPermissionRollback(&pb.PreviewPermissionRollbackRequest{ID: 1})
			if err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "preview":
				preview.Version = "forged"
			case "revision":
				err = s.DB.Model(&model.SysPermissionVersion{}).Where("id=1").Update("revision", 6).Error
			case "grants":
				err = s.DB.Create(&model.SysAuthorityMenu{AuthorityId: "88", MenuId: "9"}).Error
			}
			if err != nil {
				t.Fatal(err)
			}
			_, err = NewApplyPermissionRollbackLogic(ctx, s).ApplyPermissionRollback(&pb.ApplyPermissionRollbackRequest{ID: 1, ExpectedRevision: "5", Version: preview.Version})
			if err == nil {
				t.Fatal("stale preview/state accepted")
			}
		})
	}
}

func TestRollbackRestoresButtonAPIAndDataScopeInverse(t *testing.T) {
	for _, kind := range []string{"button", "api", "dataScope"} {
		t.Run(kind, func(t *testing.T) {
			s := rollbackService(t)
			var before, after string
			var records []any
			switch kind {
			case "button":
				before = `{"menuBtnIds":[11]}`
				after = `{"menuBtnIds":[12]}`
				records = []any{&model.SysBaseMenuBtn{MODEL_BASE: base.MODEL_BASE{ID: 11}, Name: "old", PermissionKey: "menu.7.button.old", SysBaseMenuID: 7}, &model.SysBaseMenuBtn{MODEL_BASE: base.MODEL_BASE{ID: 12}, Name: "new", PermissionKey: "menu.7.button.new", SysBaseMenuID: 7}, &model.SysAuthorityBtn{AuthorityId: 88, SysMenuID: 7, SysBaseMenuBtnID: 12}}
			case "api":
				before = `{"policies":[{"path":"/v1/old","method":"GET"}]}`
				after = `{"policies":[{"path":"/v1/new","method":"POST"}]}`
				records = []any{&model.SysApi{Path: "/v1/old", Method: "GET"}, &model.SysApi{Path: "/v1/new", Method: "POST"}, &gormadapter.CasbinRule{Ptype: "p", V0: "88", V1: "/v1/new", V2: "POST"}}
			case "dataScope":
				before = `{"dataScope":{"scope":"custom","departmentIds":[4]}}`
				after = `{"dataScope":{"scope":"self","departmentIds":[]}}`
				records = []any{&model.SysDepartment{MODEL_BASE: base.MODEL_BASE{ID: 4}, Name: "department", Code: "dept", Status: 1}, &model.SysRoleDataScope{AuthorityID: 88, Scope: "self"}}
			}
			for _, record := range records {
				if err := s.DB.Create(record).Error; err != nil {
					t.Fatal(err)
				}
			}
			if err := s.DB.Model(&model.SysPermissionChange{}).Where("id = 1").Updates(map[string]any{"kind": kind, "before": before, "after": after}).Error; err != nil {
				t.Fatal(err)
			}
			preview, err := NewPreviewPermissionRollbackLogic(context.Background(), s).PreviewPermissionRollback(&pb.PreviewPermissionRollbackRequest{ID: 1})
			if err != nil || !preview.Allowed {
				t.Fatalf("inverse preview: %v %v", preview, err)
			}
			_, err = NewApplyPermissionRollbackLogic(context.Background(), s).ApplyPermissionRollback(&pb.ApplyPermissionRollbackRequest{ID: 1, ExpectedRevision: "5", Version: preview.Version})
			if err != nil {
				t.Fatal(err)
			}
			state, err := accessutil.CapturePermissionState(s.DB.DB)
			if err != nil {
				t.Fatal(err)
			}
			doc := accessutil.GrantDocument(state.Roles[88], kind)
			target, err := decodeGrant(before, kind)
			if err != nil {
				t.Fatal(err)
			}
			got, _ := json.Marshal(doc)
			want, _ := json.Marshal(target)
			if string(got) != string(want) {
				t.Fatalf("inverse grants: got %s want %s", got, want)
			}
		})
	}
}

func TestRollbackHistoryFailureLeavesOriginalGrants(t *testing.T) {
	s := rollbackService(t)
	preview, err := NewPreviewPermissionRollbackLogic(context.Background(), s).PreviewPermissionRollback(&pb.PreviewPermissionRollbackRequest{ID: 1})
	if err != nil || !preview.Allowed {
		t.Fatalf("preview=%v err=%v", preview, err)
	}
	failure := errors.New("history storage rejected inverse")
	if err = s.DB.Callback().Create().Before("gorm:create").Register("rollback_history_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_permission_changes" {
			tx.AddError(failure)
		}
	}); err != nil {
		t.Fatal(err)
	}
	_, err = NewApplyPermissionRollbackLogic(context.Background(), s).ApplyPermissionRollback(&pb.ApplyPermissionRollbackRequest{ID: 1, ExpectedRevision: "5", Version: preview.Version})
	if !errors.Is(err, failure) {
		t.Fatalf("missing history failure: %v", err)
	}
	var count int64
	if err = s.DB.Model(&model.SysAuthorityMenu{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("partial inverse: %d %v", count, err)
	}
	revision, err := accessutil.PermissionRevision(s.DB.DB)
	if err != nil || revision != "5" {
		t.Fatalf("partial revision: %s %v", revision, err)
	}
}
