package appaccountservicelogic

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
)

func TestEmailRegistrationRequiredSceneIsolationConsumeAndPending(t *testing.T) {
	s, _ := accountFixture(t, true)
	ctx := context.Background()
	s.Config.AppJwtAuth.EmailVerificationRequired = true
	account := "email-" + time.Now().UTC().Format("150405.000000000") + "@example.com"
	request := &pb.AppRegisterRequest{Account: strings.ToUpper(account), Nickname: "email member", Password: "Password123!", IP: account}
	logic := NewRegisterAppAccountLogic(ctx, s)
	if _, err := logic.RegisterAppAccount(request); err == nil {
		t.Fatal("required email registration accepted missing code")
	}
	legacyKey := "biz#emailcode#register#email:" + account
	appKey := "biz#emailcode#app_register#email:" + account
	if err := s.BizRedis.SetexCtx(ctx, legacyKey, "123456", 300); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.BizRedis.Del(legacyKey, appKey) })
	request.EmailCode = "123456"
	if _, err := logic.RegisterAppAccount(request); err == nil {
		t.Fatal("legacy registration scene code authorized App registration")
	}
	if err := s.BizRedis.SetexCtx(ctx, appKey, "654321", 300); err != nil {
		t.Fatal(err)
	}
	request.EmailCode = "000000"
	if _, err := logic.RegisterAppAccount(request); err == nil {
		t.Fatal("wrong code accepted")
	}
	request.EmailCode = "654321"
	u, err := logic.RegisterAppAccount(request)
	if err != nil {
		t.Fatal(err)
	}
	if !u.IdentityVerified || u.Status != "pending" {
		t.Fatal("email verification must retain manual approval", u)
	}
	var identity model.AppUserIdentity
	if err = s.DB.Where("user_id = ?", u.ID).First(&identity).Error; err != nil || identity.VerifiedAt == nil {
		t.Fatal("verification time not stored", err)
	}
	if _, err = logic.RegisterAppAccount(request); err == nil || !strings.Contains(err.Error(), "验证码") {
		t.Fatal("consumed code replay was not rejected by verifier", err)
	}
	if code, err := s.BizRedis.GetCtx(ctx, legacyKey); err != nil || code != "123456" {
		t.Fatal("App verification consumed another scene", err)
	}
}
func TestOptionalEmailCodeStillVerifiedAndPhonesCannotClaimEmailVerification(t *testing.T) {
	s, _ := accountFixture(t, true)
	ctx := context.Background()
	logic := NewRegisterAppAccountLogic(ctx, s)
	request := &pb.AppRegisterRequest{Account: "optional-" + time.Now().UTC().Format("150405.000000000") + "@example.com", Password: "Password123!", Nickname: "optional", EmailCode: "000000", IP: t.Name() + time.Now().UTC().Format("150405.000000000")}
	if _, err := logic.RegisterAppAccount(request); err == nil {
		t.Fatal("optional invalid supplied code ignored")
	}
	request.EmailCode = ""
	u, err := logic.RegisterAppAccount(request)
	if err != nil || u.IdentityVerified || u.Status != "pending" {
		t.Fatal(u, err)
	}
	request.Account = fmt.Sprintf("13%09d", time.Now().UnixNano()%1000000000)
	request.EmailCode = "123456"
	if _, err = logic.RegisterAppAccount(request); err == nil {
		t.Fatal("phone accepted email code")
	}
	request.EmailCode = ""
	s.Config.AppJwtAuth.EmailVerificationRequired = true
	u, err = logic.RegisterAppAccount(request)
	if err != nil || u.IdentityVerified || u.Account != "+86"+request.Account {
		t.Fatal(u, err)
	}
}
func TestEmailOptionsAndSendReuseCooldownWithoutSendingMail(t *testing.T) {
	s, _ := accountFixture(t, true)
	ctx := context.Background()
	opts, err := NewGetAppAuthOptionsLogic(ctx, s).GetAppAuthOptions(&pb.NoDataResponse{})
	if err != nil || opts.EmailCodeEnabled {
		t.Fatal(opts, err)
	}
	if _, err = NewSendAppEmailCodeLogic(ctx, s).SendAppEmailCode(&pb.AppSendEmailCodeRequest{Account: "valid@example.com", IP: "127.0.0.1"}); err == nil {
		t.Fatal("mail unavailable was not reported")
	}
	s.Config.Mail.Host = "127.0.0.1"
	s.Config.Mail.Port = 465
	s.Config.Mail.Username = "test"
	s.Config.Mail.Password = "test"
	s.Config.Mail.From = "test@example.com"
	s.Config.AppJwtAuth.EmailVerificationRequired = true
	opts, err = NewGetAppAuthOptionsLogic(ctx, s).GetAppAuthOptions(&pb.NoDataResponse{})
	if err != nil || !opts.EmailCodeEnabled || !opts.EmailVerificationRequired {
		t.Fatal(opts, err)
	}
	account := "cooldown-" + time.Now().UTC().Format("150405.000000000") + "@example.com"
	key := "biz#emailcode#app_register#cooldown:" + account
	if err = s.BizRedis.SetexCtx(ctx, key, "1", 60); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.BizRedis.Del(key) })
	if _, err = NewSendAppEmailCodeLogic(ctx, s).SendAppEmailCode(&pb.AppSendEmailCodeRequest{Account: strings.ToUpper(account), IP: "127.0.0.1"}); err == nil || !strings.Contains(err.Error(), "频繁") {
		t.Fatal("App send bypassed reusable mail cooldown", err)
	}
	if _, err = NewSendAppEmailCodeLogic(ctx, s).SendAppEmailCode(&pb.AppSendEmailCodeRequest{Account: "13800138000", IP: "127.0.0.1"}); err == nil {
		t.Fatal("email sender accepted phone")
	}
}
