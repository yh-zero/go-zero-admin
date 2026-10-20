package userlogic

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	fileslogic "go-zero-admin/application/applet/rpc/internal/logic/fileresourceservice"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/objectstorage"
	"gorm.io/gorm/clause"
)

func sysAvatarFixture(t *testing.T) (*svc.ServiceContext, *model.SysUser) {
	t.Helper()
	service := testUserDB(t)
	service.Config.Storage = objectstorage.Config{Endpoint: "http://rustfs:9000", ExternalEndpoint: "http://127.0.0.1:9000", Region: "us-east-1", AccessKeyID: "test-access", SecretAccessKey: "test-secret", PublicBucket: "gozero-public", PrivateBucket: "gozero-private"}
	if err := service.DB.AutoMigrate(&model.SysDepartment{}, &model.SysRoleDataScope{}, &model.SysRoleScopeDepartment{}); err != nil {
		t.Fatal(err)
	}
	return service, sessionUser(t, service, "uploader", 801)
}

func sysAvatarFile(t *testing.T, service *svc.ServiceContext, ownerID int64, sequence int, visibility, status string) (model.SysFileResource, string) {
	t.Helper()
	file := model.SysFileResource{ObjectKey: fmt.Sprintf("go-zero-admin/10000000-0000-4000-8000-%012d.png", sequence), Name: "avatar.png", Mime: "image/png", Size: 100, OwnerID: ownerID, Visibility: visibility, Status: status}
	if err := service.DB.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	bucket := service.Config.Storage.PublicBucket
	if visibility == "private" {
		bucket = service.Config.Storage.PrivateBucket
	}
	return file, service.Config.Storage.ExternalEndpoint + "/" + bucket + "/" + file.ObjectKey
}

func registerAvatarUser(t *testing.T, service *svc.ServiceContext, name, avatar string) *model.SysUser {
	t.Helper()
	if _, err := NewRegisterLogic(context.Background(), service).Register(&pb.RegisterRequest{UserInfo: &pb.UserInfo{Username: name, Password: "before123", HeaderImg: avatar, AuthorityId: 801, Enable: 1}, AuthorityIds: []int64{801}}); err != nil {
		t.Fatal(err)
	}
	var user model.SysUser
	if err := service.DB.Where("username = ?", name).First(&user).Error; err != nil {
		t.Fatal(err)
	}
	return &user
}

func assertSysAvatarReferences(t *testing.T, service *svc.ServiceContext, fileID, want int64) {
	t.Helper()
	var count int64
	if err := service.DB.Model(&model.SysFileReference{}).Where("file_id = ? AND object_type = ?", fileID, "sys_user_avatar").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("file %d has %d user avatar references, want %d", fileID, count, want)
	}
}

func updateSysAvatar(service *svc.ServiceContext, id int64, avatar string) error {
	_, err := NewUpdateUserInfoLogic(context.Background(), service).UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: id, HeaderImg: avatar}, UpdateFields: []string{"headerImg"}})
	return err
}

func TestSysAvatarCreationProtectsFileDeletion(t *testing.T) {
	service, uploader := sysAvatarFixture(t)
	file, avatar := sysAvatarFile(t, service, uploader.ID, 1, "public", "active")
	user := registerAvatarUser(t, service, "uses_avatar", avatar)
	assertSysAvatarReferences(t, service, file.ID, 1)
	var reference model.SysFileReference
	if err := service.DB.Where("file_id = ? AND object_type = ?", file.ID, "sys_user_avatar").First(&reference).Error; err != nil {
		t.Fatal(err)
	}
	if reference.ObjectID != strconv.FormatInt(user.ID, 10) {
		t.Fatalf("reference bound to wrong user: %+v", reference)
	}
	actor := &pb.SessionRequest{UserID: uploader.ID, AuthorityId: uploader.AuthorityId, SessionVersion: uploader.SessionVersion}
	if _, err := fileslogic.NewBeginDeleteFileLogic(context.Background(), service).BeginDeleteFile(&pb.FileIDRequest{Actor: actor, ID: file.ID}); err == nil {
		t.Fatal("avatar in use entered file deletion")
	}
	if err := service.DB.First(&file, file.ID).Error; err != nil || file.Status != "active" {
		t.Fatalf("protected file status changed: %s %v", file.Status, err)
	}
}

