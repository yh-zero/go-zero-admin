package accessutil_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	apilogic "go-zero-admin/application/applet/rpc/internal/logic/api"
	authoritylogic "go-zero-admin/application/applet/rpc/internal/logic/authority"
	casbinlogic "go-zero-admin/application/applet/rpc/internal/logic/casbin"
	dictionarylogic "go-zero-admin/application/applet/rpc/internal/logic/dictionary"
	menulogic "go-zero-admin/application/applet/rpc/internal/logic/menu"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	base "go-zero-admin/pkg/model"
	"go-zero-admin/pkg/orm"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func service(t *testing.T) *svc.ServiceContext {
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
	err = db.AutoMigrate(&model.SysBaseMenu{}, &model.SysBaseMenuBtn{}, &model.SysBaseMenuParameter{}, &model.SysAuthority{}, &model.SysUser{}, &model.SysAuthorityMenu{}, &model.SysAuthorityBtn{}, &model.SysUserAuthority{}, &model.SysApi{}, &model.SysDictionary{}, &model.SysDictionaryInfo{}, &gormadapter.CasbinRule{})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		t.Fatal(err)
	}
	m, err := casbinmodel.NewModelFromString("[request_definition]\nr = sub, obj, act\n[policy_definition]\np = sub, obj, act\n[policy_effect]\ne = some(where (p.eft == allow))\n[matchers]\nm = r.sub == p.sub && r.obj == p.obj && r.act == p.act")
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := casbin.NewSyncedCachedEnforcer(m, adapter)
	if err != nil {
		t.Fatal(err)
	}
	s := &svc.ServiceContext{DB: &orm.DB{DB: db}, Casbin: enforcer}
	root := int64(0)
	must(t, db.Create(&model.SysAuthority{AuthorityId: 88, AuthorityName: "test role", ParentId: &root, DefaultRouter: "index"}).Error)
	must(t, db.Create(&model.SysBaseMenu{MODEL_BASE: base.MODEL_BASE{ID: 7}, Name: "index", Path: "index", Component: "views/index.vue", Meta: model.Meta{Title: "首页"}}).Error)
	return s
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func mustFail(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected failure")
	}
}
func menu(t *testing.T, s *svc.ServiceContext, name string, parent int64) *pb.SysBaseMenu {
	t.Helper()
	m := &pb.SysBaseMenu{Name: name, Path: name, ParentId: parent, Component: "views/test.vue", Meta: &pb.Meta{Title: name}}
	_, err := menulogic.NewAddMenuBaseLogic(context.Background(), s).AddMenuBase(&pb.AddMenuBaseRequest{SysBaseMenu: m})
	must(t, err)
	var stored model.SysBaseMenu
	must(t, s.DB.First(&stored, "name = ?", name).Error)
	m.ID = stored.ID
	return m
}

