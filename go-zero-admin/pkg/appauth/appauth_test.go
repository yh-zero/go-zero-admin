package appauth

import (
	"github.com/golang-jwt/jwt/v4"
	"testing"
	"time"
)

func TestNormalizeAccount(t *testing.T) {
	for _, tc := range []struct{ in, kind, want string }{{" User@EXAMPLE.com ", "email", "user@example.com"}, {"13800138000", "phone", "+8613800138000"}, {"+14155552671", "phone", "+14155552671"}} {
		kind, got, err := NormalizeAccount(tc.in)
		if err != nil || kind != tc.kind || got != tc.want {
			t.Fatalf("NormalizeAccount(%q)=(%q,%q,%v)", tc.in, kind, got, err)
		}
	}
	for _, in := range []string{"name", "Person <x@example.com>", "+00123456789", "12345678901", "a@localhost"} {
		if _, _, err := NormalizeAccount(in); err == nil {
			t.Fatalf("accepted invalid account %q", in)
		}
	}
}
func TestTokensRejectWrongDomainAlgorithmAndKind(t *testing.T) {
	cfg := Config{AccessSecret: "test-app-secret-01234567890123456789", AccessExpire: 900, RefreshExpire: 2592000}
	now := time.Now().UTC()
	access, err := Issue(cfg, 12, "session-id", 3, 1, "access", now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(cfg, access, "access"); err != nil {
		t.Fatal(err)
	}
	if _, err = Parse(cfg, access, "refresh"); err == nil {
		t.Fatal("accepted access token for refresh")
	}
	for _, claims := range []jwt.MapClaims{{"iss": "admin", "aud": Audience, "exp": now.Add(time.Hour).Unix()}, {"iss": Issuer, "aud": "admin", "exp": now.Add(time.Hour).Unix()}} {
		signed, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.AccessSecret))
		if _, err = Parse(cfg, signed, "access"); err == nil {
			t.Fatal("accepted another authentication domain")
		}
	}
	signed, _ := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{"iss": Issuer, "aud": Audience, "exp": now.Add(time.Hour).Unix()}).SignedString([]byte(cfg.AccessSecret))
	if _, err = Parse(cfg, signed, "access"); err == nil {
		t.Fatal("accepted wrong algorithm")
	}
	expired, _ := Issue(cfg, 12, "session-id", 3, 1, "access", now.Add(-time.Minute))
	if _, err = Parse(cfg, expired, "access"); err == nil {
		t.Fatal("accepted expired token")
	}
}