func TestSysAvatarReplacementClearOmissionAndSharedUse(t *testing.T) {
	service, uploader := sysAvatarFixture(t)
	first, firstURL := sysAvatarFile(t, service, uploader.ID, 1, "public", "active")
	second, secondURL := sysAvatarFile(t, service, uploader.ID, 2, "public", "active")
	one := registerAvatarUser(t, service, "first_user", firstURL)
	two := registerAvatarUser(t, service, "second_user", firstURL)
	assertSysAvatarReferences(t, service, first.ID, 2)
	for i := 0; i < 2; i++ {
		if err := updateSysAvatar(service, one.ID, secondURL); err != nil {
			t.Fatal(err)
		}
	}
	assertSysAvatarReferences(t, service, first.ID, 1)
	assertSysAvatarReferences(t, service, second.ID, 1)
	if _, err := NewUpdateUserInfoLogic(context.Background(), service).UpdateUserInfo(&pb.UpdateUserInfoRequest{UserInfo: &pb.UserInfo{ID: one.ID, NickName: "renamed", HeaderImg: firstURL}, UpdateFields: []string{"nickName"}}); err != nil {
		t.Fatal(err)
	}
	if err := service.DB.First(one, one.ID).Error; err != nil || one.HeaderImg != secondURL {
		t.Fatalf("omitted avatar was changed: %q %v", one.HeaderImg, err)
	}
	assertSysAvatarReferences(t, service, second.ID, 1)
	if _, err := NewLogoutLogic(context.Background(), service).Logout(&pb.SessionRequest{UserID: two.ID, AuthorityId: two.AuthorityId, SessionVersion: two.SessionVersion}); err != nil {
		t.Fatal(err)
	}
	assertSysAvatarReferences(t, service, first.ID, 1)
	if err := updateSysAvatar(service, one.ID, ""); err != nil {
		t.Fatal(err)
	}
	assertSysAvatarReferences(t, service, second.ID, 0)
	if _, err := NewDeleteUserLogic(context.Background(), service).DeleteUser(&pb.DeleteUserRequest{UserID: two.ID}); err != nil {
		t.Fatal(err)
	}
	assertSysAvatarReferences(t, service, first.ID, 0)
	if err := service.DB.Unscoped().First(two, two.ID).Error; err != nil || !two.DeletedAt.Valid || two.HeaderImg != "" {
		t.Fatalf("deleted user retained avatar: %+v %v", two, err)
	}
	var files int64
	if err := service.DB.Model(&model.SysFileResource{}).Count(&files).Error; err != nil || files != 2 {
		t.Fatalf("avatar release removed file records: %d %v", files, err)
	}
}

func TestSysAvatarRejectsUnavailableManagedFiles(t *testing.T) {
	for _, state := range []struct{ visibility, status string }{{"private", "active"}, {"public", "deleting"}, {"public", "deleted"}} {
		t.Run(state.visibility+"_"+state.status, func(t *testing.T) {
			service, uploader := sysAvatarFixture(t)
			_, avatar := sysAvatarFile(t, service, uploader.ID, 1, state.visibility, state.status)
			if _, err := NewRegisterLogic(context.Background(), service).Register(&pb.RegisterRequest{UserInfo: &pb.UserInfo{Username: "invalid_avatar", Password: "before123", HeaderImg: avatar, AuthorityId: 801}, AuthorityIds: []int64{801}}); err == nil {
				t.Fatal("unavailable avatar accepted during registration")
			}
			var count int64
			if err := service.DB.Model(&model.SysUser{}).Where("username = ?", "invalid_avatar").Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("rejected avatar left new user: %d %v", count, err)
			}
			user := sessionUser(t, service, "preserved_user", 801)
			previous := user.HeaderImg
			if err := updateSysAvatar(service, user.ID, avatar); err == nil {
				t.Fatal("unavailable avatar accepted during update")
			}
			if err := service.DB.First(user, user.ID).Error; err != nil || user.HeaderImg != previous {
				t.Fatalf("rejected update changed avatar: %q %v", user.HeaderImg, err)
			}
		})
	}
}

func TestSysAvatarExternalAndDefaultReplacementReleaseManagedReference(t *testing.T) {
	for _, next := range []string{"/default-avatar.svg", "https://custom.example.com/avatar.png"} {
		t.Run(next, func(t *testing.T) {
			service, uploader := sysAvatarFixture(t)
			file, avatar := sysAvatarFile(t, service, uploader.ID, 1, "public", "active")
			user := registerAvatarUser(t, service, "legacy_avatar", avatar)
			if err := service.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysFileReference{FileID: file.ID, ObjectType: "sys_user_avatar", ObjectID: strconv.FormatInt(user.ID, 10)}).Error; err != nil {
				t.Fatal(err)
			}
			if err := updateSysAvatar(service, user.ID, next); err != nil {
				t.Fatal(err)
			}
			assertSysAvatarReferences(t, service, file.ID, 0)
			if err := service.DB.First(user, user.ID).Error; err != nil || user.HeaderImg != next {
				t.Fatalf("compatible avatar changed: %q %v", user.HeaderImg, err)
			}
		})
	}
}

