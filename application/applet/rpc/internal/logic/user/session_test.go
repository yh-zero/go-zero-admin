package userlogic

import (
	"context"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/hash"
	"testing"
)

func sessionUser(t *testing.T, svcCtx *svc.ServiceContext, name string, role int64) *model.SysUser {
	t.Helper()
	_, err := NewRegisterLogic(context.Background(), svcCtx).Register(&pb.RegisterRequest{UserInfo: &pb.UserInfo{Username: name, Password: "before123", AuthorityId: role, Enable: 1}, AuthorityIds: []int64{role}})
	if err != nil {
		t.Fatal(err)
	}
	var user model.SysUser
	if err := svcCtx.DB.Where("username = ?", name).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	return &user
}
func assertSession(t *testing.T, svcCtx *svc.ServiceContext, req *pb.SessionRequest, want bool) {
	t.Helper()
	got, err := NewCheckSessionLogic(context.Background(), svcCtx).CheckSession(req)
	if err != nil || got.Valid != want {
		t.Fatalf("session valid want %v; got %+v %v", want, got, err)
	}
}
func TestPasswordAndLogoutRevokeSessions(t *testing.T) {
	svcCtx := testUserDB(t)
	user := sessionUser(t, svcCtx, "session_account", 801)
	ctx := context.Background()
	req := &pb.SessionRequest{UserID: user.ID, SessionVersion: user.SessionVersion, AuthorityId: 801}
	assertSession(t, svcCtx, req, true)
	me, err := NewGetCurrentUserLogic(ctx, svcCtx).GetCurrentUser(req)
	if err != nil || me.UserInfo.Username != user.Username || me.UserInfo.Password != "" {
		t.Fatal("current user lookup leaked password or failed", err)
	}
	change := NewChangePasswordLogic(ctx, svcCtx)
	for _, input := range []*pb.ChangePasswordRequest{
		{Session: req, OldPassword: "wrong", NewPassword: "after1234"},
		{Session: req, OldPassword: "before123", NewPassword: "short"},
		{Session: req, OldPassword: "before123", NewPassword: "before123"},
	} {
		if _, err := change.ChangePassword(input); err == nil {
			t.Fatal("invalid password change succeeded")
		}
	}
	assertSession(t, svcCtx, req, true)
	if _, err := change.ChangePassword(&pb.ChangePasswordRequest{Session: req, OldPassword: "before123", NewPassword: "after1234"}); err != nil {
		t.Fatal(err)
	}
	assertSession(t, svcCtx, req, false)
	if _, err := NewGetCurrentUserLogic(ctx, svcCtx).GetCurrentUser(req); err == nil {
		t.Fatal("old session read current user")
	}
	var fresh model.SysUser
	svcCtx.DB.First(&fresh, user.ID)
	if !hash.BcryptCheck("after1234", fresh.Password) {
		t.Fatal("new password not persisted")
	}
	req.SessionVersion = fresh.SessionVersion
	assertSession(t, svcCtx, req, true)
	if _, err := NewLogoutLogic(ctx, svcCtx).Logout(req); err != nil {
		t.Fatal(err)
	}
	assertSession(t, svcCtx, req, false)
	if _, err := NewLogoutLogic(ctx, svcCtx).Logout(req); err != nil {
		t.Fatal(err)
	}
	req.SessionVersion++
	assertSession(t, svcCtx, req, true) // Replaying old logout cannot revoke a subsequent login.
}
func TestUserMutationsRevokeSessions(t *testing.T) {
	for _, action := range []string{"freeze", "roles", "reset", "delete"} {
		t.Run(action, func(t *testing.T) {
			svcCtx := testUserDB(t)
			user := sessionUser(t, svcCtx, "account", 801)
			ctx := context.Background()
			req := &pb.SessionRequest{UserID: user.ID, SessionVersion: user.SessionVersion, AuthorityId: 801}
			assertSession(t, svcCtx, req, true)
			var err error
			switch action {
			case "freeze":
				_, err = NewUpdateUserInfoLogic(ctx, svcCtx).UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: user.ID, Enable: 2}, UpdateFields: []string{"enable"}})
			case "roles":
				_, err = NewUpdateUserInfoLogic(ctx, svcCtx).UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: user.ID, AuthorityId: 802}, UpdateFields: []string{"authorityId"}, UpdateAuthorities: true, AuthorityIds: []int64{802}})
			case "reset":
				svcCtx.Config.Default.UserPassword = "goZero"
				_, err = NewResetUserPasswordLogic(ctx, svcCtx).ResetUserPassword(&pb.ResetUserPasswordRequest{UserID: user.ID})
			case "delete":
				_, err = NewDeleteUserLogic(ctx, svcCtx).DeleteUser(&pb.DeleteUserRequest{UserID: user.ID})
			}
			if err != nil {
				t.Fatal(err)
			}
			assertSession(t, svcCtx, req, false)
		})
	}
}
func TestLastAdministratorCannotBeRemoved(t *testing.T) {
	svcCtx := testUserDB(t)
	ctx := context.Background()
	if err := svcCtx.DB.Create(&model.SysAuthority{AuthorityId: 1, AuthorityName: "admin"}).Error; err != nil {
		t.Fatal(err)
	}
	admin := sessionUser(t, svcCtx, "administrator", 1)
	update := NewUpdateUserInfoLogic(ctx, svcCtx)
	if _, err := update.UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: admin.ID, Enable: 2}, UpdateFields: []string{"enable"}}); err == nil {
		t.Fatal("last administrator frozen")
	}
	if _, err := update.UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: admin.ID, AuthorityId: 801}, UpdateFields: []string{"authorityId"}, UpdateAuthorities: true, AuthorityIds: []int64{801}}); err == nil {
		t.Fatal("last administrator demoted")
	}
	if _, err := NewDeleteUserLogic(ctx, svcCtx).DeleteUser(&pb.DeleteUserRequest{UserID: admin.ID}); err == nil {
		t.Fatal("last administrator deleted")
	}
	assertSession(t, svcCtx, &pb.SessionRequest{UserID: admin.ID, SessionVersion: admin.SessionVersion, AuthorityId: 1}, true)
	sessionUser(t, svcCtx, "administrator2", 1)
	if _, err := NewDeleteUserLogic(ctx, svcCtx).DeleteUser(&pb.DeleteUserRequest{UserID: admin.ID}); err != nil {
		t.Fatal(err)
	}
}
func TestTokenChecksVersionBeforeIssuing(t *testing.T) {
	svcCtx := testUserDB(t)
	user := sessionUser(t, svcCtx, "token_account", 801)
	svcCtx.Config.JwtAuth.AccessSecret = "session-test-secret"
	svcCtx.Config.JwtAuth.AccessExpire = 7200
	req := &pb.GetUserTokeRequest{ID: user.ID, AuthorityId: 801, SessionVersion: user.SessionVersion, Username: user.Username}
	result, err := NewGetUserTokeLogic(context.Background(), svcCtx).GetUserToke(req)
	if err != nil || result.Token == "" {
		t.Fatal("active user token not issued", err)
	}
	req.SessionVersion = 0
	if _, err := NewGetUserTokeLogic(context.Background(), svcCtx).GetUserToke(req); err == nil {
		t.Fatal("legacy session version accepted")
	}
}

func TestUnchangedRolesAndProfileKeepSession(t *testing.T) {
	svcCtx := testUserDB(t)
	user := sessionUser(t, svcCtx, "profile_account", 801)
	_, err := NewUpdateUserInfoLogic(context.Background(), svcCtx).UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: user.ID, AuthorityId: 801, Enable: 1, NickName: "new name"}, UpdateFields: []string{"authorityId", "enable", "nickName"}, UpdateAuthorities: true, AuthorityIds: []int64{801}})
	if err != nil {
		t.Fatal(err)
	}
	assertSession(t, svcCtx, &pb.SessionRequest{UserID: user.ID, AuthorityId: 801, SessionVersion: user.SessionVersion}, true)
}
