package accessutil_test

import (
	"context"
	"errors"
	"testing"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	authoritylogic "go-zero-admin/application/applet/rpc/internal/logic/authority"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/middlecasbin"
	"gorm.io/gorm"
)

func TestPolicyAndDurableVersionRollBackWhenAuditFails(t *testing.T) {
	s := service(t)
	must(t, s.DB.Create(&middlecasbin.PolicyVersion{ID: 1, Version: 3}).Error)
	must(t, s.DB.Callback().Create().Before("gorm:create").Register("test:policy-audit-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_audit_logs" {
			tx.AddError(errors.New("injected audit failure"))
		}
	}))
	err := accessutil.PolicyTransaction(context.Background(), s, func(tx *gorm.DB) error {
		return tx.Create(&gormadapter.CasbinRule{Ptype: "p", V0: "88", V1: "/resource", V2: "GET"}).Error
	})
	mustFail(t, err)
	var version middlecasbin.PolicyVersion
	must(t, s.DB.First(&version, 1).Error)
	if version.Version != 3 {
		t.Fatal("audit failure advanced durable policy version")
	}
	var count int64
	must(t, s.DB.Model(&gormadapter.CasbinRule{}).Count(&count).Error)
	if count != 0 {
		t.Fatal("audit failure committed permission rule")
	}
	must(t, s.DB.Callback().Create().Remove("test:policy-audit-failure"))
	ctx := audit.WithRequest(context.Background(), audit.Request{Path: "/v1/sys/casbin/updateCasbinData", Method: "PUT"})
	must(t, accessutil.PolicyTransaction(ctx, s, func(tx *gorm.DB) error {
		return tx.Create(&gormadapter.CasbinRule{Ptype: "p", V0: "88", V1: "/resource", V2: "GET"}).Error
	}))
	var event audit.Event
	must(t, s.DB.Where("action=?", "commitPolicy").First(&event).Error)
	if event.Object != "/v1/sys/casbin/updateCasbinData" || event.Method != "PUT" || event.ActorID != 0 || event.Params != "{}" {
		t.Fatalf("unexpected transaction audit: %+v", event)
	}
}

func TestDeletingRoleCleansDataScopeAssociations(t *testing.T) {
	s := service(t)
	must(t, s.DB.Create(&model.SysRoleDataScope{AuthorityID: 88, Scope: "custom"}).Error)
	must(t, s.DB.Create(&model.SysRoleScopeDepartment{AuthorityID: 88, DepartmentID: 11}).Error)
	_, err := authoritylogic.NewDeleteAuthorityLogic(context.Background(), s).DeleteAuthority(&pb.DeleteAuthorityRequest{ID: 88})
	must(t, err)
	for _, table := range []interface{}{&model.SysRoleDataScope{}, &model.SysRoleScopeDepartment{}} {
		var count int64
		must(t, s.DB.Model(table).Where("authority_id=?", 88).Count(&count).Error)
		if count != 0 {
			t.Fatal("deleted role retained data scope associations")
		}
	}
}
