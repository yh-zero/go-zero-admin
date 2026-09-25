package accessutil_test

import (
	"context"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	apilogic "go-zero-admin/application/applet/rpc/internal/logic/api"
	authoritylogic "go-zero-admin/application/applet/rpc/internal/logic/authority"
	casbinlogic "go-zero-admin/application/applet/rpc/internal/logic/casbin"
	menulogic "go-zero-admin/application/applet/rpc/internal/logic/menu"
	userlogic "go-zero-admin/application/applet/rpc/internal/logic/user"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	base "go-zero-admin/pkg/model"
	"sync"
	"testing"
)

func TestLegacyMissingRecoveryPoliciesDoNotBlockUnrelatedWork(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	must(t, s.DB.Create(&model.SysAuthority{AuthorityId: 1, AuthorityName: "legacy admin"}).Error)
	_, err := apilogic.NewCreateApiLogic(ctx, s).CreateApi(&pb.CreateApiRequest{SysApi: &pb.SysApi{Path: "/v1/sys/newResource", Method: "GET"}})
	must(t, err)
	_, err = casbinlogic.NewUpdateCasbinDataLogic(ctx, s).UpdateCasbinData(&pb.UpdateCasbinDataRequest{AuthorityId: 88})
	must(t, err)
	// A partial legacy recovery grant may improve, but an existing grant cannot be lost.
	must(t, s.DB.Create(&model.SysApi{Path: "/v1/sys/menu/getMenu", Method: "GET"}).Error)
	must(t, s.DB.Create(&gormadapter.CasbinRule{Ptype: "p", V0: "1", V1: "/v1/sys/menu/getMenu", V2: "GET"}).Error)
	_, err = casbinlogic.NewUpdateCasbinDataLogic(ctx, s).UpdateCasbinData(&pb.UpdateCasbinDataRequest{AuthorityId: 1})
	mustFail(t, err)
	_, err = authoritylogic.NewDeleteAuthorityLogic(ctx, s).DeleteAuthority(&pb.DeleteAuthorityRequest{ID: 1})
	mustFail(t, err)
}

func TestAdministratorRecoveryMenuAndButtonsArePreserved(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	seedRecoveryRole(t, s)
	root := model.SysBaseMenu{MODEL_BASE: base.MODEL_BASE{ID: 10}, Name: "admin", Path: "admin", Component: "views/superAdmin/index.vue"}
	page := model.SysBaseMenu{MODEL_BASE: base.MODEL_BASE{ID: 11}, ParentId: 10, Name: "authority", Path: "authority", Component: "views/superAdmin/authority/authority.vue"}
	must(t, s.DB.Create(&root).Error)
	must(t, s.DB.Create(&page).Error)
	must(t, s.DB.Create(&model.SysAuthorityMenu{AuthorityId: "1", MenuId: "10"}).Error)
	must(t, s.DB.Create(&model.SysAuthorityMenu{AuthorityId: "1", MenuId: "11"}).Error)
	var buttonIDs []int64
	for _, name := range []string{"menus", "buttons", "apis"} {
		button := model.SysBaseMenuBtn{SysBaseMenuID: 11, Name: name, Desc: name}
		must(t, s.DB.Create(&button).Error)
		buttonIDs = append(buttonIDs, button.ID)
		must(t, s.DB.Create(&model.SysAuthorityBtn{AuthorityId: 1, SysMenuID: 11, SysBaseMenuBtnID: button.ID}).Error)
	}
	_, err := authoritylogic.NewAddAuthorityMenuLogic(ctx, s).AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 1, MenuIds: ""})
	mustFail(t, err)
	_, err = menulogic.NewUpdateAuthorityButtonsLogic(ctx, s).UpdateAuthorityButtons(&pb.UpdateAuthorityButtonsRequest{AuthorityId: 1, MenuBtnIds: buttonIDs[:2]})
	mustFail(t, err)
	_, err = menulogic.NewUpdateBaseMenuLogic(ctx, s).UpdateBaseMenu(&pb.UpdateBaseMenuRequest{SysBaseMenu: &pb.SysBaseMenu{ID: 10, Name: "admin", Path: "admin", Component: root.Component, Hidden: true, Meta: &pb.Meta{Title: "Admin"}}})
	mustFail(t, err)
	_, err = menulogic.NewUpdateBaseMenuLogic(ctx, s).UpdateBaseMenu(&pb.UpdateBaseMenuRequest{SysBaseMenu: &pb.SysBaseMenu{ID: 11, ParentId: 10, Name: "authority", Path: "authority", Component: "views/index.vue", Meta: &pb.Meta{Title: "Roles"}}})
	mustFail(t, err)
	// Keeping the component but deleting its permission button definitions also fails.
	_, err = menulogic.NewUpdateBaseMenuLogic(ctx, s).UpdateBaseMenu(&pb.UpdateBaseMenuRequest{SysBaseMenu: &pb.SysBaseMenu{ID: 11, ParentId: 10, Name: "authority", Path: "authority", Component: page.Component, Meta: &pb.Meta{Title: "Roles"}}})
	mustFail(t, err)
	_, err = authoritylogic.NewAddAuthorityMenuLogic(ctx, s).AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, MenuIds: ""})
	must(t, err)
	// No special restrictions on preserving all recovery capabilities unchanged.
	_, err = authoritylogic.NewAddAuthorityMenuLogic(ctx, s).AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: accessutil.AdminAuthorityID, MenuIds: "11"})
	must(t, err)
	_, err = menulogic.NewUpdateAuthorityButtonsLogic(ctx, s).UpdateAuthorityButtons(&pb.UpdateAuthorityButtonsRequest{AuthorityId: 1, MenuBtnIds: buttonIDs})
	must(t, err)
	var count int64
	must(t, s.DB.Model(&model.SysAuthorityBtn{}).Where("authority_id = ?", 1).Count(&count).Error)
	if count != 3 {
		t.Fatal("recovery button changes did not roll back")
	}
}

func TestRegisterAndDeleteRoleCannotCreateOrphanMembership(t *testing.T) {
	s := service(t)
	seedRecoveryRole(t, s)
	ctx := context.Background()
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		_, _ = userlogic.NewRegisterLogic(ctx, s).Register(&pb.RegisterRequest{UserInfo: &pb.UserInfo{Username: "role_race", Password: "before123", AuthorityId: 88, Enable: 1}, AuthorityIds: []int64{88}})
	}()
	go func() {
		defer wait.Done()
		_, _ = authoritylogic.NewDeleteAuthorityLogic(ctx, s).DeleteAuthority(&pb.DeleteAuthorityRequest{ID: 88})
	}()
	wait.Wait()
	var users, roles int64
	must(t, s.DB.Model(&model.SysUser{}).Where("username = ?", "role_race").Count(&users).Error)
	must(t, s.DB.Model(&model.SysAuthority{}).Where("authority_id = ? AND deleted_at IS NULL", 88).Count(&roles).Error)
	if users == 1 && roles != 1 {
		t.Fatal("user committed with a deleted default role")
	}
	if users == 0 && roles != 0 {
		t.Fatal("both operations unexpectedly failed")
	}
}
