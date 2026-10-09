package userlogic

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/hash"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

func TestRegisterRejectsOversizedProfileBeforeWrite(t *testing.T) {
	for _, field := range []string{"username", "nickName", "headerImg", "phone", "email"} {
		t.Run(field, func(t *testing.T) {
			service := testUserDB(t)
			input := &pb.UserInfo{Username: "new_user", Password: "before123", AuthorityId: 801, Enable: 1}
			oversized := strings.Repeat("界", 192)
			switch field {
			case "username":
				input.Username = oversized
			case "nickName":
				input.NickName = oversized
			case "headerImg":
				input.HeaderImg = oversized
			case "phone":
				input.Phone = oversized
			case "email":
				input.Email = oversized
			}
			_, err := NewRegisterLogic(context.Background(), service).Register(&pb.RegisterRequest{UserInfo: input, AuthorityIds: []int64{801}})
			var business *xerr.CodeError
			if !errors.As(err, &business) || business.GetErrCode() != xerr.REUQEST_PARAM_ERROR {
				t.Fatalf("oversized %s must return input error, got %v", field, err)
			}
			var count int64
			service.DB.Model(&model.SysUser{}).Count(&count)
			if count != 0 {
				t.Fatal("invalid profile was persisted")
			}
		})
	}
}

func TestUserUpdatesValidateOnlyExplicitFields(t *testing.T) {
	for _, field := range []string{"nickName", "headerImg", "phone", "email", "sideMode"} {
		t.Run(field, func(t *testing.T) {
			input := &pb.UserInfo{ID: 1, NickName: strings.Repeat("界", 192), HeaderImg: strings.Repeat("界", 192), Phone: strings.Repeat("界", 192), Email: strings.Repeat("界", 192), SideMode: strings.Repeat("界", 192)}
			if _, err := userUpdates(&pb.UpdateUserInfoRequest{UserInfo: input, UpdateFields: []string{field}}); err == nil {
				t.Fatal("oversized explicit field accepted")
			}
			if updates, err := userUpdates(&pb.UpdateUserInfoRequest{UserInfo: input}); err != nil || len(updates) != 0 {
				t.Fatal("unrequested historical profile should be ignored", err)
			}
		})
	}
	for _, email := range []string{"broken", "Display Name <user@example.test>", "user@example.test\n", " user@example.test "} {
		if _, err := userUpdates(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{Email: email}, UpdateFields: []string{"email"}}); err == nil {
			t.Errorf("invalid explicit email accepted: %q", email)
		}
	}
	for _, email := range []string{"", "User+tag@example.test"} {
		updates, err := userUpdates(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{Email: email}, UpdateFields: []string{"email"}})
		if err != nil || updates["email"] != email {
			t.Fatalf("valid email/clear must be preserved: %q, %v", email, err)
		}
	}
	if updates, err := userUpdates(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{NickName: strings.Repeat("界", 191)}, UpdateFields: []string{"nickName"}}); err != nil || updates["nick_name"] != strings.Repeat("界", 191) {
		t.Fatal("varchar bound is characters, not UTF-8 bytes", err)
	}
}

func TestLoginNormalizesNewNamesAndPreservesExactHistoricalNames(t *testing.T) {
	service := testUserDB(t)
	normalized := sessionUser(t, service, "MixedCase", 801)
	logic := NewGetUserInfoLogic(context.Background(), service)
	got, err := logic.GetUserInfo(&pb.GetUserInfoRequest{UserName: " \tMixedCase\n", Password: "before123"})
	if err != nil || got.UserInfo.ID != normalized.ID {
		t.Fatalf("new registered name must allow surrounding whitespace: %v", err)
	}
	legacy := *normalized
	legacy.ID = 0
	legacy.Username = " MixedCase "
	legacy.Password, err = hash.BcryptHash("legacy123")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Omit("Authority", "Authorities").Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Create(&model.SysUserAuthority{SysUserId: legacy.ID, SysAuthorityAuthorityId: 801}).Error; err != nil {
		t.Fatal(err)
	}
	got, err = logic.GetUserInfo(&pb.GetUserInfoRequest{UserName: legacy.Username, Password: "legacy123"})
	if err != nil || got.UserInfo.ID != legacy.ID {
		t.Fatal("exact historical username must win over normalized fallback", err)
	}
	if _, err := logic.GetUserInfo(&pb.GetUserInfoRequest{UserName: legacy.Username, Password: "before123"}); err == nil {
		t.Fatal("wrong password on exact account must not fall back to normalized account")
	}
	if _, err := logic.GetUserInfo(&pb.GetUserInfoRequest{UserName: "mixedcase", Password: "before123"}); err == nil {
		t.Fatal("login must not alter case semantics")
	}
	if _, err := logic.GetUserInfo(&pb.GetUserInfoRequest{UserName: " MixedCase ", Password: " before123 "}); err == nil {
		t.Fatal("login must not normalize passwords")
	}
}

