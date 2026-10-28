package accessutil_test

import (
	"context"
	"fmt"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	authoritylogic "go-zero-admin/application/applet/rpc/internal/logic/authority"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	base "go-zero-admin/pkg/model"
	"go-zero-admin/pkg/orm"
	"go-zero-admin/pkg/testmysql"
	"gorm.io/gorm"
	"testing"
	"time"
)

func mysqlPermissionFixture(t *testing.T) *svc.ServiceContext {
	db := testmysql.Open(t)
	must(t, db.AutoMigrate(&model.SysPermissionVersion{}, &model.SysPermissionChange{}, &model.SysAuthority{}, &model.SysUser{}, &model.SysUserAuthority{}, &model.SysBaseMenu{}, &model.SysBaseMenuBtn{}, &model.SysBaseMenuParameter{}, &model.SysAuthorityMenu{}, &model.SysAuthorityBtn{}, &model.SysRoleDataScope{}, &model.SysRoleScopeDepartment{}, &model.SysApi{}, &model.SysDepartment{}, &gormadapter.CasbinRule{}, &audit.Event{}))
	must(t, db.Create(&model.SysPermissionVersion{ID: 1, Revision: 1}).Error)
	for _, id := range []int64{1, 88, 89} {
		must(t, db.Create(&model.SysAuthority{AuthorityId: id, AuthorityName: "role"}).Error)
	}
	must(t, db.Create(&model.SysBaseMenu{MODEL_BASE: base.MODEL_BASE{ID: 7}, Name: "home", Path: "home"}).Error)
	must(t, db.Create(&model.SysAuthorityMenu{AuthorityId: "88", MenuId: "7"}).Error)
	return &svc.ServiceContext{DB: &orm.DB{DB: db}}
}

func TestMySQLPermissionReadCannotMixSelectionAndRevision(t *testing.T) {
	s := mysqlPermissionFixture(t)
	ctx := context.Background()
	readReady := make(chan struct{})
	releaseRead := make(chan struct{})
	readDone := make(chan error, 1)
	writeDone := make(chan error, 1)
	old := currentPermissionRevision(t, s)
	go func() {
		readDone <- accessutil.PermissionRead(s.DB.DB, func(tx *gorm.DB) error {
			first, err := accessutil.LoadPermissionEdit(tx, 88, "menu")
			if err != nil {
				return err
			}
			close(readReady)
			<-releaseRead
			second, err := accessutil.LoadPermissionEdit(tx, 88, "menu")
			if err != nil {
				return err
			}
			if first.Revision != old || second.Revision != old || len(first.MenuIds) != 1 || len(second.MenuIds) != 1 {
				return fmt.Errorf("mixed selection/revision snapshot")
			}
			return nil
		})
	}()
	<-readReady
	go func() {
		_, err := authoritylogic.NewAddAuthorityMenuLogic(ctx, s).AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, ExpectedRevision: old})
		writeDone <- err
	}()
	select {
	case err := <-writeDone:
		close(releaseRead)
		t.Fatalf("writer escaped read lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(releaseRead)
	must(t, <-readDone)
	must(t, <-writeDone)
	var count int64
	must(t, s.DB.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ?", 88).Count(&count).Error)
	if count != 0 || currentPermissionRevision(t, s) == old {
		t.Fatal("revocation did not commit after consistent read")
	}
}

func TestMySQLPermissionMutualParentsSerialize(t *testing.T) {
	s := mysqlPermissionFixture(t)
	start := make(chan struct{})
	errs := make(chan error, 2)
	for _, ids := range [][2]int64{{88, 89}, {89, 88}} {
		go func(ids [2]int64) {
			<-start
			_, err := authoritylogic.NewUpdateAuthorityLogic(context.Background(), s).UpdateAuthority(&pb.UpdateAuthorityRequest{SysAuthority: &pb.SysAuthority{AuthorityId: ids[0], ParentId: ids[1], AuthorityName: "role"}})
			errs <- err
		}(ids)
	}
	close(start)
	failed := 0
	for i := 0; i < 2; i++ {
		if <-errs != nil {
			failed++
		}
	}
	if failed != 1 {
		t.Fatalf("expected one parent-cycle rejection, got %d", failed)
	}
	var roles []model.SysAuthority
	must(t, s.DB.Find(&roles).Error)
	parents := map[int64]int64{}
	for _, r := range roles {
		if r.ParentId != nil {
			parents[r.AuthorityId] = *r.ParentId
		}
	}
	for id, parent := range parents {
		must(t, accessutil.ValidateParent(id, parent, parents))
	}
}

func TestMySQLPermissionHomeAndRevocationSerialize(t *testing.T) {
	s := mysqlPermissionFixture(t)
	revision := currentPermissionRevision(t, s)
	start := make(chan struct{})
	errs := make(chan error, 2)
	go func() {
		<-start
		_, err := authoritylogic.NewUpdateAuthorityLogic(context.Background(), s).UpdateAuthority(&pb.UpdateAuthorityRequest{SysAuthority: &pb.SysAuthority{AuthorityId: 88, AuthorityName: "role", DefaultRouter: "home"}})
		errs <- err
	}()
	go func() {
		<-start
		_, err := authoritylogic.NewAddAuthorityMenuLogic(context.Background(), s).AddAuthorityMenu(&pb.AddAuthorityMenuRequest{AuthorityId: 88, ExpectedRevision: revision})
		errs <- err
	}()
	close(start)
	failed := 0
	for i := 0; i < 2; i++ {
		if <-errs != nil {
			failed++
		}
	}
	if failed != 1 {
		t.Fatalf("expected one stale/homepage rejection, got %d", failed)
	}
	var role model.SysAuthority
	must(t, s.DB.First(&role, "authority_id = ?", 88).Error)
	if role.DefaultRouter != "" {
		var count int64
		must(t, s.DB.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ? AND sys_base_menu_id = ?", 88, 7).Count(&count).Error)
		if count != 1 {
			t.Fatal("default homepage points to a revoked menu")
		}
	}
}
