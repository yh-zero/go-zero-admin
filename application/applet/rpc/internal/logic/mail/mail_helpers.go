package maillogic

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"math/big"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"go-zero-admin/application/applet/rpc/internal/config"
	"go-zero-admin/pkg/result/xerr"

	"github.com/jordan-wright/email"
)

// Key layout for verification codes and rate limits. All keys carry the scene so
// the same email can hold independent codes for "register" / "reset_password".
const (
	emailCodePrefix   = "biz#emailcode#%s#email:%s" // scene, address as supplied
	emailCooldownPref = "biz#emailcode#%s#cooldown:%s"
	emailDailyPref    = "biz#emailcode#%s#daily:%s" // scene, address; YYYY-MM-DD suffix added
	emailAttemptPref  = "biz#emailcode#%s#attempts:%s"
	// The IP key deliberately omits the scene: the hourly budget is shared
	// across all scenes so an attacker cannot multiply it per scene.
	emailIPHourPref = "biz#emailcode#ip:%s" // ip; YYYYMMDDHH suffix added

	expireEmailCode   = 300 // seconds the code itself is valid
	cooldownSeconds   = 60  // 1 minute between consecutive sends to the same address
	dailyLimit        = 20  // per email per scene per UTC day
	ipHourlyLimit     = 30  // per source IP across ALL scenes per UTC hour
	maxVerifyAttempts = 5   // wrong-code guesses before the code is invalidated
)

// normalizeEmail trims and rejects control characters that could enable SMTP
// injection. We deliberately do NOT lowercase the address; the local-part is
// case-sensitive per RFC 5321 and the Redis key is the exact address the user
// supplied (after trim).
func normalizeEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value || strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("请输入有效邮箱地址")
	}
	return addr.Address, nil
}

// emailCode returns a 6-digit zero-padded verification code.
func emailCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func mailScene(scene string) string {
	if scene == "" {
		return "default"
	}
	return scene
}

func validateMail(c config.MailConfig) error {
	if c.Host == "" || c.Username == "" || c.Password == "" || c.From == "" || c.Port < 1 || c.Port > 65535 {
		return xerr.NewErrCodeMsg(300005, "邮件服务未配置")
	}
	if _, err := mail.ParseAddress(c.From); err != nil {
		return xerr.NewErrCodeMsg(300005, "邮件发件地址配置无效")
	}
	return nil
}

// sendMail dials SMTP using implicit TLS and honors request cancellation via
// DialContext, identical to the previous API-side implementation. It returns
// once the server has accepted the message body; a late QUIT failure must not
// roll back the success.
func sendMail(ctx context.Context, c config.MailConfig, address, code string) error {
	dialer := tls.Dialer{NetDialer: &net.Dialer{Timeout: 10 * time.Second}, Config: &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}}
	conn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(c.Host, strconv.Itoa(c.Port)))
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if err = client.Auth(smtp.PlainAuth("", c.Username, c.Password, c.Host)); err != nil {
		return err
	}
	from, err := mail.ParseAddress(c.From)
	if err != nil {
		return err
	}
	if err = client.Mail(from.Address); err != nil {
		return err
	}
	if err = client.Rcpt(address); err != nil {
		return err
	}
	e := email.NewEmail()
	e.From = c.From
	e.To = []string{address}
	e.Subject = "邮箱验证码"
	e.Text = []byte("验证码：" + code + "，5分钟内有效。")
	data, err := e.Bytes()
	if err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	if _, err = writer.Write(data); err != nil {
		_ = writer.Close()
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	_ = client.Quit()
	return nil
}
