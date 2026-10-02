package accessutil_test

import (
	"context"
	"testing"
	"time"

	menulogic "go-zero-admin/application/applet/rpc/internal/logic/menu"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
)

func TestMenuRenameUpdatesDefaultRoutersAtomically(t *testing.T) {
	s := service(t)
	deleted := time.Now()
	must(t, s.DB.Create(&model.SysAuthority{AuthorityId: 89, AuthorityName: "other", DefaultRouter: "different"}).Error)
	must(t, s.DB.Create(&model.SysAuthority{AuthorityId: 90, AuthorityName: "deleted", DefaultRouter: "index", DeletedAt: &deleted}).Error)
	request := &pb.UpdateBaseMenuRequest{SysBaseMenu: &pb.SysBaseMenu{ID: 7, Name: "home", Path: "index", Component: "views/index.vue", Meta: &pb.Meta{Title: "Home"}}}
	logic := menulogic.NewUpdateBaseMenuLogic(context.Background(), s)
	_, err := logic.UpdateBaseMenu(request)
	must(t, err)
	for id, want := range map[int64]string{88: "home", 89: "different", 90: "index"} {
		var role model.SysAuthority
		must(t, s.DB.Where("authority_id = ?", id).First(&role).Error)
		if role.DefaultRouter != want {
			t.Fatalf("role %d default route = %q, want %q", id, role.DefaultRouter, want)
		}
	}
	// A failure after the role update must roll back both tables.
	must(t, s.DB.Exec("CREATE TRIGGER reject_menu_parameter BEFORE INSERT ON sys_base_menu_parameters WHEN NEW.key = 'reject' BEGIN SELECT RAISE(FAIL, 'rejected parameter'); END").Error)
	request.SysBaseMenu.Name = "failed_home"
	request.SysBaseMenu.Parameters = []*pb.SysBaseMenuParameter{{Type: "query", Key: "reject", Value: "test"}}
	_, err = logic.UpdateBaseMenu(request)
	mustFail(t, err)
	var role model.SysAuthority
	var menu model.SysBaseMenu
	must(t, s.DB.Where("authority_id = ?", 88).First(&role).Error)
	must(t, s.DB.First(&menu, 7).Error)
	if role.DefaultRouter != "home" || menu.Name != "home" {
		t.Fatal("failed relation save did not roll back menu and default router together")
	}
}