func TestUserListDoesNotReadPasswordAndKeepsAssociations(t *testing.T) {
	service := testUserDB(t)
	created := sessionUser(t, service, "list_projection", 801)
	observed := false
	if err := service.DB.Callback().Query().After("gorm:query").Register("test:observe_user_list_password", func(tx *gorm.DB) {
		if users, ok := tx.Statement.Dest.(*[]model.SysUser); ok && len(*users) > 0 {
			observed = true
			if (*users)[0].Password != "" {
				t.Error("list fetched password hash before masking")
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	got, err := NewGetUserListLogic(context.Background(), service).GetUserList(&pb.GetUserListRequest{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}})
	if err != nil {
		t.Fatal(err)
	}
	if !observed || got.Total != 1 || len(got.UserInfoList) != 1 {
		t.Fatal("list was not queried")
	}
	user := got.UserInfoList[0]
	if user.ID != created.ID || user.Username != created.Username || user.SessionVersion != created.SessionVersion || user.UUID != created.UUID.String() || user.Authority.AuthorityId != 801 || len(user.Authorities) != 1 || user.Password != "" {
		t.Fatalf("projection lost response or preload fields: %+v", user)
	}
}

func TestRegisterPreservesUnicodeBoundAndTrimmedName(t *testing.T) {
	service := testUserDB(t)
	name := strings.Repeat("界", 191)
	_, err := NewRegisterLogic(context.Background(), service).Register(&pb.RegisterRequest{UserInfo: &pb.UserInfo{Username: " \t" + name + "\n", NickName: name, Phone: name, HeaderImg: name, Email: "User+tag@example.test", Password: "before123", AuthorityId: 801}, AuthorityIds: []int64{801}})
	if err != nil {
		t.Fatal("191 Unicode characters must fit production varchar", err)
	}
	var stored model.SysUser
	if err := service.DB.Where("username = ?", name).First(&stored).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Username != name || stored.NickName != name || stored.Phone != name || stored.HeaderImg != name || stored.Email != "User+tag@example.test" {
		t.Fatal("register changed supplied profile")
	}
}

func TestUnrelatedUpdateDoesNotValidateHistoricalEmail(t *testing.T) {
	service := testUserDB(t)
	created := sessionUser(t, service, "legacy_profile", 801)
	if err := service.DB.Model(created).Updates(map[string]any{"email": "legacy text", "phone": "keep", "header_img": "keep"}).Error; err != nil {
		t.Fatal(err)
	}
	_, err := NewUpdateUserInfoLogic(context.Background(), service).UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: created.ID, NickName: "changed", Email: "still invalid", Phone: "", HeaderImg: ""}, UpdateFields: []string{"nickName"}})
	if err != nil {
		t.Fatal("unrelated edit must allow historical email", err)
	}
	var stored model.SysUser
	if err := service.DB.First(&stored, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Email != "legacy text" || stored.Phone != "keep" || stored.HeaderImg != "keep" || stored.NickName != "changed" {
		t.Fatal("unsupplied profile fields changed")
	}
	_, err = NewUpdateUserInfoLogic(context.Background(), service).UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: created.ID}, UpdateFields: []string{"email", "phone", "headerImg"}})
	if err != nil {
		t.Fatal("explicit empty fields must clear historical values", err)
	}
	if err := service.DB.First(&stored, created.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.Email != "" || stored.Phone != "" || stored.HeaderImg != "" {
		t.Fatal("explicit clear was lost")
	}
}
