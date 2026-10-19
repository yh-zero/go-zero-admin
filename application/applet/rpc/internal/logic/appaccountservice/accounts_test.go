package appaccountservicelogic

import (
	"context"
	"github.com/casbin/casbin/v2"
	cm "github.com/casbin/casbin/v2/model"
	"github.com/glebarez/sqlite"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/appauth"
	base "go-zero-admin/pkg/model"
	"go-zero-admin/pkg/orm"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"strings"
	"testing"
	"time"
)

func accountFixture(t *testing.T, credentials bool) (*svc.ServiceContext, *pb.SessionRequest) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sql.Close() })
	return accountFixtureWithDB(t, credentials, db)
}

func accountFixtureWithDB(t *testing.T, credentials bool, db *gorm.DB) (*svc.ServiceContext, *pb.SessionRequest) {
	t.Helper()
	if err := db.AutoMigrate(&model.AppUser{}, &model.AppUserIdentity{}, &model.AppSession{}, &model.SysUser{}, &model.SysAuthority{}, &model.SysUserAuthority{}, &model.SysDeviceSession{}, &model.SysAuditLog{}); err != nil {
		t.Fatal(err)
	}
	m, err := cm.NewModelFromString("[request_definition]\nr = sub, obj, act\n[policy_definition]\np = sub, obj, act\n[policy_effect]\ne = some(where (p.eft == allow))\n[matchers]\nm = r.sub == p.sub && r.obj == p.obj && r.act == p.act")
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := casbin.NewSyncedCachedEnforcer(m)
	if err != nil {
		t.Fatal(err)
	}
	enforcer.EnableCache(false)
	for _, p := range [][]string{{"1", "/v1/sys/app-members/users", "GET"}, {"1", "/v1/sys/app-members/status", "PUT"}, {"1", "/v1/sys/app-members/sessions", "GET"}, {"1", "/v1/sys/app-members/session", "DELETE"}} {
		if _, err = enforcer.AddPolicy(p); err != nil {
			t.Fatal(err)
		}
	}
	for _, row := range []any{&model.SysAuthority{AuthorityId: 1, AuthorityName: "admin"}, &model.SysUser{MODEL_BASE: base.MODEL_BASE{ID: 99}, Username: "admin-test", AuthorityId: 1, Enable: 1, SessionVersion: 1}, &model.SysUserAuthority{SysUserId: 99, SysAuthorityAuthorityId: 1}, &model.SysDeviceSession{ID: "admin-device", UserID: 99, AuthorityID: 1, SessionVersion: 1, ExpiresAt: time.Now().UTC().Add(time.Hour)}} {
		if err = db.Omit("Authority", "Authorities").Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	s := &svc.ServiceContext{DB: &orm.DB{DB: db}, Casbin: enforcer}
	s.Config.AppJwtAuth = appauth.Config{AccessSecret: "test-app-secret-01234567890123456789", AccessExpire: 900, RefreshExpire: 2592000}
	if credentials {
		addr := os.Getenv("APP_TEST_REDIS_ADDR")
		if addr == "" {
			t.Skip("APP_TEST_REDIS_ADDR not set; credential integration requires Redis")
		}
		s.BizRedis = redis.New(addr, redis.WithPass(os.Getenv("APP_TEST_REDIS_PASSWORD")))
	}
	return s, &pb.SessionRequest{UserID: 99, SessionVersion: 1, AuthorityId: 1, SessionID: "admin-device"}
}
func seedActive(t *testing.T, s *svc.ServiceContext, id int64) (*model.AppUser, string) {
	t.Helper()
	u := &model.AppUser{ID: id, Nickname: "member", Status: "active", SessionVersion: 1}
	if err := s.DB.Create(u).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Create(&model.AppUserIdentity{UserID: id, Kind: "email", Identifier: strings.ToLower(t.Name()) + string(rune('a'+id)) + "@example.com", PasswordHash: "unused"}).Error; err != nil {
		t.Fatal(err)
	}
	ss := model.AppSession{ID: strings.Repeat(string(rune('a'+id)), 36), UserID: id, SessionVersion: 1, RefreshGeneration: 1, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	if err := s.DB.Create(&ss).Error; err != nil {
		t.Fatal(err)
	}
	token, err := appauth.Issue(s.Config.AppJwtAuth, id, ss.ID, 1, 1, "access", time.Now().UTC().Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	return u, token
}
func TestRegistrationReviewLoginAndDuplicate(t *testing.T) {
	s, a := accountFixture(t, true)
	ctx := context.Background()
	account := strings.ToLower(t.Name()) + time.Now().Format("150405.000000000") + "@example.com"
	ip := account
	registration := NewRegisterAppAccountLogic(ctx, s)
	u, err := registration.RegisterAppAccount(&pb.AppRegisterRequest{Account: strings.ToUpper(account), Password: "Password123!", Nickname: " new user ", IP: ip})
	if err != nil {
		t.Fatal(err)
	}
	if u.Status != "pending" || u.Account != account || u.Nickname != "new user" {
		t.Fatalf("unexpected registration: %+v", u)
	}
	if _, err = registration.RegisterAppAccount(&pb.AppRegisterRequest{Account: account, Password: "Password123!", Nickname: "other", IP: ip}); err == nil {
		t.Fatal("duplicate identity accepted")
	}
	login := NewLoginAppAccountLogic(ctx, s)
	request := &pb.AppLoginRequest{Account: account, Password: "Password123!", IP: ip}
	if _, err = login.LoginAppAccount(request); err == nil {
		t.Fatal("pending user logged in")
	}
	if _, err = NewSetAppUserStatusLogic(ctx, s).SetAppUserStatus(&pb.AppStatusRequest{Actor: a, UserID: u.ID, Status: "active"}); err != nil {
		t.Fatal(err)
	}
	tokens, err := login.LoginAppAccount(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = ValidateAccess(ctx, s, tokens.AccessToken); err != nil {
		t.Fatal(err)
	}
	var identity model.AppUserIdentity
	s.DB.First(&identity)
	if identity.PasswordHash == request.Password || !strings.HasPrefix(identity.PasswordHash, "$2") {
		t.Fatal("password not bcrypt hashed")
	}
	if _, err = NewSetAppUserStatusLogic(ctx, s).SetAppUserStatus(&pb.AppStatusRequest{Actor: a, UserID: u.ID, Status: "disabled", Reason: "disabled test"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = ValidateAccess(ctx, s, tokens.AccessToken); err == nil {
		t.Fatal("disabled user's old token accepted")
	}
	if _, err = NewSetAppUserStatusLogic(ctx, s).SetAppUserStatus(&pb.AppStatusRequest{Actor: a, UserID: u.ID, Status: "active"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = ValidateAccess(ctx, s, tokens.AccessToken); err == nil {
		t.Fatal("restoration resurrected old token")
	}
	if _, err = login.LoginAppAccount(request); err != nil {
		t.Fatal(err)
	}
}
func TestRefreshRotationRejectsReplayAndRevokesSession(t *testing.T) {
	s, _ := accountFixture(t, false)
	ctx := context.Background()
	u, _ := seedActive(t, s, 1)
	var session model.AppSession
	s.DB.Where("user_id = ?", u.ID).First(&session)
	first, err := appauth.Issue(s.Config.AppJwtAuth, u.ID, session.ID, 1, 1, "refresh", session.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
	s.DB.Model(&session).Update("refresh_hash", appauth.Digest(first))
	refresh := NewRefreshAppTokenLogic(ctx, s)
	second, err := refresh.RefreshAppToken(&pb.AppRefreshRequest{RefreshToken: first})
	if err != nil {
		t.Fatal(err)
	}
	if second.RefreshToken == first {
		t.Fatal("refresh token did not rotate")
	}
	third, err := refresh.RefreshAppToken(&pb.AppRefreshRequest{RefreshToken: second.RefreshToken})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = refresh.RefreshAppToken(&pb.AppRefreshRequest{RefreshToken: first}); err == nil {
		t.Fatal("historic refresh replay accepted")
	}
	if _, _, err = ValidateAccess(ctx, s, third.AccessToken); err == nil {
		t.Fatal("replay did not revoke current session")
	}
	if _, err = refresh.RefreshAppToken(&pb.AppRefreshRequest{RefreshToken: third.RefreshToken}); err == nil {
		t.Fatal("revoked refresh accepted")
	}
}
func TestSessionOwnershipLogoutAndProfile(t *testing.T) {
	s, _ := accountFixture(t, false)
	ctx := context.Background()
	u, token := seedActive(t, s, 1)
	_, other := seedActive(t, s, 2)
	otherClaims, _ := appauth.Parse(s.Config.AppJwtAuth, other, "access")
	if _, err := NewRevokeMyAppSessionLogic(ctx, s).RevokeMyAppSession(&pb.AppRevokeSelfRequest{AccessToken: token, ID: otherClaims.SessionID}); err == nil {
		t.Fatal("cross-user revoke allowed")
	}
	profile, err := NewUpdateAppProfileLogic(ctx, s).UpdateAppProfile(&pb.AppProfileRequest{AccessToken: token, Nickname: "Changed"})
	if err != nil || profile.Nickname != "Changed" {
		t.Fatal(profile, err)
	}
	extra := model.AppSession{ID: "second-device", UserID: u.ID, SessionVersion: 1, ExpiresAt: time.Now().UTC().Add(time.Hour)}
	s.DB.Create(&extra)
	sessions, err := NewGetMyAppSessionsLogic(ctx, s).GetMyAppSessions(&pb.AppAccessRequest{AccessToken: token})
	if err != nil || len(sessions.List) != 2 {
		t.Fatal(sessions, err)
	}
	if _, err = NewLogoutAppAccountLogic(ctx, s).LogoutAppAccount(&pb.AppLogoutRequest{AccessToken: token, AllDevices: true}); err != nil {
		t.Fatal(err)
	}
	var count int64
	s.DB.Model(&model.AppSession{}).Where("user_id = ? AND revoked_at IS NULL", u.ID).Count(&count)
	if count != 0 {
		t.Fatal("all-device logout left active sessions")
	}
	if _, _, err = ValidateAccess(ctx, s, other); err != nil {
		t.Fatal("logout affected another user", err)
	}
}
func TestAdminRequiresSessionPermissionAndAtomicAudit(t *testing.T) {
	s, a := accountFixture(t, false)
	ctx := context.Background()
	u, _ := seedActive(t, s, 1)
	logic := NewSetAppUserStatusLogic(ctx, s)
	missing := &pb.SessionRequest{UserID: a.UserID, AuthorityId: a.AuthorityId, SessionVersion: a.SessionVersion}
	if _, err := logic.SetAppUserStatus(&pb.AppStatusRequest{Actor: missing, UserID: u.ID, Status: "disabled", Reason: "test"}); err == nil {
		t.Fatal("missing real session authorized")
	}
	s.Casbin.RemovePolicy("1", "/v1/sys/app-members/status", "PUT")
	if _, err := logic.SetAppUserStatus(&pb.AppStatusRequest{Actor: a, UserID: u.ID, Status: "disabled", Reason: "test"}); err == nil {
		t.Fatal("revoked permission authorized")
	}
	s.Casbin.AddPolicy("1", "/v1/sys/app-members/status", "PUT")
	if _, err := logic.SetAppUserStatus(&pb.AppStatusRequest{Actor: a, UserID: u.ID, Status: "disabled"}); err == nil {
		t.Fatal("disable without reason accepted")
	}
	s.DB.Migrator().DropTable(&model.SysAuditLog{})
	if _, err := logic.SetAppUserStatus(&pb.AppStatusRequest{Actor: a, UserID: u.ID, Status: "disabled", Reason: "test"}); err == nil {
		t.Fatal("audit failure did not fail operation")
	}
	s.DB.First(u, u.ID)
	if u.Status != "active" {
		t.Fatal("audit failure did not roll back status")
	}
}
func TestCredentialRateLimitAndMissingRedisFailClosed(t *testing.T) {
	s, _ := accountFixture(t, false)
	ctx := context.Background()
	req := &pb.AppLoginRequest{Account: "nobody@example.com", Password: "Password123!", IP: "rate-test"}
	if _, err := NewLoginAppAccountLogic(ctx, s).LoginAppAccount(req); err == nil {
		t.Fatal("login allowed without Redis")
	}
	addr := os.Getenv("APP_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("APP_TEST_REDIS_ADDR not set")
	}
	s.BizRedis = redis.New(addr, redis.WithPass(os.Getenv("APP_TEST_REDIS_PASSWORD")))
	key := t.Name() + time.Now().Format("150405.000000000")
	for i := 0; i < 10; i++ {
		if err := credentialLimit(ctx, s, "login", key, key); err != nil {
			t.Fatal("premature limit", err)
		}
	}
	if err := credentialLimit(ctx, s, "login", key, key); err == nil {
		t.Fatal("credential rate limit did not engage")
	}
}

func TestPrivateRPCRejectsRevokedExpiredAndForeignDomainTokens(t *testing.T) {
	s, _ := accountFixture(t, false)
	ctx := context.Background()
	u, token := seedActive(t, s, 1)
	calls := []func(string) error{
		func(raw string) error {
			_, err := NewCheckAppSessionLogic(ctx, s).CheckAppSession(&pb.AppAccessRequest{AccessToken: raw})
			return err
		},
		func(raw string) error {
			_, err := NewGetAppProfileLogic(ctx, s).GetAppProfile(&pb.AppAccessRequest{AccessToken: raw})
			return err
		},
		func(raw string) error {
			_, err := NewUpdateAppProfileLogic(ctx, s).UpdateAppProfile(&pb.AppProfileRequest{AccessToken: raw, Nickname: "new"})
			return err
		},
		func(raw string) error {
			_, err := NewGetMyAppSessionsLogic(ctx, s).GetMyAppSessions(&pb.AppAccessRequest{AccessToken: raw})
			return err
		},
		func(raw string) error {
			_, err := NewRevokeMyAppSessionLogic(ctx, s).RevokeMyAppSession(&pb.AppRevokeSelfRequest{AccessToken: raw, ID: "other"})
			return err
		},
		func(raw string) error {
			_, err := NewLogoutAppAccountLogic(ctx, s).LogoutAppAccount(&pb.AppLogoutRequest{AccessToken: raw})
			return err
		},
	}
	for _, call := range calls {
		if err := call("admin-or-malformed-token"); err == nil {
			t.Fatal("private RPC accepted invalid App token")
		}
	}
	claims, _ := appauth.Parse(s.Config.AppJwtAuth, token, "access")
	s.DB.Model(&model.AppSession{}).Where("id = ?", claims.SessionID).Update("expires_at", time.Now().UTC().Add(-time.Minute))
	for _, call := range calls {
		if err := call(token); err == nil {
			t.Fatal("private RPC accepted expired database session")
		}
	}
	s.DB.Model(&model.AppSession{}).Where("id = ?", claims.SessionID).Updates(map[string]any{"expires_at": time.Now().UTC().Add(time.Hour), "revoked_at": time.Now()})
	for _, call := range calls {
		if err := call(token); err == nil {
			t.Fatal("private RPC accepted revoked database session")
		}
	}
	s.DB.Model(&model.AppSession{}).Where("id = ?", claims.SessionID).Update("revoked_at", nil)
	s.DB.Model(u).Update("session_version", 2)
	for _, call := range calls {
		if err := call(token); err == nil {
			t.Fatal("private RPC accepted old user session version")
		}
	}
}
func TestAdminListFiltersAndSessionRevocationAudit(t *testing.T) {
	s, a := accountFixture(t, false)
	ctx := context.Background()
	u, token := seedActive(t, s, 1)
	seedActive(t, s, 2)
	response, err := NewListAppUsersLogic(ctx, s).ListAppUsers(&pb.AppAdminListRequest{Actor: a, Status: "active", Page: &pb.PageRequest{PageNo: 1, PageSize: 1, Keyword: "member"}})
	if err != nil || response.Total != 2 || len(response.List) != 1 || response.List[0].IdentityKind != "email" || response.List[0].Account == "" {
		t.Fatal(response, err)
	}
	claims, _ := appauth.Parse(s.Config.AppJwtAuth, token, "access")
	if _, err = NewRevokeAppUserSessionLogic(ctx, s).RevokeAppUserSession(&pb.AppAdminRevokeRequest{Actor: a, ID: claims.SessionID}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = ValidateAccess(ctx, s, token); err == nil {
		t.Fatal("admin-revoked session remains valid")
	}
	var event model.SysAuditLog
	if err = s.DB.First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.ActorID != a.UserID || event.ActorName != "admin-test" || !strings.HasPrefix(event.Object, "app-user:") {
		t.Fatal("audit identity/domain incorrect", event.ActorID, event.ActorName, event.Object)
	}
	s.DB.Model(&model.SysDeviceSession{}).Where("id = ?", a.SessionID).Update("revoked_at", time.Now())
	if _, err = NewListAppUserSessionsLogic(ctx, s).ListAppUserSessions(&pb.AppAdminUserRequest{Actor: a, UserID: u.ID}); err == nil {
		t.Fatal("revoked admin session retained access")
	}
}
