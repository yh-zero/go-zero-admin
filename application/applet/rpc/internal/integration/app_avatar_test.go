package integration

import (
	"context"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	cm "github.com/casbin/casbin/v2/model"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/appauth"
	"go-zero-admin/pkg/objectstorage"
)

func TestAppAvatarRPCSeparatesAccountDomainsAndPreservesResponseField(t *testing.T) {
	f := fixture(t)
	if err := f.db.AutoMigrate(&model.AppUser{}, &model.AppUserIdentity{}, &model.AppSession{}); err != nil {
		t.Fatal(err)
	}
	policy, err := cm.NewModelFromString("[request_definition]\nr = sub, obj, act\n[policy_definition]\np = sub, obj, act\n[policy_effect]\ne = some(where (p.eft == allow))\n[matchers]\nm = r.sub == p.sub && r.obj == p.obj && r.act == p.act")
	if err != nil {
		t.Fatal(err)
	}
	f.service.Casbin, err = casbin.NewSyncedCachedEnforcer(policy)
	if err != nil {
		t.Fatal(err)
	}
	f.service.Casbin.EnableCache(false)
	if _, err = f.service.Casbin.AddPolicy("1", "/v1/sys/app-members/avatar", "PUT"); err != nil {
		t.Fatal(err)
	}
	f.service.Config.Storage = objectstorage.Config{ExternalEndpoint: "https://files.example.test", PublicBucket: "gozero-public", PrivateBucket: "gozero-private"}
	f.service.Config.AppJwtAuth = appauth.Config{AccessSecret: "integration-app-avatar-secret-0123456789", AccessExpire: 900, RefreshExpire: 2592000}
	appUser := model.AppUser{ID: f.admin.UserID, Nickname: "App member", Status: "active", SessionVersion: 1}
	if err := f.db.Create(&appUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Create(&model.AppUserIdentity{UserID: appUser.ID, Kind: "email", Identifier: "app-avatar@example.test", PasswordHash: "unused"}).Error; err != nil {
		t.Fatal(err)
	}
	session := model.AppSession{ID: "00000000-0000-4000-8000-000000000130", UserID: appUser.ID, SessionVersion: 1, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	if err := f.db.Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	const key = "go-zero-admin/00000000-0000-4000-8000-000000000101.png"
	file := model.SysFileResource{ID: 101, ObjectKey: key, Name: "avatar.png", Mime: "image/png", Size: 20, OwnerID: f.admin.UserID, Visibility: "public", Status: "active"}
	if err := f.db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	token, err := appauth.Issue(f.service.Config.AppJwtAuth, appUser.ID, session.ID, 1, 1, "access", time.Now().UTC().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	const avatarURL = "https://files.example.test/gozero-public/" + key
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client := pb.NewAppAccountServiceClient(f.conn)
	if _, err := client.SetAppUserAvatar(ctx, &pb.AppAvatarRequest{Actor: f.admin, UserID: appUser.ID, AvatarURL: avatarURL}); err != nil {
		t.Fatal(err)
	}
	profile, err := client.GetAppProfile(ctx, &pb.AppAccessRequest{AccessToken: token})
	if err != nil || profile.AvatarURL != avatarURL {
		t.Fatalf("avatar lost across the generated RPC boundary: %+v %v", profile, err)
	}
	var admin model.SysUser
	if err := f.db.First(&admin, f.admin.UserID).Error; err != nil {
		t.Fatal(err)
	}
	if admin.HeaderImg != "/default-avatar.svg" {
		t.Fatal("App avatar changed the same-numbered administrator")
	}
	var reference model.SysFileReference
	if err := f.db.Where("file_id = ?", file.ID).First(&reference).Error; err != nil {
		t.Fatal(err)
	}
	if reference.ObjectType != "app_user_avatar" || reference.ObjectID != "30" {
		t.Fatalf("App avatar crossed reference namespaces: %+v", reference)
	}
}
