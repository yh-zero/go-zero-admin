package maillogic

import (
	"strings"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/config"
)

func TestNormalizeEmail(t *testing.T) {
	// Trailing whitespace (including a trailing newline) is stripped by
	// TrimSpace before use, so it is benign. CRLF in the middle is header
	// injection and must be rejected.
	for _, value := range []string{"", "invalid", "Display <test@example.test>", "test@example.test\r\nBcc: victim@example.test", "test@ex\nample.test"} {
		if _, err := normalizeEmail(value); err == nil {
			t.Fatalf("accepted invalid recipient: %q", value)
		}
	}
	if got, err := normalizeEmail(" test@example.test "); err != nil || got != "test@example.test" {
		t.Fatalf("valid recipient: %q %v", got, err)
	}
}

func TestValidateMail(t *testing.T) {
	if err := validateMail(config.MailConfig{}); err == nil {
		t.Fatal("missing SMTP config accepted")
	}
	cfg := config.MailConfig{Host: "smtp.example.test", Port: 465, Username: "u", Password: "p", From: "sender@example.test"}
	if err := validateMail(cfg); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	cfg.Port = 70000
	if err := validateMail(cfg); err == nil {
		t.Fatal("out-of-range port accepted")
	}
}

func TestEmailCodeFormat(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		code, err := emailCode()
		if err != nil || len(code) != 6 || strings.Trim(code, "0123456789") != "" {
			t.Fatalf("invalid code: %q %v", code, err)
		}
		seen[code] = true
	}
	if len(seen) < 90 {
		t.Fatalf("codes suspiciously repetitive: %d unique of 100", len(seen))
	}
}

func TestSceneFallsBackToDefault(t *testing.T) {
	if mailScene("") != "default" || mailScene("register") != "register" {
		t.Fatal("scene normalization wrong")
	}
}
