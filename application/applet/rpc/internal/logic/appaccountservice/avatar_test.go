package appaccountservicelogic

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/objectstorage"
)

func avatarFixture(t *testing.T) (*svc.ServiceContext, *pb.SessionRequest, string, string) {
	t.Helper()
	s, actor := accountFixture(t, false)
	if err := s.DB.AutoMigrate(&model.SysFileResource{}, &model.SysFileReference{}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Casbin.AddPolicy("1", "/v1/sys/app-members/avatar", "PUT"); err != nil {
		t.Fatal(err)
	}
	s.Config.Storage = objectstorage.Config{ExternalEndpoint: "https://files.example.test", PublicBucket: "gozero-public", PrivateBucket: "gozero-private"}
	for _, file := range []model.SysFileResource{
		{ID: 101, ObjectKey: "go-zero-admin/00000000-0000-4000-8000-000000000101.png", Name: "one.png", Mime: "image/png", Size: 20, OwnerID: actor.UserID, Visibility: "public", Status: "active"},
		{ID: 102, ObjectKey: "go-zero-admin/00000000-0000-4000-8000-000000000102.png", Name: "two.png", Mime: "image/png", Size: 20, OwnerID: actor.UserID, Visibility: "public", Status: "active"},
		{ID: 103, ObjectKey: "go-zero-admin/00000000-0000-4000-8000-000000000103.png", Name: "private.png", Mime: "image/png", Size: 20, OwnerID: actor.UserID, Visibility: "private", Status: "active"},
		{ID: 104, ObjectKey: "go-zero-admin/00000000-0000-4000-8000-000000000104.png", Name: "deleting.png", Mime: "image/png", Size: 20, OwnerID: actor.UserID, Visibility: "public", Status: "deleting"},
	} {
		if err := s.DB.Create(&file).Error; err != nil {
			t.Fatal(err)
		}
	}
	return s, actor, "https://files.example.test/gozero-public/go-zero-admin/00000000-0000-4000-8000-000000000101.png", "https://files.example.test/gozero-public/go-zero-admin/00000000-0000-4000-8000-000000000102.png"
}

func avatarReferenceCount(t *testing.T, s *svc.ServiceContext, userID, fileID int64) int64 {
	t.Helper()
	var count int64
	if err := s.DB.Model(&model.SysFileReference{}).Where("file_id = ? AND object_type = ? AND object_id = ?", fileID, "app_user_avatar", strconv.FormatInt(userID, 10)).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func TestAppAvatarReferencesFollowReplacementAndClear(t *testing.T) {
	s, actor, first, second := avatarFixture(t)
	u, token := seedActive(t, s, 7)
	ctx := context.Background()
	set := NewSetAppUserAvatarLogic(ctx, s)
	for _, next := range []string{first, first, second} {
		if _, err := set.SetAppUserAvatar(&pb.AppAvatarRequest{Actor: actor, UserID: u.ID, AvatarURL: next}); err != nil {
			t.Fatal(err)
		}
	}
	if avatarReferenceCount(t, s, u.ID, 101) != 0 || avatarReferenceCount(t, s, u.ID, 102) != 1 {
		t.Fatal("replacement did not move exactly one protected reference")
	}
	profile, err := NewUpdateAppProfileLogic(ctx, s).UpdateAppProfile(&pb.AppProfileRequest{AccessToken: token, Nickname: "new name"})
	if err != nil || profile.AvatarURL != second {
		t.Fatalf("nickname update lost avatar: %+v %v", profile, err)
	}
	listed, err := NewListAppUsersLogic(ctx, s).ListAppUsers(&pb.AppAdminListRequest{Actor: actor})
	if err != nil || len(listed.List) != 1 || listed.List[0].AvatarURL != second {
		t.Fatalf("admin list lost avatar: %+v %v", listed, err)
	}
	if _, err := set.SetAppUserAvatar(&pb.AppAvatarRequest{Actor: actor, UserID: u.ID, AvatarURL: ""}); err != nil {
		t.Fatal(err)
	}
	profile, err = NewGetAppProfileLogic(ctx, s).GetAppProfile(&pb.AppAccessRequest{AccessToken: token})
	if err != nil || profile.AvatarURL != "" || avatarReferenceCount(t, s, u.ID, 102) != 0 {
		t.Fatalf("clear must restore the App client's default and unpin the object: %+v %v", profile, err)
	}
}

func TestAppAvatarRejectsUnauthorizedOrUnavailableManagedFiles(t *testing.T) {
	for _, scenario := range []string{"missing session", "revoked session", "missing permission", "private", "deleting", "missing user", "script URL", "overlong URL"} {
		t.Run(scenario, func(t *testing.T) {
			s, actor, first, _ := avatarFixture(t)
			u, _ := seedActive(t, s, 7)
			request := &pb.AppAvatarRequest{Actor: actor, UserID: u.ID, AvatarURL: first}
			switch scenario {
			case "missing session":
				request.Actor = nil
			case "revoked session":
				if err := s.DB.Model(&model.SysUser{}).Where("id = ?", actor.UserID).Update("session_version", 2).Error; err != nil {
					t.Fatal(err)
				}
			case "missing permission":
				if _, err := s.Casbin.RemovePolicy("1", "/v1/sys/app-members/avatar", "PUT"); err != nil {
					t.Fatal(err)
				}
			case "private":
				request.AvatarURL = "https://files.example.test/gozero-private/go-zero-admin/00000000-0000-4000-8000-000000000103.png?X-Amz-Signature=old"
			case "deleting":
				request.AvatarURL = "https://files.example.test/gozero-public/go-zero-admin/00000000-0000-4000-8000-000000000104.png"
			case "missing user":
				request.UserID = 999
			case "script URL":
				request.AvatarURL = "javascript:alert(1)"
			case "overlong URL":
				request.AvatarURL = "https://example.test/" + strings.Repeat("a", 1024)
			}
			if _, err := NewSetAppUserAvatarLogic(context.Background(), s).SetAppUserAvatar(request); err == nil {
				t.Fatal("invalid App avatar mutation accepted")
			}
			var stored model.AppUser
			if err := s.DB.First(&stored, u.ID).Error; err != nil {
				t.Fatal(err)
			}
			var references, audits int64
			if err := s.DB.Model(&model.SysFileReference{}).Count(&references).Error; err != nil {
				t.Fatal(err)
			}
			if err := s.DB.Model(&model.SysAuditLog{}).Count(&audits).Error; err != nil {
				t.Fatal(err)
			}
			if stored.AvatarURL != "" || references != 0 || audits != 0 {
				t.Fatalf("rejected mutation changed avatar/reference/audit: %+v refs=%d audits=%d", stored, references, audits)
			}
		})
	}
}

func TestAppAvatarDefaultIsEmptyAndAdminDefaultMarkerClears(t *testing.T) {
	s, actor, first, _ := avatarFixture(t)
	u, token := seedActive(t, s, 7)
	ctx := context.Background()
	profile, err := NewGetAppProfileLogic(ctx, s).GetAppProfile(&pb.AppAccessRequest{AccessToken: token})
	if err != nil || profile.AvatarURL != "" {
		t.Fatalf("new App users must use the client's default avatar: %+v %v", profile, err)
	}
	set := NewSetAppUserAvatarLogic(ctx, s)
	if _, err := set.SetAppUserAvatar(&pb.AppAvatarRequest{Actor: actor, UserID: u.ID, AvatarURL: first}); err != nil {
		t.Fatal(err)
	}
	if _, err := set.SetAppUserAvatar(&pb.AppAvatarRequest{Actor: actor, UserID: u.ID, AvatarURL: "/default-avatar.svg"}); err != nil {
		t.Fatal(err)
	}
	profile, err = NewGetAppProfileLogic(ctx, s).GetAppProfile(&pb.AppAccessRequest{AccessToken: token})
	if err != nil || profile.AvatarURL != "" || avatarReferenceCount(t, s, u.ID, 101) != 0 {
		t.Fatalf("admin default marker was stored or left a reference: %+v %v", profile, err)
	}
}

func TestAppAvatarAuditFailureRollsBackProfileAndReferences(t *testing.T) {
	s, actor, first, second := avatarFixture(t)
	u, _ := seedActive(t, s, 7)
	logic := NewSetAppUserAvatarLogic(context.Background(), s)
	if _, err := logic.SetAppUserAvatar(&pb.AppAvatarRequest{Actor: actor, UserID: u.ID, AvatarURL: first}); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Exec("CREATE TRIGGER reject_app_avatar_audit BEFORE INSERT ON sys_audit_logs BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := logic.SetAppUserAvatar(&pb.AppAvatarRequest{Actor: actor, UserID: u.ID, AvatarURL: second}); err == nil {
		t.Fatal("audit failure reported success")
	}
	var stored model.AppUser
	if err := s.DB.First(&stored, u.ID).Error; err != nil {
		t.Fatal(err)
	}
	if stored.AvatarURL != first || avatarReferenceCount(t, s, u.ID, 101) != 1 || avatarReferenceCount(t, s, u.ID, 102) != 0 {
		t.Fatal("audit rollback did not preserve the old avatar and its reference")
	}
}

func TestAppAvatarSurvivesLogoutAndDisable(t *testing.T) {
	for _, operation := range []string{"logout", "disable"} {
		t.Run(operation, func(t *testing.T) {
			s, actor, first, _ := avatarFixture(t)
			u, token := seedActive(t, s, 7)
			ctx := context.Background()
			if _, err := NewSetAppUserAvatarLogic(ctx, s).SetAppUserAvatar(&pb.AppAvatarRequest{Actor: actor, UserID: u.ID, AvatarURL: first}); err != nil {
				t.Fatal(err)
			}
			var err error
			if operation == "logout" {
				_, err = NewLogoutAppAccountLogic(ctx, s).LogoutAppAccount(&pb.AppLogoutRequest{AccessToken: token})
			} else {
				_, err = NewSetAppUserStatusLogic(ctx, s).SetAppUserStatus(&pb.AppStatusRequest{Actor: actor, UserID: u.ID, Status: "disabled", Reason: "review"})
			}
			if err != nil {
				t.Fatal(err)
			}
			var stored model.AppUser
			if err := s.DB.First(&stored, u.ID).Error; err != nil {
				t.Fatal(err)
			}
			if stored.AvatarURL != first || avatarReferenceCount(t, s, u.ID, 101) != 1 {
				t.Fatal("session/status mutation unpinned an avatar still in use")
			}
		})
	}
}
