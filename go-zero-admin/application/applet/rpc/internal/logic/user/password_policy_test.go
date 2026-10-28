package userlogic

import (
	"context"
	"math"
	"strings"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/hash"
)

func TestRegistrationAndResetUseSamePasswordPolicy(t *testing.T) {
	s := testUserDB(t)
	ctx := context.Background()
	for _, password := range []string{"1234567", strings.Repeat("a", 73), strings.Repeat("密", 25)} {
		_, err := NewRegisterLogic(ctx, s).Register(&pb.RegisterRequest{UserInfo: &pb.UserInfo{Username: "invalid", Password: password, AuthorityId: 801}, AuthorityIds: []int64{801}})
		if err == nil {
			t.Fatalf("registration accepted %d password bytes", len(password))
		}
	}
	user := sessionUser(t, s, "reset_policy", 801)
	for _, password := range []string{"", "goZero", strings.Repeat("a", 73)} {
		s.Config.Default.UserPassword = password
		if _, err := NewResetUserPasswordLogic(ctx, s).ResetUserPassword(&pb.ResetUserPasswordRequest{UserID: user.ID}); err == nil {
			t.Fatalf("reset accepted %d configured password bytes", len(password))
		}
		var stored model.SysUser
		if err := s.DB.First(&stored, user.ID).Error; err != nil {
			t.Fatal(err)
		}
		if stored.Password != user.Password || stored.SessionVersion != user.SessionVersion {
			t.Fatal("failed reset changed password or revoked session")
		}
	}
	s.Config.Default.UserPassword = "goZero123"
	if _, err := NewResetUserPasswordLogic(ctx, s).ResetUserPassword(&pb.ResetUserPasswordRequest{UserID: user.ID}); err != nil {
		t.Fatal(err)
	}
	var stored model.SysUser
	if err := s.DB.First(&stored, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !hash.BcryptCheck("goZero123", stored.Password) || stored.SessionVersion != user.SessionVersion+1 {
		t.Fatal("valid reset did not persist password and revoke session")
	}
}

func TestLegacyShortPasswordCanLoginAndUpgrade(t *testing.T) {
	s := testUserDB(t)
	ctx := context.Background()
	user := sessionUser(t, s, "legacy_password", 801)
	legacyHash, err := hash.BcryptHash("goZero")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Model(&model.SysUser{}).Where("id = ?", user.ID).Update("password", legacyHash).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewGetUserInfoLogic(ctx, s).GetUserInfo(&pb.GetUserInfoRequest{UserName: user.Username, Password: "goZero"}); err != nil {
		t.Fatal("legacy short password login was broken", err)
	}
	session := &pb.SessionRequest{UserID: user.ID, AuthorityId: user.AuthorityId, SessionVersion: user.SessionVersion}
	if _, err := NewChangePasswordLogic(ctx, s).ChangePassword(&pb.ChangePasswordRequest{Session: session, OldPassword: "goZero", NewPassword: "newpass123"}); err != nil {
		t.Fatal("legacy password upgrade failed", err)
	}
	if _, err := NewGetUserInfoLogic(ctx, s).GetUserInfo(&pb.GetUserInfoRequest{UserName: user.Username, Password: "newpass123"}); err != nil {
		t.Fatal("upgraded password cannot login", err)
	}
}

func TestUserListRejectsOffsetOverflowBeforeQuery(t *testing.T) {
	logic := NewGetUserListLogic(context.Background(), &svc.ServiceContext{})
	if _, err := logic.GetUserList(&pb.GetUserListRequest{PageRequest: &pb.PageRequest{PageNo: math.MaxInt64, PageSize: 500}}); err == nil {
		t.Fatal("overflowing user-list page was accepted")
	}
}
