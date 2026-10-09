package accessutil_test

import (
	"context"
	"fmt"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	authoritylogic "go-zero-admin/application/applet/rpc/internal/logic/authority"
	casbinlogic "go-zero-admin/application/applet/rpc/internal/logic/casbin"
	menulogic "go-zero-admin/application/applet/rpc/internal/logic/menu"
	permissionlogic "go-zero-admin/application/applet/rpc/internal/logic/permission"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	base "go-zero-admin/pkg/model"
	"gorm.io/gorm"
	"testing"
)

func TestPermissionMissingRevisionRejected(t *testing.T) {
	s := service(t)
	_, err := authoritylogic.NewAddAuthorityMenuLogic(context.Background(), s).AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, MenuIds: "7"})
	if err == nil {
		t.Fatal("missing expectedRevision must reject a legacy overwrite")
	}
}

func TestPermissionOldRevisionCannotRestoreRevokedMenuOrButtons(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	old := currentPermissionRevision(t, s)
	_, err := authoritylogic.NewAddAuthorityMenuLogic(ctx, s).AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, MenuIds: "7", ExpectedRevision: old})
	must(t, err)
	_, err = authoritylogic.NewAddAuthorityMenuLogic(ctx, s).AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, MenuIds: "", ExpectedRevision: currentPermissionRevision(t, s)})
	must(t, err)
	latest := currentPermissionRevision(t, s)
	for _, save := range []func() error{
		func() error {
			_, e := authoritylogic.NewAddAuthorityMenuLogic(ctx, s).AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, MenuIds: "7", ExpectedRevision: old})
			return e
		},
		func() error {
			_, e := menulogic.NewUpdateAuthorityButtonsLogic(ctx, s).UpdateAuthorityButtons(&pb.UpdateAuthorityButtonsRequest{AuthorityId: 88, ExpectedRevision: old})
			return e
		},
		func() error {
			_, e := casbinlogic.NewUpdateCasbinDataLogic(ctx, s).UpdateCasbinData(&pb.UpdateCasbinDataRequest{AuthorityId: 88, ExpectedRevision: old})
			return e
		},
		func() error {
			_, e := casbinlogic.NewUpdateCasbinDataByApiIdsLogic(ctx, s).UpdateCasbinDataByApiIds(&pb.UpdateCasbinDataByApiIdsRequest{AuthorityId: 88, ExpectedRevision: old})
			return e
		},
	} {
		mustFail(t, save())
		if currentPermissionRevision(t, s) != latest {
			t.Fatal("failed stale save advanced revision")
		}
	}
	edit, err := permissionlogic.NewGetPermissionEditLogic(ctx, s).GetPermissionEdit(&pb.GetPermissionEditRequest{AuthorityId: 88, Kind: "menu"})
	must(t, err)
	if len(edit.MenuIds) != 0 || edit.Revision != latest || len(edit.Menus) != 1 {
		t.Fatalf("invalid edit snapshot %+v", edit)
	}
}

func TestMenuMovePreviewRejectsNewAffectedRole(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	a := menu(t, s, "a", 0)
	b := menu(t, s, "b", 0)
	c := menu(t, s, "c", a.ID)
	preview, err := menulogic.NewPreviewMenuMoveLogic(ctx, s).PreviewMenuMove(&pb.PreviewMenuMoveRequest{ID: c.ID, ParentId: b.ID})
	must(t, err)
	_, err = authoritylogic.NewAddAuthorityMenuLogic(ctx, s).AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, MenuIds: fmt.Sprint(c.ID), ExpectedRevision: currentPermissionRevision(t, s)})
	must(t, err)
	c.ParentId = b.ID
	_, err = menulogic.NewUpdateBaseMenuLogic(ctx, s).UpdateBaseMenu(&pb.UpdateBaseMenuRequest{SysBaseMenu: c, MovePreviewVersion: preview.Version})
	mustFail(t, err)
	var stored model.SysBaseMenu
	must(t, s.DB.First(&stored, c.ID).Error)
	if stored.ParentId != a.ID {
		t.Fatal("stale preview changed menu parent")
	}
}

