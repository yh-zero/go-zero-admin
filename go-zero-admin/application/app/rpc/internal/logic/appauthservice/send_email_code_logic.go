package appauthservicelogic

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

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

type SendEmailCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendEmailCodeLogic(ctx context.Context, s *svc.ServiceContext) *SendEmailCodeLogic {
	return &SendEmailCodeLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *SendEmailCodeLogic) SendEmailCode(in *apppb.AppSendEmailCodeRequest) (*apppb.NoDataResponse, error) {
	account, err := emailAccount(in.Account)
	if err != nil {
		return nil, err
	}
	if !smtpEnabled(l.svcCtx) {
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "邮箱验证服务未配置")
	}
	if err = takeLimit(l.ctx, l.svcCtx, "email:ip", in.IP, 3600, 10); err != nil {
		return nil, err
	}
	if err = takeLimit(l.ctx, l.svcCtx, "email:account", account, 60, 1); err != nil {
		return nil, err
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	code := fmt.Sprintf("%06d", n.Int64())
	// 先保存后发送：发送失败立即删除，不能留下未送达却有效的验证码。
	if err = l.svcCtx.EmailCodes.SetEx(l.ctx, codeKey(account), codeDigest(account, code), time.Duration(l.svcCtx.Config.SMTP.ValidSeconds)*time.Second).Err(); err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if err = sendCode(l.ctx, l.svcCtx, account, code); err != nil {
		if cleanupErr := cleanupFailedEmailCode(l.ctx, l.svcCtx.EmailCodes, codeKey(account), codeDigest(account, code)); cleanupErr != nil {
			// Redis 原错误可能包含参数，不记录邮箱、验证码、摘要或底层错误。
			l.Error("failed to clean up undelivered email verification code")
		}
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "验证码发送失败，请稍后重试")
	}
	return &apppb.NoDataResponse{}, nil
}

type emailCodeCache interface {
	Eval(context.Context, string, []string, ...any) *goredis.Cmd
}

func cleanupFailedEmailCode(ctx context.Context, cache emailCodeCache, key, digest string) error {
	// 请求取消不应取消补偿；只删除本次摘要，避免迟到失败误删新验证码。
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	const removeOwnCode = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
    return redis.call("DEL", KEYS[1])
end
return 0`
	return cache.Eval(cleanupCtx, removeOwnCode, []string{key}, digest).Err()
}
func smtpEnabled(s *svc.ServiceContext) bool {
	c := s.Config.SMTP
	address, err := mail.ParseAddress(c.From)
	return c.Host != "" && c.Port > 0 && c.Port <= 65535 && c.ValidSeconds > 0 &&
		err == nil && address.Address == c.From && !strings.ContainsAny(c.Host+c.From+c.FromName, "\r\n")
}
func sendCode(ctx context.Context, s *svc.ServiceContext, to, code string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	c := s.Config.SMTP
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	var err error
	tlsConfig := &tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}
	endpoint := net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	if c.ImplicitTLS {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsConfig}).DialContext(ctx, "tcp", endpoint)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", endpoint)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	return sendSMTP(ctx, s, conn, to, code)
}

// sendSMTP 负责已建立连接上的 SMTP 会话；连接建立独立处理隐式 TLS。
func sendSMTP(ctx context.Context, s *svc.ServiceContext, conn net.Conn, to, code string) error {
	c := s.Config.SMTP
	deadline := time.Now().Add(10 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if !c.ImplicitTLS {
		if err = client.StartTLS(&tls.Config{ServerName: c.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	}
	if c.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", c.Username, c.Password, c.Host)); err != nil {
			return err
		}
	}
	if err = client.Mail(c.From); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	from := (&mail.Address{Name: c.FromName, Address: c.From}).String()
	message := "From: " + from + "\r\nTo: " + to + "\r\nSubject: Your verification code\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" +
		fmt.Sprintf("Your verification code is %s. It expires in %d seconds.\r\n", code, c.ValidSeconds)
	if _, err = writer.Write([]byte(message)); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	// DATA 的最终 250 表示服务器已接受邮件，QUIT 仅结束会话。
	_ = client.Quit()
	return nil
}
