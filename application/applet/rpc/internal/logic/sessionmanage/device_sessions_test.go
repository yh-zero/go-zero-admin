package sessionmanagelogic

import (
	"context"
	"fmt"
	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v4"
	userlogic "go-zero-admin/application/applet/rpc/internal/logic/user"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/ctxJwt"
	"go-zero-admin/pkg/orm"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"testing"
	"time"
)

func deviceDB(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if err = db.AutoMigrate(&model.SysAuthority{}, &model.SysUser{}, &model.SysUserAuthority{}, &model.SysDeviceSession{}, &audit.Event{}); err != nil {
		t.Fatal(err)
	}
	for _, role := range []int64{1, 801} {
		if err = db.Create(&model.SysAuthority{AuthorityId: role, AuthorityName: fmt.Sprint(role)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for id := int64(1); id <= 3; id++ {
		role := int64(801)
		if id == 1 {
			role = 1
		}
		account := model.SysUser{Username: fmt.Sprintf("device-user-%d", id), AuthorityId: role, Enable: 1, SessionVersion: 1}
		account.ID = id
		if err = db.Create(&account).Error; err != nil {
			t.Fatal(err)
		}
		if err = db.Create(&model.SysUserAuthority{SysUserId: id, SysAuthorityAuthorityId: role}).Error; err != nil {
			t.Fatal(err)
		}
	}
	result := &svc.ServiceContext{DB: &orm.DB{DB: db}}
	result.Config.JwtAuth.AccessSecret = "isolated-device-test-secret"
	result.Config.JwtAuth.AccessExpire = 3600
	return result
}

func issue(t *testing.T, s *svc.ServiceContext, id, role int64) *pb.SessionRequest {
	t.Helper()
	response, err := userlogic.NewGetUserTokeLogic(context.Background(), s).GetUserToke(&pb.GetUserTokeRequest{ID: id, AuthorityId: role, SessionVersion: 1, Username: "forged", IP: "127.0.0.1", UserAgent: "device-test"})
	if err != nil {
		t.Fatal(err)
	}
	token, err := jwt.Parse(response.Token, func(token *jwt.Token) (interface{}, error) { return []byte(s.Config.JwtAuth.AccessSecret), nil }, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		t.Fatal("invalid issued JWT", err)
	}
	claims := token.Claims.(jwt.MapClaims)
	identity := ctxJwt.GetJwtData(context.WithValue(context.Background(), ctxJwt.CtxKeyJwtData, claims[ctxJwt.CtxKeyJwtData]))
	if identity.SessionID == "" || identity.Username == "forged" {
		t.Fatal("missing device ID or caller profile trusted", identity)
	}
	return &pb.SessionRequest{UserID: identity.ID, AuthorityId: identity.AuthorityId, SessionVersion: identity.SessionVersion, SessionID: identity.SessionID}
}

func valid(t *testing.T, s *svc.ServiceContext, actor *pb.SessionRequest, want bool) {
	t.Helper()
	result, err := userlogic.NewCheckSessionLogic(context.Background(), s).CheckSession(actor)
	if err != nil || result.Valid != want {
		t.Fatalf("want valid %v, got %+v %v", want, result, err)
	}
}
func list(t *testing.T, s *svc.ServiceContext, actor *pb.SessionRequest, user int64) *pb.DeviceSessionListResponse {
	t.Helper()
	response, err := NewGetDeviceSessionsLogic(context.Background(), s).GetDeviceSessions(&pb.DeviceSessionListRequest{Actor: actor, UserID: user, PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}})
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func TestDeviceRevocationAndGlobalLogout(t *testing.T) {
	s := deviceDB(t)
	first := issue(t, s, 2, 801)
	second := issue(t, s, 2, 801)
	result := list(t, s, first, 2)
	if result.Total != 2 {
		t.Fatal("distinct devices not listed", result)
	}
	current := 0
	for _, row := range result.List {
		if row.Current {
			current++
		}
		if row.IP != "127.0.0.1" || row.UserAgent != "device-test" {
			t.Fatal(row)
		}
	}
	if current != 1 {
		t.Fatal("current device not identified")
	}
	revoke := NewRevokeDeviceSessionLogic(context.Background(), s)
	if _, err := revoke.RevokeDeviceSession(&pb.RevokeDeviceSessionRequest{Actor: second, ID: first.SessionID}); err != nil {
		t.Fatal(err)
	}
	valid(t, s, first, false)
	valid(t, s, second, true)
	if list(t, s, second, 2).Total != 1 {
		t.Fatal("revoked device listed")
	}
	if _, err := revoke.RevokeDeviceSession(&pb.RevokeDeviceSessionRequest{Actor: second, ID: first.SessionID}); err != nil {
		t.Fatal("revocation not idempotent", err)
	}
	legacy := &pb.SessionRequest{UserID: 2, AuthorityId: 801, SessionVersion: 1}
	valid(t, s, legacy, true)
	if _, err := userlogic.NewLogoutLogic(context.Background(), s).Logout(second); err != nil {
		t.Fatal(err)
	}
	valid(t, s, second, false)
	valid(t, s, legacy, false)
}
func TestDeviceOwnershipAdminAndExpiry(t *testing.T) {
	s := deviceDB(t)
	admin := issue(t, s, 1, 1)
	first := issue(t, s, 2, 801)
	other := issue(t, s, 3, 801)
	logic := NewGetDeviceSessionsLogic(context.Background(), s)
	if _, err := logic.GetDeviceSessions(&pb.DeviceSessionListRequest{Actor: first, UserID: 3, PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}}); err == nil {
		t.Fatal("other user's devices leaked")
	}
	revoke := NewRevokeDeviceSessionLogic(context.Background(), s)
	if _, err := revoke.RevokeDeviceSession(&pb.RevokeDeviceSessionRequest{Actor: first, ID: other.SessionID}); err == nil {
		t.Fatal("other user's device revoked")
	}
	if _, err := revoke.RevokeDeviceSession(&pb.RevokeDeviceSessionRequest{Actor: admin, ID: other.SessionID, SelfOnly: true}); err == nil {
		t.Fatal("personal route escalated admin visibility")
	}
	if list(t, s, admin, 0).Total != 3 {
		t.Fatal("admin cannot list all devices")
	}
	if _, err := revoke.RevokeDeviceSession(&pb.RevokeDeviceSessionRequest{Actor: admin, ID: other.SessionID}); err != nil {
		t.Fatal(err)
	}
	valid(t, s, other, false)
	if err := s.DB.Model(&model.SysDeviceSession{}).Where("id=?", first.SessionID).Update("expires_at", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	valid(t, s, first, false)
	if list(t, s, admin, 0).Total != 1 {
		t.Fatal("expired device listed")
	}
	if _, err := logic.GetDeviceSessions(&pb.DeviceSessionListRequest{Actor: admin, PageRequest: &pb.PageRequest{PageNo: 9223372036854775807, PageSize: 500}}); err == nil {
		t.Fatal("overflow accepted")
	}
}
func TestDeviceAuditingFailureRollsBack(t *testing.T) {
	s := deviceDB(t)
	first := issue(t, s, 2, 801)
	second := issue(t, s, 2, 801)
	if err := s.DB.Exec("CREATE TRIGGER reject_device_audit BEFORE INSERT ON sys_audit_logs BEGIN SELECT RAISE(FAIL,'audit unavailable'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewRevokeDeviceSessionLogic(context.Background(), s).RevokeDeviceSession(&pb.RevokeDeviceSessionRequest{Actor: second, ID: first.SessionID}); err == nil {
		t.Fatal("missing audit accepted")
	}
	valid(t, s, first, true)
	if _, err := userlogic.NewLogoutLogic(context.Background(), s).Logout(second); err == nil {
		t.Fatal("global logout committed without audit")
	}
	valid(t, s, second, true)
	if _, err := userlogic.NewGetUserTokeLogic(context.Background(), s).GetUserToke(&pb.GetUserTokeRequest{ID: 2, AuthorityId: 801, SessionVersion: 1}); err == nil {
		t.Fatal("token issued without durable session audit")
	}
	if list(t, s, second, 2).Total != 2 {
		t.Fatal("failed token transaction retained device")
	}
}
