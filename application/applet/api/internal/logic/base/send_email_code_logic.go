package base

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"math/big"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"

	"go-zero-admin/application/applet/api/internal/config"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result/xerr"

	"github.com/jordan-wright/email"
	"github.com/zeromicro/go-zero/core/logx"
)

const prefixEmailCode = "biz#emailcode#email:%s"
const expireEmailCode = 300

type SendEmailCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailCodeLogic {
	return &SendEmailCodeLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}
func normalizeEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	addr, err := mail.ParseAddress(value)
	if err != nil || addr.Address != value || strings.ContainsAny(value, "\r\n") {
		return "", fmt.Errorf("请输入有效邮箱地址")
	}
	return addr.Address, nil
}
func mailSettings(c config.MailConfig) config.MailConfig {
	if c.Host == "" {
		c.Host = os.Getenv("SMTP_HOST")
	}
	if c.Username == "" {
		c.Username = os.Getenv("SMTP_USER")
	}
	if c.From == "" {
		c.From = os.Getenv("SMTP_FROM")
	}
	if c.Password == "" {
		c.Password = os.Getenv("SMTP_PASSWORD")
	}
	if c.Password == "" {
		c.Password = os.Getenv("MailPassword")
	}
	if c.Port == 0 {
		c.Port, _ = strconv.Atoi(os.Getenv("SMTP_PORT"))
	}
	if c.Port == 0 {
		c.Port = 465
	}
	return c
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
func emailCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
func (l *SendEmailCodeLogic) SendEmailCode(req *types.SendEmailCodeRequest) (*types.MessageResponse, error) {
	address, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(300003, err.Error())
	}
	cfg := mailSettings(l.svcCtx.Config.Mail)
	if err = validateMail(cfg); err != nil {
		return nil, err
	}
	key := fmt.Sprintf(prefixEmailCode, address)
	cooldown := "biz#emailcode#cooldown:" + address
	// Even force resend keeps a one-minute limit, including concurrent requests.
	allowed, err := l.svcCtx.BizRedis.SetnxExCtx(l.ctx, cooldown, "1", 60)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	if !allowed {
		return nil, xerr.NewErrCodeMsg(300004, "发送过于频繁，请稍后重试")
	}
	old, err := l.svcCtx.BizRedis.GetCtx(l.ctx, key)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	if old != "" && !req.IsForce {
		return nil, xerr.NewErrCodeMsg(300004, "验证码仍有效，请稍后再发送")
	}
	code, err := emailCode()
	if err != nil {
		return nil, err
	}
	if err = l.svcCtx.BizRedis.SetexCtx(l.ctx, key, code, expireEmailCode); err != nil {
		return nil, xerr.NewErrCodeMsg(900000, "验证码保存失败")
	}
	if err = sendCode(l.ctx, cfg, address, code); err != nil {
		// Delete only this attempt's code, not a later successful resend.
		_, _ = l.svcCtx.BizRedis.EvalCtx(context.Background(), "if redis.call('get',KEYS[1]) == ARGV[1] then return redis.call('del',KEYS[1]) else return 0 end", []string{key}, code)
		l.Error("邮件验证码发送失败")
		return nil, xerr.NewErrCodeMsg(300005, "验证码发送失败，请检查邮件服务")
	}
	return &types.MessageResponse{Message: "验证码发送成功"}, nil
}
func sendCode(ctx context.Context, c config.MailConfig, address, code string) error {
	endpoint := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	dialer := tls.Dialer{NetDialer: &net.Dialer{Timeout: 10 * time.Second}, Config: &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}}
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
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
	// A successful DATA reply means the server accepted the email. A later QUIT
	// disconnect must not delete the already delivered verification code.
	_ = client.Quit()
	return nil
}
