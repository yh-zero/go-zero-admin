package menulogic

import (
	"testing"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	base "go-zero-admin/pkg/model"
)

func TestBusinessAccountRouteIsReserved(t *testing.T) {
	// Reserved route names are rejected before database access.
	if err := validateMenu(nil, &pb.SysBaseMenu{Name: "BusinessAccount", Path: "different-path", Meta: &pb.Meta{Title: "冲突菜单"}}); err == nil {
		t.Fatal("personal account route name was accepted")
	}
	for _, path := range []string{"account", "/account", "/account/", "/account/password", "/Account/Password", "/ACCOUNT"} {
		if err := validateResolvedPaths(nil, &pb.SysBaseMenu{Path: path}); err == nil {
			t.Errorf("reserved account path accepted: %s", path)
		}
	}
	for _, path := range []string{"accounting", "/accounting/report", "/business/account"} {
		if err := validateResolvedPaths(nil, &pb.SysBaseMenu{Path: path}); err != nil {
			t.Errorf("unrelated business path rejected: %s: %v", path, err)
		}
	}
	// A child may use an absolute path; checking only the parent is insufficient.
	all := []model.SysBaseMenu{
		{MODEL_BASE: base.MODEL_BASE{ID: 1}, Path: "business"},
		{MODEL_BASE: base.MODEL_BASE{ID: 2}, ParentId: 1, Path: "/account/profile"},
	}
	if err := validateResolvedPaths(all, &pb.SysBaseMenu{ID: 1, Path: "moved-business"}); err == nil {
		t.Fatal("absolute account path in a descendant was accepted")
	}
}
