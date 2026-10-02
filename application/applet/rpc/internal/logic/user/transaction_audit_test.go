package userlogic

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/hash"
	"gorm.io/gorm"
)

func TestPasswordResetRollsBackWhenTransactionAuditFails(t *testing.T) {
	s := testUserDB(t)
	user := sessionUser(t, s, "audit_password_reset", 801)
	s.Config.Default.UserPassword = "newResetPassword123"
	if err := s.DB.Callback().Create().Before("gorm:create").Register("test:audit-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_audit_logs" {
			tx.AddError(errors.New("injected audit failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewResetUserPasswordLogic(context.Background(), s).ResetUserPassword(&pb.ResetUserPasswordRequest{UserID: user.ID}); err == nil {
		t.Fatal("audit failure was ignored")
	}
	var stored model.SysUser
	if err := s.DB.First(&stored, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Password != user.Password || stored.SessionVersion != user.SessionVersion {
		t.Fatal("password/session change escaped failed audit transaction")
	}
	if err := s.DB.Callback().Create().Remove("test:audit-failure"); err != nil {
		t.Fatal(err)
	}
	ctx := audit.WithRequest(context.Background(), audit.Request{Path: "/v1/sys/user/resetUserPassword", Method: "PUT"})
	audit.SetActor(ctx, audit.Actor{ID: 91, Name: "operator", AuthorityID: 1})
	if _, err := NewResetUserPasswordLogic(ctx, s).ResetUserPassword(&pb.ResetUserPasswordRequest{UserID: user.ID}); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.First(&stored, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !hash.BcryptCheck(s.Config.Default.UserPassword, stored.Password) || stored.SessionVersion != user.SessionVersion+1 {
		t.Fatal("successful reset did not persist")
	}
	var event audit.Event
	if err := s.DB.Where("module=? AND action=?", "user", "resetPassword").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.ActorID != 91 || event.Object != strconv.FormatInt(user.ID, 10) || event.Params != "{}" {
		t.Fatalf("unexpected audit event: %+v", event)
	}
}

func TestRegistrationRecordsUnknownActorWithoutSkippingAudit(t *testing.T) {
	s := testUserDB(t)
	if _, err := NewRegisterLogic(context.Background(), s).Register(&pb.RegisterRequest{UserInfo: &pb.UserInfo{Username: "maintenance_registered", Password: "password123", AuthorityId: 801}, AuthorityIds: []int64{801}}); err != nil {
		t.Fatal(err)
	}
	var event audit.Event
	if err := s.DB.Where("action=?", "register").First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.ActorID != 0 || event.Object == "" {
		t.Fatal("maintenance audit should retain zero actor and object ID")
	}
}

func TestDeletingUserCleansOrganizationMemberships(t *testing.T) {
	s := testUserDB(t)
	user := sessionUser(t, s, "delete_organization_member", 801)
	if err := s.DB.Create(&model.SysUserDepartment{UserID: user.ID, DepartmentID: 11}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Create(&model.SysUserPosition{UserID: user.ID, PositionID: 12}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewDeleteUserLogic(context.Background(), s).DeleteUser(&pb.DeleteUserRequest{UserID: user.ID}); err != nil {
		t.Fatal(err)
	}
	for _, table := range []interface{}{&model.SysUserDepartment{}, &model.SysUserPosition{}} {
		var count int64
		if err := s.DB.Model(table).Where("user_id=?", user.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatal("deleted user retained organization memberships")
		}
	}
}