func TestMenuAuthorizationTransactionsAndButtons(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	parent := menu(t, s, "parent", 0)
	child := menu(t, s, "child", parent.ID)
	child.MenuBtn = []*pb.SysBaseMenuBtn{{Name: "create", Desc: "新增"}}
	_, err := menulogic.NewUpdateBaseMenuLogic(ctx, s).UpdateBaseMenu(&pb.UpdateBaseMenuRequest{SysBaseMenu: child})
	must(t, err)
	var button model.SysBaseMenuBtn
	must(t, s.DB.First(&button, "sys_base_menu_id = ?", child.ID).Error)
	grant := authoritylogic.NewAddAuthorityMenuLogic(ctx, s)
	_, err = grant.AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, MenuIds: strconv.FormatInt(child.ID, 10)})
	must(t, err)
	result, err := menulogic.NewGetMenuAuthorityLogic(ctx, s).GetMenuAuthority(&pb.GetMenuAuthorityRequest{AuthorityId: 88})
	must(t, err)
	if len(result.SysMenuList) != 2 {
		t.Fatalf("parents not included: %+v", result)
	}
	found := false
	for _, m := range result.SysMenuList {
		if m.ID == child.ID {
			found = len(m.SysBaseMenu.MenuBtn) == 1 && m.SysBaseMenu.Name == "child"
		}
	}
	if !found {
		t.Fatal("target role response lost menu fields/button definitions")
	}
	_, err = menulogic.NewUpdateAuthorityButtonsLogic(ctx, s).UpdateAuthorityButtons(&pb.UpdateAuthorityButtonsRequest{AuthorityId: 88, MenuBtnIds: []int64{button.ID}})
	must(t, err)
	child.MenuBtn[0].ID = button.ID
	child.Meta.Title = "changed"
	_, err = menulogic.NewUpdateBaseMenuLogic(ctx, s).UpdateBaseMenu(&pb.UpdateBaseMenuRequest{SysBaseMenu: child})
	must(t, err)
	buttons, err := menulogic.NewGetAuthorityButtonsLogic(ctx, s).GetAuthorityButtons(&pb.GetAuthorityButtonsRequest{AuthorityId: 88})
	must(t, err)
	if len(buttons.MenuBtnIds) != 1 || buttons.MenuBtnIds[0] != button.ID {
		t.Fatal("menu edit dropped retained button grants")
	}
	parent.ParentId = child.ID
	_, err = menulogic.NewUpdateBaseMenuLogic(ctx, s).UpdateBaseMenu(&pb.UpdateBaseMenuRequest{SysBaseMenu: parent})
	mustFail(t, err)
	_, err = menulogic.NewDeleteBaseMenuLogic(ctx, s).DeleteBaseMenu(&pb.DeleteBaseMenuRequest{ID: child.ID})
	mustFail(t, err)
	_, err = grant.AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, MenuIds: "999999"})
	mustFail(t, err)
	// Fail after the replacement DELETE to prove transaction rollback keeps old grants.
	must(t, s.DB.Callback().Create().Before("gorm:create").Register("test_menu_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_authority_menus" {
			tx.AddError(errors.New("injected insert failure"))
		}
	}))
	_, err = grant.AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, MenuIds: "7"})
	mustFail(t, err)
	must(t, s.DB.Callback().Create().Remove("test_menu_failure"))
	result, err = menulogic.NewGetMenuAuthorityLogic(ctx, s).GetMenuAuthority(&pb.GetMenuAuthorityRequest{AuthorityId: 88})
	must(t, err)
	if len(result.SysMenuList) != 2 {
		t.Fatal("failed replacement lost old menu grants")
	}
	_, err = grant.AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, MenuIds: ""})
	must(t, err)
	buttons, err = menulogic.NewGetAuthorityButtonsLogic(ctx, s).GetAuthorityButtons(&pb.GetAuthorityButtonsRequest{AuthorityId: 88})
	must(t, err)
	if len(buttons.MenuBtnIds) != 0 {
		t.Fatal("clearing menus left button grants")
	}
	_, err = menulogic.NewDeleteBaseMenuLogic(ctx, s).DeleteBaseMenu(&pb.DeleteBaseMenuRequest{ID: child.ID})
	must(t, err)
}

func TestRoleDefaultsHierarchyAndRootMove(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	create := authoritylogic.NewCreateAuthorityLogic(ctx, s)
	_, err := create.CreateAuthority(&pb.CreateAuthorityRequest{SysAuthority: &pb.SysAuthority{AuthorityId: 99, AuthorityName: "child role", ParentId: 88}})
	must(t, err)
	var grant model.SysAuthorityMenu
	must(t, s.DB.First(&grant, "sys_authority_authority_id = ?", 99).Error)
	if grant.MenuId != "7" {
		t.Fatalf("homepage must be found by name, got %s", grant.MenuId)
	}
	allowed, err := s.Casbin.Enforce("99", "/v1/sys/menu/getMenu", "GET")
	must(t, err)
	if !allowed {
		t.Fatal("default GET permission not effective")
	}
	_, err = authoritylogic.NewUpdateAuthorityLogic(ctx, s).UpdateAuthority(&pb.UpdateAuthorityRequest{SysAuthority: &pb.SysAuthority{AuthorityId: 88, AuthorityName: "parent", ParentId: 99}})
	mustFail(t, err)
	_, err = authoritylogic.NewUpdateAuthorityLogic(ctx, s).UpdateAuthority(&pb.UpdateAuthorityRequest{SysAuthority: &pb.SysAuthority{AuthorityId: 99, AuthorityName: "new root", ParentId: 0, DefaultRouter: "index"}})
	must(t, err)
	var role model.SysAuthority
	must(t, s.DB.First(&role, "authority_id = ?", 99).Error)
	if role.ParentId == nil || *role.ParentId != 0 {
		t.Fatal("parentId=0 was ignored")
	}
	_, err = authoritylogic.NewDeleteAuthorityLogic(ctx, s).DeleteAuthority(&pb.DeleteAuthorityRequest{ID: 99})
	must(t, err)
	allowed, err = s.Casbin.Enforce("99", "/v1/sys/menu/getMenu", "GET")
	must(t, err)
	if allowed {
		t.Fatal("deleted role kept cached API permission")
	}
}

