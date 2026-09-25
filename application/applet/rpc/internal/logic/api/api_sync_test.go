package apilogic

import (
	"strings"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/data/api/generated"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func syncDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sql.Close() })
	if err := db.AutoMigrate(&model.SysApi{}, &gormadapter.CasbinRule{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSwaggerCatalogOnlyPermissionResources(t *testing.T) {
	source := []byte(`{"swagger":"2.0","basePath":"/v1/sys","paths":{
        "/public":{"get":{"summary":"Public"}},
        "/me":{"get":{"security":[{"BearerAuth":[]}],"x-casbin-resource":false}},
        "/items":{"parameters":[],"get":{"security":[{"BearerAuth":[]}],"x-casbin-resource":true,"tags":["items"],"summary":"List items"}},
        "/optional":{"get":{"security":[{"BearerAuth":[]},{}]}}
    }}`)
	resources, err := readSwaggerResources(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(resources) != 1 || resources[0].Key != "GET /v1/sys/items" || resources[0].Description != "List items" || resources[0].ApiGroup != "items" {
		t.Fatalf("wrong permission resources: %+v", resources)
	}
	embedded, err := readSwaggerResources(generated.Swagger)
	if err != nil || len(embedded) == 0 {
		t.Fatalf("embedded contract must parse: %v", err)
	}
	for _, row := range embedded {
		switch row.Path {
		case "/v1/sys/login", "/v1/sys/randomImage", "/v1/sys/me", "/v1/sys/changePassword", "/v1/sys/logout":
			t.Fatalf("public/personal endpoint became an authorization resource: %s", row.Path)
		}
	}
}

func TestApiSyncSelectedChangesPreserveGrantsAndObsolete(t *testing.T) {
	db := syncDB(t)
	old := model.SysApi{Path: "/v1/sys/old", Method: "GET", ApiGroup: "old", Description: "old description"}
	obsolete := model.SysApi{Path: "/v1/sys/removed", Method: "DELETE", ApiGroup: "legacy"}
	for _, record := range []*model.SysApi{&old, &obsolete} {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	rule := gormadapter.CasbinRule{Ptype: "p", V0: "88", V1: old.Path, V2: old.Method}
	if err := db.Create(&rule).Error; err != nil {
		t.Fatal(err)
	}
	resources := []syncResource{
		{Key: "GET /v1/sys/new", Path: "/v1/sys/new", Method: "GET", ApiGroup: "new", Description: "new resource"},
		{Key: "GET /v1/sys/old", Path: old.Path, Method: old.Method, ApiGroup: "new", Description: "updated description"},
		{Key: "POST /v1/sys/unselected", Path: "/v1/sys/unselected", Method: "POST", ApiGroup: "new"},
	}
	preview, err := previewApiSync(resources, []model.SysApi{old, obsolete})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Added) != 2 || len(preview.Changed) != 1 || len(preview.Obsolete) != 1 {
		t.Fatalf("wrong diff: %+v", preview)
	}
	var applied *pb.ApplyApiSyncResponse
	err = db.Transaction(func(tx *gorm.DB) error {
		var err error
		applied, err = applyApiSync(tx, resources, &pb.ApplyApiSyncRequest{Version: preview.Version, Keys: []string{resources[0].Key, resources[1].Key}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if applied.Added != 1 || applied.Updated != 1 {
		t.Fatalf("wrong counts: %+v", applied)
	}
	var updated model.SysApi
	if err := db.First(&updated, old.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Path != old.Path || updated.Method != old.Method || updated.Description != "updated description" {
		t.Fatalf("resource identity changed: %+v", updated)
	}
	var count int64
	db.Model(&gormadapter.CasbinRule{}).Where("id=? AND v1=? AND v2=?", rule.ID, old.Path, old.Method).Count(&count)
	if count != 1 {
		t.Fatal("sync altered existing permission")
	}
	db.Model(&model.SysApi{}).Where("id=?", obsolete.ID).Count(&count)
	if count != 1 {
		t.Fatal("sync removed obsolete resource")
	}
	db.Model(&model.SysApi{}).Where("path=?", "/v1/sys/unselected").Count(&count)
	if count != 0 {
		t.Fatal("sync applied an unselected resource")
	}
	db.Model(&gormadapter.CasbinRule{}).Count(&count)
	if count != 1 {
		t.Fatal("sync granted a new permission")
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		_, err := applyApiSync(tx, resources, &pb.ApplyApiSyncRequest{Version: preview.Version, Keys: []string{resources[0].Key}})
		return err
	}); err == nil {
		t.Fatal("stale preview must be rejected")
	}
}

func TestApiSyncRejectsUntrustedOrObsoleteSelectionAtomically(t *testing.T) {
	for _, bad := range []string{"DELETE /v1/sys/obsolete", "GET /v1/sys/not-in-swagger", "GET /v1/sys/new"} {
		t.Run(strings.ReplaceAll(bad, "/", "_"), func(t *testing.T) {
			db := syncDB(t)
			obsolete := model.SysApi{Path: "/v1/sys/obsolete", Method: "DELETE"}
			if err := db.Create(&obsolete).Error; err != nil {
				t.Fatal(err)
			}
			resources := []syncResource{
				{Key: "GET /v1/sys/new", Path: "/v1/sys/new", Method: "GET"},
				{Key: "PUT /v1/sys/other", Path: "/v1/sys/other", Method: "PUT"},
			}
			preview, err := previewApiSync(resources, []model.SysApi{obsolete})
			if err != nil {
				t.Fatal(err)
			}
			err = db.Transaction(func(tx *gorm.DB) error {
				_, err := applyApiSync(tx, resources, &pb.ApplyApiSyncRequest{Version: preview.Version, Keys: []string{resources[0].Key, bad}})
				return err
			})
			if err == nil {
				t.Fatal("invalid selection accepted")
			}
			var count int64
			db.Model(&model.SysApi{}).Count(&count)
			if count != 1 {
				t.Fatal("invalid request partially applied")
			}
		})
	}
}

func TestApiSyncPreviewVersionIsStableAndDetectsChanges(t *testing.T) {
	one := model.SysApi{Path: "/v1/sys/one", Method: "GET", ApiGroup: "old"}
	two := model.SysApi{Path: "/v1/sys/two", Method: "GET"}
	resources := []syncResource{{Key: "GET /v1/sys/one", Path: one.Path, Method: one.Method, ApiGroup: "new"}}
	a, _ := previewApiSync(resources, []model.SysApi{one, two})
	b, _ := previewApiSync(resources, []model.SysApi{two, one})
	if a.Version != b.Version {
		t.Fatal("DB row order affected preview version")
	}
	one.Description = "concurrent edit"
	c, _ := previewApiSync(resources, []model.SysApi{one, two})
	if a.Version == c.Version {
		t.Fatal("metadata edit did not invalidate preview")
	}
	if _, err := previewApiSync(resources, []model.SysApi{one, one}); err == nil {
		t.Fatal("duplicate existing API must not be silently merged")
	}
}