func TestPermissionSnapshotOnlyCurrentActorAndFingerprintIgnoresOtherRole(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	must(t, s.DB.Omit("Authority", "Authorities").Create(&model.SysUser{MODEL_BASE: base.MODEL_BASE{ID: 20}, Username: "snapshot", AuthorityId: 88, Enable: 1, SessionVersion: 1}).Error)
	must(t, s.DB.Create(&model.SysUserAuthority{SysUserId: 20, SysAuthorityAuthorityId: 88}).Error)
	_, err := authoritylogic.NewAddAuthorityMenuLogic(ctx, s).AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, MenuIds: "7", ExpectedRevision: currentPermissionRevision(t, s)})
	must(t, err)
	logic := permissionlogic.NewGetPermissionSnapshotLogic(ctx, s)
	request := &pb.GetPermissionSnapshotRequest{Actor: &pb.SessionRequest{UserID: 20, AuthorityId: 88, SessionVersion: 1}}
	before, err := logic.GetPermissionSnapshot(request)
	must(t, err)
	must(t, accessutil.PermissionTransaction(s.DB.DB, func(tx *gorm.DB) error {
		return tx.Create(&model.SysAuthority{AuthorityId: 99, AuthorityName: "other"}).Error
	}))
	after, err := logic.GetPermissionSnapshot(request)
	must(t, err)
	if before.Revision == after.Revision || before.Fingerprint != after.Fingerprint || len(after.Menus) != 1 {
		t.Fatalf("unexpected snapshot refresh: %+v %+v", before, after)
	}
	request.Actor.AuthorityId = 99
	_, err = logic.GetPermissionSnapshot(request)
	mustFail(t, err)
}

func TestPermissionKeyRemainsAfterButtonAndMenuRename(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	request := &pb.UpdateBaseMenuRequest{SysBaseMenu: &pb.SysBaseMenu{ID: 7, Name: "index", Path: "index", Meta: &pb.Meta{Title: "home"}, MenuBtn: []*pb.SysBaseMenuBtn{{Name: "create"}}}}
	_, err := menulogic.NewUpdateBaseMenuLogic(ctx, s).UpdateBaseMenu(request)
	must(t, err)
	var button model.SysBaseMenuBtn
	must(t, s.DB.First(&button, "sys_base_menu_id = ?", 7).Error)
	if button.PermissionKey == "" {
		t.Fatal("new button must have stable permission key")
	}
	request.SysBaseMenu.Name = "renamed"
	request.SysBaseMenu.MenuBtn[0].ID = button.ID
	request.SysBaseMenu.MenuBtn[0].Name = "renamed"
	_, err = menulogic.NewUpdateBaseMenuLogic(ctx, s).UpdateBaseMenu(request)
	must(t, err)
	var after model.SysBaseMenuBtn
	must(t, s.DB.First(&after, button.ID).Error)
	if after.PermissionKey != button.PermissionKey {
		t.Fatal("rename changed stable permission key")
	}
}
func TestMenuMoveRequiresBoundPreviewAndAddsAncestors(t *testing.T) {
	s := service(t)
	ctx := context.Background()
	a := menu(t, s, "a", 0)
	b := menu(t, s, "b", 0)
	c := menu(t, s, "c", a.ID)
	must(t, s.DB.Create(&model.SysAuthorityMenu{AuthorityId: "88", MenuId: fmt.Sprint(c.ID)}).Error)
	c.ParentId = b.ID
	_, err := menulogic.NewUpdateBaseMenuLogic(ctx, s).UpdateBaseMenu(&pb.UpdateBaseMenuRequest{SysBaseMenu: c})
	if err == nil {
		t.Fatal("move without confirmed preview must fail")
	}
	preview, err := menulogic.NewPreviewMenuMoveLogic(ctx, s).PreviewMenuMove(&pb.PreviewMenuMoveRequest{ID: c.ID, ParentId: b.ID})
	must(t, err)
	if len(preview.AddedLinks) != 1 || preview.AddedLinks[0].MenuId != b.ID {
		t.Fatalf("wrong move impact: %+v", preview)
	}
	_, err = menulogic.NewUpdateBaseMenuLogic(ctx, s).UpdateBaseMenu(&pb.UpdateBaseMenuRequest{SysBaseMenu: c, MovePreviewVersion: preview.Version})
	must(t, err)
	var count int64
	must(t, s.DB.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ? AND sys_base_menu_id = ?", 88, b.ID).Count(&count).Error)
	if count != 1 {
		t.Fatal("move must add confirmed ancestors")
	}
}
