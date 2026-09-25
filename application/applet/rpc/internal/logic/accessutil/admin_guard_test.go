package accessutil_test

import (
	"context"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	apilogic "go-zero-admin/application/applet/rpc/internal/logic/api"
	casbinlogic "go-zero-admin/application/applet/rpc/internal/logic/casbin"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/gorm"
)

func seedRecoveryRole(t *testing.T, s *svc.ServiceContext) []model.SysApi {
	t.Helper()
	zero := int64(0)
	must(t, s.DB.Create(&model.SysAuthority{AuthorityId: accessutil.AdminAuthorityID, AuthorityName: "administrator", ParentId: &zero}).Error)
	paths := []string{"/v1/sys/menu/getMenu", "/v1/sys/authority/getAuthorityList", "/v1/sys/api/getApiList", "/v1/sys/api/getAllApiList", "/v1/sys/casbin/getPathByAuthorityId", "/v1/sys/casbin/updateCasbinDataByApiIds"}
	resources := make([]model.SysApi, 0, len(paths))
	for i, path := range paths {
		method := "GET"
		if i == len(paths)-1 {
			method = "PUT"
		}
		resource := model.SysApi{Path: path, Method: method}
		must(t, s.DB.Create(&resource).Error)
		must(t, s.DB.Create(&gormadapter.CasbinRule{Ptype: "p", V0: "1", V1: path, V2: method}).Error)
		resources = append(resources, resource)
	}
	must(t, s.Casbin.LoadPolicy())
	return resources
}

func TestAdministratorCannotLoseAllRecoveryPermissions(t *testing.T) {
	s := service(t)
	resources := seedRecoveryRole(t, s)
	ctx := context.Background()
	_, err := casbinlogic.NewUpdateCasbinDataByApiIdsLogic(ctx, s).UpdateCasbinDataByApiIds(&pb.UpdateCasbinDataByApiIdsRequest{AuthorityId: 1})
	mustFail(t, err)
	_, err = casbinlogic.NewUpdateCasbinDataLogic(ctx, s).UpdateCasbinData(&pb.UpdateCasbinDataRequest{AuthorityId: 1})
	mustFail(t, err)
	_, err = apilogic.NewDeleteApisByIdsLogic(ctx, s).DeleteApisByIds(&pb.DeleteApisByIdsRequest{Ids: []int64{resources[0].ID}})
	mustFail(t, err)
	_, err = apilogic.NewUpdateApiLogic(ctx, s).UpdateApi(&pb.UpdateApiRequest{SysApi: &pb.SysApi{ID: resources[0].ID, Path: "/v1/sys/renamedRecovery", Method: "GET"}})
	mustFail(t, err)
	var count int64
	must(t, s.DB.Model(&gormadapter.CasbinRule{}).Where("v0=?", "1").Count(&count).Error)
	if count != int64(len(resources)) {
		t.Fatal("denied policy changes were not rolled back")
	}
	var stored model.SysApi
	must(t, s.DB.First(&stored, resources[0].ID).Error)
	if stored.Path != resources[0].Path {
		t.Fatal("denied API rename changed resource")
	}
	// Ordinary roles remain revocable; this protection is specific to recovery.
	_, err = casbinlogic.NewUpdateCasbinDataByApiIdsLogic(ctx, s).UpdateCasbinDataByApiIds(&pb.UpdateCasbinDataByApiIdsRequest{AuthorityId: 88})
	must(t, err)
}

func TestAdministratorRecoveryAcceptsExistingWildcardPolicy(t *testing.T) {
	s := service(t)
	seedRecoveryRole(t, s)
	must(t, s.DB.Where("v0=?", "1").Delete(&gormadapter.CasbinRule{}).Error)
	for _, method := range []string{"GET", "PUT"} {
		must(t, s.DB.Create(&gormadapter.CasbinRule{Ptype: "p", V0: "1", V1: "/v1/sys/*", V2: method}).Error)
	}
	must(t, accessutil.EnsureAdminRecoveryPolicies(s.DB.DB))
}

func TestAdministratorRecoveryHandlesMalformedLegacyRule(t *testing.T) {
	s := service(t)
	seedRecoveryRole(t, s)
	must(t, s.DB.Where("v0=?", "1").Delete(&gormadapter.CasbinRule{}).Error)
	must(t, s.DB.Create(&gormadapter.CasbinRule{Ptype: "p", V0: "1", V1: "/v1/sys/[", V2: "GET"}).Error)
	// No panic: bad legacy data should produce a recoverable validation error.
	mustFail(t, accessutil.EnsureAdminRecoveryPolicies(s.DB.DB))
	_, _, err := accessutil.API("/v1/sys/[", "GET")
	mustFail(t, err)
	for _, path := range []string{"/v1/sys/*", "/v1/sys/:id", "/v1/sys/items/:id/details"} {
		_, _, err := accessutil.API(path, "GET")
		must(t, err)
	}
}

func TestLastUsableAdministratorGuard(t *testing.T) {
	s := service(t)
	seedRecoveryRole(t, s)
	createAdmin := func(username string) model.SysUser {
		user := model.SysUser{Username: username, AuthorityId: 1, Enable: 1}
		must(t, s.DB.Omit("Authority", "Authorities").Create(&user).Error)
		must(t, s.DB.Create(&model.SysUserAuthority{SysUserId: user.ID, SysAuthorityAuthorityId: 1}).Error)
		return user
	}
	first := createAdmin("first-admin")
	for _, operation := range []string{"disable", "default-role", "membership", "delete"} {
		t.Run(operation, func(t *testing.T) {
			err := s.DB.Transaction(func(tx *gorm.DB) error {
				if err := accessutil.LockAdminGuard(tx); err != nil {
					return err
				}
				var err error
				switch operation {
				case "disable":
					err = tx.Model(&model.SysUser{}).Where("id=?", first.ID).Update("enable", 2).Error
				case "default-role":
					err = tx.Model(&model.SysUser{}).Where("id=?", first.ID).Update("authority_id", 88).Error
				case "membership":
					err = tx.Where("sys_user_id=?", first.ID).Delete(&model.SysUserAuthority{}).Error
				case "delete":
					err = tx.Delete(&first).Error
				}
				if err != nil {
					return err
				}
				return accessutil.EnsureUsableAdministrator(tx)
			})
			mustFail(t, err)
			must(t, accessutil.EnsureUsableAdministrator(s.DB.DB))
		})
	}
	createAdmin("second-admin")
	must(t, s.DB.Transaction(func(tx *gorm.DB) error {
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		if err := tx.Model(&model.SysUser{}).Where("id=?", first.ID).Update("enable", 2).Error; err != nil {
			return err
		}
		return accessutil.EnsureUsableAdministrator(tx)
	}))
}
