package userlogic

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	mysqldriver "github.com/go-sql-driver/mysql"
)

func TestRegistrationDuplicateKeyMessage(t *testing.T) {
	duplicate := fmt.Errorf("insert user: %w", &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry 'user' for key 'sys_users.uk_sys_users_active_username'"})
	translated := registrationError(duplicate)
	var business *xerr.CodeError
	if !errors.As(translated, &business) || business.GetErrCode() != xerr.REUQEST_PARAM_ERROR || business.GetErrMsg() != "用户名已注册" {
		t.Fatalf("duplicate registration message lost: %v", translated)
	}
	otherIndex := &mysqldriver.MySQLError{Number: 1062, Message: "Duplicate entry 'id' for key 'PRIMARY'"}
	if registrationError(otherIndex) != otherIndex {
		t.Fatal("unrelated unique failure must not be called a duplicate username")
	}
	connection := &mysqldriver.MySQLError{Number: 1040, Message: "too many connections"}
	if registrationError(connection) != connection {
		t.Fatal("internal errors must stay internal")
	}
}

// SQLite exercises the same generated-value and unique-NULL semantics in isolation.
// The MySQL migration uses STORED; SQLite ALTER TABLE only supports adding VIRTUAL.
func TestActiveUsernameConstraintAndSoftDeleteReuse(t *testing.T) {
	service := testUserDB(t)
	if err := service.DB.Exec("ALTER TABLE sys_users ADD COLUMN active_username TEXT GENERATED ALWAYS AS (CASE WHEN deleted_at IS NULL THEN username ELSE NULL END) VIRTUAL").Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Exec("CREATE UNIQUE INDEX uk_sys_users_active_username ON sys_users(active_username)").Error; err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	register := NewRegisterLogic(ctx, service)
	request := func() *pb.RegisterRequest {
		return &pb.RegisterRequest{UserInfo: &pb.UserInfo{Username: "reusable_user", Password: "test123456", NickName: "Reusable", AuthorityId: 801, Enable: 1}, AuthorityIds: []int64{801}}
	}
	for i := 0; i < 2; i++ {
		if _, err := register.Register(request()); err != nil {
			t.Fatalf("register attempt %d: %v", i, err)
		}
		// Bypass the application read check, as a concurrent request could do.
		duplicate := model.SysUser{Username: "reusable_user", AuthorityId: 801, Enable: 1}
		if err := service.DB.Omit("Authority", "Authorities").Create(&duplicate).Error; err == nil {
			t.Fatal("database accepted two active usernames")
		}
		var current model.SysUser
		if err := service.DB.Where("username = ?", "reusable_user").First(&current).Error; err != nil {
			t.Fatal(err)
		}
		if _, err := NewDeleteUserLogic(ctx, service).DeleteUser(&pb.DeleteUserRequest{UserID: current.ID}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := register.Register(request()); err != nil {
		t.Fatalf("soft deleted names must remain reusable: %v", err)
	}
	var active, total int64
	service.DB.Model(&model.SysUser{}).Where("username = ?", "reusable_user").Count(&active)
	service.DB.Unscoped().Model(&model.SysUser{}).Where("username = ?", "reusable_user").Count(&total)
	if active != 1 || total != 3 {
		t.Fatalf("history must remain and only one account active: active=%d total=%d", active, total)
	}
}