func TestSysAvatarAuditFailureRollsBackUserAndReferences(t *testing.T) {
	for _, action := range []string{"register", "updateUser", "deleteUser"} {
		t.Run(action, func(t *testing.T) {
			service, uploader := sysAvatarFixture(t)
			first, firstURL := sysAvatarFile(t, service, uploader.ID, 1, "public", "active")
			second, secondURL := sysAvatarFile(t, service, uploader.ID, 2, "public", "active")
			var user *model.SysUser
			if action != "register" {
				user = registerAvatarUser(t, service, "existing_user", firstURL)
				// Existing references may predate this write; retain them on rollback.
				if err := service.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&model.SysFileReference{FileID: first.ID, ObjectType: "sys_user_avatar", ObjectID: strconv.FormatInt(user.ID, 10)}).Error; err != nil {
					t.Fatal(err)
				}
			}
			if err := service.DB.Exec("CREATE TRIGGER reject_avatar_audit BEFORE INSERT ON sys_audit_logs WHEN NEW.module = 'user' AND NEW.action = '" + action + "' BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END").Error; err != nil {
				t.Fatal(err)
			}
			var err error
			switch action {
			case "register":
				_, err = NewRegisterLogic(context.Background(), service).Register(&pb.RegisterRequest{UserInfo: &pb.UserInfo{Username: "failed_user", Password: "before123", HeaderImg: secondURL, AuthorityId: 801}, AuthorityIds: []int64{801}})
			case "updateUser":
				err = updateSysAvatar(service, user.ID, secondURL)
			case "deleteUser":
				_, err = NewDeleteUserLogic(context.Background(), service).DeleteUser(&pb.DeleteUserRequest{UserID: user.ID})
			}
			if err == nil {
				t.Fatal("failed avatar audit was ignored")
			}
			assertSysAvatarReferences(t, service, second.ID, 0)
			if action == "register" {
				var count int64
				if err := service.DB.Model(&model.SysUser{}).Where("username = ?", "failed_user").Count(&count).Error; err != nil || count != 0 {
					t.Fatalf("audit failure left new user: %d %v", count, err)
				}
			} else {
				assertSysAvatarReferences(t, service, first.ID, 1)
				if err := service.DB.First(user, user.ID).Error; err != nil || user.HeaderImg != firstURL || user.DeletedAt.Valid {
					t.Fatalf("audit failure changed user: %+v %v", user, err)
				}
			}
		})
	}
}

func TestSysAvatarURLAliasesCannotBypassFileDeletionProtection(t *testing.T) {
	for _, variant := range []struct {
		name, prefix string
		mayReject    bool
	}{
		{"protocol_relative", "//127.0.0.1:9000/gozero-public/", false},
		{"uppercase_scheme", "HTTP://127.0.0.1:9000/gozero-public/", true},
		{"opaque_http", "http:127.0.0.1:9000/gozero-public/", true},
	} {
		t.Run(variant.name, func(t *testing.T) {
			service, uploader := sysAvatarFixture(t)
			file, _ := sysAvatarFile(t, service, uploader.ID, 1, "public", "active")
			user := sessionUser(t, service, "alias_user", 801)
			previous := user.HeaderImg
			err := updateSysAvatar(service, user.ID, variant.prefix+file.ObjectKey)
			if err != nil {
				if !variant.mayReject {
					t.Fatalf("supported avatar alias rejected: %v", err)
				}
				if err := service.DB.First(user, user.ID).Error; err != nil || user.HeaderImg != previous {
					t.Fatalf("rejected alias changed user avatar: %q %v", user.HeaderImg, err)
				}
				assertSysAvatarReferences(t, service, file.ID, 0)
				return
			}
			assertSysAvatarReferences(t, service, file.ID, 1)
			actor := &pb.SessionRequest{UserID: uploader.ID, AuthorityId: uploader.AuthorityId, SessionVersion: uploader.SessionVersion}
			if _, err := fileslogic.NewBeginDeleteFileLogic(context.Background(), service).BeginDeleteFile(&pb.FileIDRequest{Actor: actor, ID: file.ID}); err == nil {
				t.Fatal("accepted avatar alias bypassed file deletion protection")
			}
			if err := service.DB.First(&file, file.ID).Error; err != nil || file.Status != "active" {
				t.Fatalf("in-use alias file entered deletion: %s %v", file.Status, err)
			}
		})
	}
}