func TestAPIPoliciesValidateRollbackAndReload(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	_, err := apilogic.NewCreateApiLogic(ctx, s).CreateApi(&pb.CreateApiRequest{SysApi: &pb.SysApi{Path: "/v1/test/resource", Method: "get", Description: "test"}})
	must(t, err)
	var api model.SysApi
	must(t, s.DB.First(&api).Error)
	if api.Method != "GET" {
		t.Fatal("HTTP method not normalized")
	}
	save := casbinlogic.NewUpdateCasbinDataByApiIdsLogic(ctx, s)
	_, err = save.UpdateCasbinDataByApiIds(&pb.UpdateCasbinDataByApiIdsRequest{AuthorityId: 88, ApiIds: []int64{api.ID}})
	must(t, err)
	allowed, err := s.Casbin.Enforce("88", api.Path, "GET")
	must(t, err)
	if !allowed {
		t.Fatal("saved policy not effective")
	}
	_, err = save.UpdateCasbinDataByApiIds(&pb.UpdateCasbinDataByApiIdsRequest{AuthorityId: 88, ApiIds: []int64{api.ID, 99999}})
	mustFail(t, err)
	allowed, err = s.Casbin.Enforce("88", api.Path, "GET")
	must(t, err)
	if !allowed {
		t.Fatal("invalid ID wiped prior grants")
	}
	_, err = apilogic.NewUpdateApiLogic(ctx, s).UpdateApi(&pb.UpdateApiRequest{SysApi: &pb.SysApi{ID: api.ID, Path: "/v1/test/renamed", Method: "PUT"}})
	must(t, err)
	allowed, err = s.Casbin.Enforce("88", api.Path, "GET")
	must(t, err)
	if allowed {
		t.Fatal("old cached API permission survived rename")
	}
	allowed, err = s.Casbin.Enforce("88", "/v1/test/renamed", "PUT")
	must(t, err)
	if !allowed {
		t.Fatal("renamed policy not effective")
	}
	_, err = casbinlogic.NewUpdateCasbinDataLogic(ctx, s).UpdateCasbinData(&pb.UpdateCasbinDataRequest{AuthorityId: 88, CasbinInfoList: []*pb.CasbinInfo{{Path: "/v1/not-registered", Method: "GET"}}})
	mustFail(t, err)
	_, err = apilogic.NewGetApiListLogic(ctx, s).GetApiList(&pb.GetApiListRequest{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}, OrderKey: "DROP TABLE"})
	mustFail(t, err)
	_, err = apilogic.NewDeleteApisByIdsLogic(ctx, s).DeleteApisByIds(&pb.DeleteApisByIdsRequest{Ids: []int64{api.ID, 99999}})
	mustFail(t, err)
	_, err = apilogic.NewDeleteApiLogic(ctx, s).DeleteApi(&pb.DeleteApiRequest{SysApi: &pb.SysApi{ID: api.ID}})
	must(t, err)
	allowed, err = s.Casbin.Enforce("88", "/v1/test/renamed", "PUT")
	must(t, err)
	if allowed {
		t.Fatal("deleted API kept cached policy")
	}
}

