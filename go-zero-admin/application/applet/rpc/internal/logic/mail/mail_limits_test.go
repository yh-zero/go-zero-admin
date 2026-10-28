package maillogic

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	redisclient "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/application/applet/rpc/internal/config"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
)

type mailLimitFixture struct {
	logic  *SendEmailCodeLogic
	raw    *redisclient.Client
	prefix string
	stem   string
	calls  atomic.Int64
	mu     sync.Mutex
	ips    map[string]bool
}

func newMailLimitFixture(t *testing.T) *mailLimitFixture {
	t.Helper()
	address := os.Getenv("APP_TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("APP_TEST_REDIS_ADDR is required for real Redis mail limit tests")
	}
	id, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}
	hex := strings.ReplaceAll(id.String(), "-", "")
	f := &mailLimitFixture{prefix: "mailtest-" + hex, stem: fmt.Sprintf("2001:db8:%s:%s:%s:%s:%s:", hex[:4], hex[4:8], hex[8:12], hex[12:16], hex[16:20]), ips: map[string]bool{}}
	f.raw = redisclient.NewClient(&redisclient.Options{Addr: address, Password: os.Getenv("APP_TEST_REDIS_PASSWORD")})
	if err = f.raw.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	s := &svc.ServiceContext{BizRedis: redis.New(address, redis.WithPass(os.Getenv("APP_TEST_REDIS_PASSWORD")))}
	s.Config.Mail = config.MailConfig{Host: "smtp.example.test", Port: 465, Username: "test", Password: "test", From: "test@example.test"}
	f.logic = NewSendEmailCodeLogic(context.Background(), s)
	f.logic.send = func(context.Context, config.MailConfig, string, string) error { f.calls.Add(1); return nil }
	t.Cleanup(func() {
		defer f.raw.Close()
		patterns := []string{"*" + f.prefix + "*", "*" + strings.ToUpper(f.prefix) + "*"}
		for ip := range f.ips {
			patterns = append(patterns, "*"+ip+"*", "*"+strings.ToUpper(ip)+"*")
		}
		for _, pattern := range patterns {
			iter := f.raw.Scan(context.Background(), 0, pattern, 100).Iterator()
			for iter.Next(context.Background()) {
				if err := f.raw.Del(context.Background(), iter.Val()).Err(); err != nil {
					t.Error(err)
				}
			}
			if err := iter.Err(); err != nil {
				t.Error(err)
			}
		}
	})
	return f
}

func (f *mailLimitFixture) ip(n int) string {
	ip := net.ParseIP(f.stem + fmt.Sprintf("%x", n)).String()
	f.mu.Lock()
	f.ips[ip] = true
	f.mu.Unlock()
	return ip
}

func (f *mailLimitFixture) send(address, ip, scene string) error {
	_, err := f.logic.SendEmailCode(&pb.SendEmailCodeRequest{Email: address, ClientIP: ip, Scene: scene, IsForce: true})
	return err
}

func assertMailErrorCode(t *testing.T, err error, code uint32) {
	t.Helper()
	var coded *xerr.CodeError
	if !errors.As(err, &coded) || coded.GetErrCode() != code {
		t.Fatalf("expected mail error %d, got %v", code, err)
	}
}

func TestMailDailyIPLimitSharedAcrossEmailsAndScenes(t *testing.T) {
	f := newMailLimitFixture(t)
	for i := 0; i < 6; i++ {
		ip := f.ip(1)
		if i%2 == 1 {
			ip = strings.ToUpper(ip)
		}
		err := f.send(fmt.Sprintf("%s-%d@example.test", f.prefix, i), ip, fmt.Sprintf("scene-%d", i))
		if i < 5 && err != nil {
			t.Fatal(err)
		}
		if i == 5 {
			assertMailErrorCode(t, err, 300004)
		}
	}
	if f.calls.Load() != 5 {
		t.Fatalf("sent %d emails, want 5", f.calls.Load())
	}
}

func TestMailDailyEmailLimitSharedAcrossIPScenesAndCase(t *testing.T) {
	f := newMailLimitFixture(t)
	for i := 0; i < 6; i++ {
		address := f.prefix + "@example.test"
		if i%2 == 1 {
			address = strings.ToUpper(address)
		}
		err := f.send(address, f.ip(i+1), fmt.Sprintf("scene-%d", i))
		if i < 5 && err != nil {
			t.Fatal(err)
		}
		if i == 5 {
			assertMailErrorCode(t, err, 300004)
		}
	}
	if f.calls.Load() != 5 {
		t.Fatalf("sent %d emails, want 5", f.calls.Load())
	}
	// A rejection caused by the exhausted email must not consume the fresh IP.
	for i := 0; i < 5; i++ {
		if err := f.send(fmt.Sprintf("%s-fresh-%d@example.test", f.prefix, i), f.ip(6), "register"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMailInvalidOrMissingClientIPRejected(t *testing.T) {
	f := newMailLimitFixture(t)
	for i, ip := range []string{"", "unknown", "not-an-ip", "203.0.113.1, 203.0.113.2", "203.0.113.1:80", "fe80::1%eth0"} {
		assertMailErrorCode(t, f.send(fmt.Sprintf("%s-%d@example.test", f.prefix, i), ip, "register"), 300003)
	}
	if f.calls.Load() != 0 {
		t.Fatal("invalid client IP reached SMTP")
	}
}

func TestMailConcurrentRequestsCannotExceedFive(t *testing.T) {
	for _, shared := range []string{"ip", "email"} {
		t.Run(shared, func(t *testing.T) {
			f := newMailLimitFixture(t)
			start, results := make(chan struct{}), make(chan error, 20)
			for i := 0; i < 20; i++ {
				go func(i int) {
					address, ip := fmt.Sprintf("%s-%d@example.test", f.prefix, i), f.ip(1)
					if shared == "email" {
						address, ip = f.prefix+"@example.test", f.ip(i+1)
					}
					<-start
					results <- f.send(address, ip, fmt.Sprintf("scene-%d", i))
				}(i)
			}
			close(start)
			success := 0
			var rejected []error
			for i := 0; i < 20; i++ {
				if err := <-results; err == nil {
					success++
				} else {
					rejected = append(rejected, err)
				}
			}
			for _, err := range rejected {
				assertMailErrorCode(t, err, 300004)
			}
			if success != 5 || f.calls.Load() != 5 {
				t.Fatalf("success=%d SMTP calls=%d, want exactly 5", success, f.calls.Load())
			}
		})
	}
}

func TestMailCooldownDoesNotSpendDailyBudgetAndCodeIsSingleUse(t *testing.T) {
	f := newMailLimitFixture(t)
	address, ip, scene := f.prefix+"@example.test", f.ip(1), "register"
	if err := f.send(address, ip, scene); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		assertMailErrorCode(t, f.send(address, ip, scene), 300004)
	}
	codeKey := fmt.Sprintf(emailCodePrefix, scene, address)
	code, err := f.raw.Get(context.Background(), codeKey).Result()
	if err != nil || len(code) != 6 {
		t.Fatalf("code missing: %v", err)
	}
	if ttl := f.raw.TTL(context.Background(), codeKey).Val(); ttl <= 295*time.Second || ttl > 300*time.Second {
		t.Fatalf("code TTL=%v", ttl)
	}
	if ttl := f.raw.TTL(context.Background(), fmt.Sprintf(emailCooldownPref, scene, address)).Val(); ttl <= 55*time.Second || ttl > 60*time.Second {
		t.Fatalf("cooldown TTL=%v", ttl)
	}
	verify := NewVerifyEmailCodeLogic(context.Background(), f.logic.svcCtx)
	for i := 0; i < 2; i++ {
		got, err := verify.VerifyEmailCode(&pb.VerifyEmailCodeRequest{Email: address, Scene: scene, Code: code})
		if err != nil || got.Matched != (i == 0) {
			t.Fatalf("single-use verification attempt=%d response=%v err=%v", i, got, err)
		}
	}
	for i := 1; i < 6; i++ {
		err := f.send(fmt.Sprintf("%s-other-%d@example.test", f.prefix, i), ip, "reset_password")
		if i < 5 && err != nil {
			t.Fatal(err)
		}
		if i == 5 {
			assertMailErrorCode(t, err, 300004)
		}
	}
	if f.calls.Load() != 5 {
		t.Fatalf("cooldown attempts consumed the budget: sends=%d", f.calls.Load())
	}
}

func TestMailFailedSMTPStillSpendsDailyBudget(t *testing.T) {
	f := newMailLimitFixture(t)
	f.logic.send = func(context.Context, config.MailConfig, string, string) error {
		f.calls.Add(1)
		return errors.New("fake SMTP failure")
	}
	address, ip := f.prefix+"@example.test", f.ip(1)
	for i := 0; i < 6; i++ {
		err := f.send(address, ip, "register")
		if i < 5 {
			assertMailErrorCode(t, err, 300005)
		} else {
			assertMailErrorCode(t, err, 300004)
		}
	}
	if f.calls.Load() != 5 {
		t.Fatalf("failed SMTP bypassed limit: calls=%d", f.calls.Load())
	}
	if n := f.raw.Exists(context.Background(), fmt.Sprintf(emailCodePrefix, "register", address), fmt.Sprintf(emailCooldownPref, "register", address)).Val(); n != 0 {
		t.Fatal("failed send left a usable code or cooldown")
	}
}

func TestMailDailyBudgetExpiresAtAndResetsOnBeijingMidnight(t *testing.T) {
	f := newMailLimitFixture(t)
	ctx := context.Background()
	// Exercise the real Redis scripts on either side of a future Beijing
	// midnight without changing the server clock or waiting until midnight.
	midnight := time.Now().UTC().Truncate(24 * time.Hour).Add(16 * time.Hour)
	if !midnight.After(time.Now().UTC()) {
		midnight = midnight.Add(24 * time.Hour)
	}
	before := midnight.Add(-time.Second)
	f.logic.now = func() time.Time { return before }
	address, ip := f.prefix+"@example.test", f.ip(1)
	for i := 0; i < 5; i++ {
		if err := f.send(address, ip, fmt.Sprintf("before-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	assertMailErrorCode(t, f.send(address, ip, "before-rejected"), 300004)
	day := before.Add(8 * time.Hour).Format("20060102")
	for _, key := range []string{"biz#emailcode#daily:email:" + address + ":" + day, "biz#emailcode#daily:ip:" + ip + ":" + day} {
		if count, err := f.raw.Get(ctx, key).Int(); err != nil || count != 5 {
			t.Fatalf("daily count=%d err=%v", count, err)
		}
		if expires, err := f.raw.Do(ctx, "EXPIRETIME", key).Int64(); err != nil || expires != midnight.Unix() {
			t.Fatalf("daily expiry=%d want=%d err=%v", expires, midnight.Unix(), err)
		}
	}
	f.logic.now = func() time.Time { return midnight }
	for i := 0; i < 5; i++ {
		if err := f.send(address, ip, fmt.Sprintf("after-%d", i)); err != nil {
			t.Fatal(err)
		}
	}
	assertMailErrorCode(t, f.send(address, ip, "after-rejected"), 300004)
	if f.calls.Load() != 10 {
		t.Fatalf("day rollover allowed %d SMTP calls, want 5 on each day", f.calls.Load())
	}
}

func TestMailExistingCodeAndForceProbeDoNotSpendBudget(t *testing.T) {
	f := newMailLimitFixture(t)
	address, ip, scene := f.prefix+"@example.test", f.ip(1), "register"
	if err := f.send(address, ip, scene); err != nil {
		t.Fatal(err)
	}
	// Only expire the cooldown; the 5-minute code is still valid.
	if err := f.raw.Del(context.Background(), fmt.Sprintf(emailCooldownPref, scene, address)).Err(); err != nil {
		t.Fatal(err)
	}
	_, err := f.logic.SendEmailCode(&pb.SendEmailCodeRequest{Email: address, ClientIP: ip, Scene: scene})
	assertMailErrorCode(t, err, 300004)
	assertMailErrorCode(t, f.send(address, ip, scene), 300004)
	for i := 0; i < 4; i++ {
		if err := f.send(fmt.Sprintf("%s-other-%d@example.test", f.prefix, i), ip, scene); err != nil {
			t.Fatal(err)
		}
	}
	if f.calls.Load() != 5 {
		t.Fatalf("non-sending probes consumed budget: sent=%d", f.calls.Load())
	}
}

func TestMailUnavailableRedisFailsClosed(t *testing.T) {
	logic := NewSendEmailCodeLogic(context.Background(), &svc.ServiceContext{})
	_, err := logic.SendEmailCode(&pb.SendEmailCodeRequest{Email: "valid@example.test", ClientIP: "203.0.113.1"})
	assertMailErrorCode(t, err, 900000)
}

func TestMailExpiredDayWindowCannotSendWithoutCounting(t *testing.T) {
	f := newMailLimitFixture(t)
	f.logic.now = func() time.Time { return time.Now().Add(-48 * time.Hour) }
	assertMailErrorCode(t, f.send(f.prefix+"@example.test", f.ip(1), "register"), 900000)
	if f.calls.Load() != 0 {
		t.Fatal("an expired daily window allowed an uncounted SMTP attempt")
	}
}

func TestEmailDayWindowUsesBeijingCalendar(t *testing.T) {
	for _, test := range []struct{ now, day, expires string }{
		{"2026-10-19T15:59:59.999Z", "20261019", "2026-10-19T16:00:00Z"},
		{"2026-10-19T16:00:00Z", "20261020", "2026-10-20T16:00:00Z"},
		{"2026-10-20T00:00:00Z", "20261020", "2026-10-20T16:00:00Z"},
		{"2026-12-31T16:00:00Z", "20270101", "2027-01-01T16:00:00Z"},
	} {
		now, err := time.Parse(time.RFC3339Nano, test.now)
		if err != nil {
			t.Fatal(err)
		}
		day, expires := emailDayWindow(now)
		if day != test.day || expires.UTC().Format(time.RFC3339) != test.expires {
			t.Fatalf("at %s got day=%s expiry=%s", test.now, day, expires)
		}
	}
}