func TestDictionaryZeroValuesFilteringAndDependencies(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	create := dictionarylogic.NewCreateSysDictionaryLogic(ctx, s)
	_, err := create.CreateSysDictionary(&pb.CreateSysDictionaryRequest{SysDictionary: &pb.SysDictionary{Name: "测试", Type: "test", Status: 1, Desc: "description"}})
	must(t, err)
	_, err = create.CreateSysDictionary(&pb.CreateSysDictionaryRequest{SysDictionary: &pb.SysDictionary{Name: "重复", Type: "test", Status: 1}})
	mustFail(t, err)
	var dictionary model.SysDictionary
	must(t, s.DB.First(&dictionary).Error)
	_, err = dictionarylogic.NewCreateSysDictionaryInfoLogic(ctx, s).CreateSysDictionaryInfo(&pb.CreateSysDictionaryInfoRequest{SysDictionaryInfo: &pb.SysDictionaryInfo{Label: "item", Value: 5, Sort: 4, Extend: "old", Status: 1, SysDictionaryID: dictionary.ID}})
	must(t, err)
	var item model.SysDictionaryInfo
	must(t, s.DB.First(&item).Error)
	_, err = dictionarylogic.NewUpdateSysDictionaryInfoLogic(ctx, s).UpdateSysDictionaryInfo(&pb.UpdateSysDictionaryInfoRequest{SysDictionaryInfo: &pb.SysDictionaryInfo{ID: item.ID, Label: "item", Value: 0, Sort: 0, Extend: "", Status: 2, SysDictionaryID: dictionary.ID}})
	must(t, err)
	must(t, s.DB.First(&item, item.ID).Error)
	if item.Value != 0 || item.Sort != 0 || item.Extend != "" || item.Status != 2 {
		t.Fatalf("zero values not saved: %+v", item)
	}
	list, err := dictionarylogic.NewGetSysDictionaryInfoListLogic(ctx, s).GetSysDictionaryInfoList(&pb.GetSysDictionaryInfoListRequest{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}, HasValue: true, SysDictionaryInfo: &pb.SysDictionaryInfo{SysDictionaryID: dictionary.ID, Value: 0}})
	must(t, err)
	if list.Total != 1 {
		t.Fatal("zero-value filter or disabled management item incorrect")
	}
	_, err = dictionarylogic.NewDeleteSysDictionaryLogic(ctx, s).DeleteSysDictionary(&pb.DeleteSysDictionaryRequest{ID: dictionary.ID})
	mustFail(t, err)
	_, err = dictionarylogic.NewDeleteSysDictionaryInfoLogic(ctx, s).DeleteSysDictionaryInfo(&pb.DeleteSysDictionaryInfoRequest{ID: item.ID})
	must(t, err)
	_, err = dictionarylogic.NewDeleteSysDictionaryLogic(ctx, s).DeleteSysDictionary(&pb.DeleteSysDictionaryRequest{ID: dictionary.ID})
	must(t, err)
}

func TestParentValidation(t *testing.T) {
	must(t, accessutil.ValidateParent(3, 0, map[int64]int64{1: 0, 2: 1}))
	mustFail(t, accessutil.ValidateParent(1, 2, map[int64]int64{1: 0, 2: 1}))
	mustFail(t, accessutil.ValidateParent(3, 9, map[int64]int64{1: 0}))
	mustFail(t, accessutil.ValidateParent(3, 1, map[int64]int64{1: 2, 2: 1}))
}

func TestMenusRejectReservedAndResolvedPathConflicts(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	parent := menu(t, s, "admin", 0)
	menu(t, s, "users", parent.ID)
	for _, invalid := range []*pb.SysBaseMenu{
		{Name: "Root", Path: "reserved"},
		{Name: "auth-page", Path: "/auth/login"},
		{Name: "session-page", Path: "/_session/home"},
		{Name: "traversal", Path: "../admin"},
		{Name: "absolute", Path: "/admin/users"},
		{Name: "parameter", Path: "users/:id"},
	} {
		invalid.Meta = &pb.Meta{Title: invalid.Name}
		_, err := menulogic.NewAddMenuBaseLogic(ctx, s).AddMenuBase(&pb.AddMenuBaseRequest{SysBaseMenu: invalid})
		mustFail(t, err)
	}
}

func TestFailedPolicyInsertKeepsExistingRoleGrants(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	api := model.SysApi{Path: "/v1/test/keep", Method: "GET"}
	must(t, s.DB.Create(&api).Error)
	save := casbinlogic.NewUpdateCasbinDataByApiIdsLogic(ctx, s)
	_, err := save.UpdateCasbinDataByApiIds(&pb.UpdateCasbinDataByApiIdsRequest{AuthorityId: 88, ApiIds: []int64{api.ID}})
	must(t, err)
	must(t, s.DB.Callback().Create().Before("gorm:create").Register("test_policy_failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "casbin_rule" {
			tx.AddError(errors.New("injected policy insert failure"))
		}
	}))
	_, err = save.UpdateCasbinDataByApiIds(&pb.UpdateCasbinDataByApiIdsRequest{AuthorityId: 88, ApiIds: []int64{api.ID}})
	mustFail(t, err)
	must(t, s.DB.Callback().Create().Remove("test_policy_failure"))
	var count int64
	must(t, s.DB.Model(&gormadapter.CasbinRule{}).Where("ptype = ? AND v0 = ?", "p", "88").Count(&count).Error)
	if count != 1 {
		t.Fatal("failed transaction lost persisted rules")
	}
	allowed, err := s.Casbin.Enforce("88", api.Path, "GET")
	must(t, err)
	if !allowed {
		t.Fatal("failed transaction changed effective permission")
	}
}
