package appauthservicelogic

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
	"math/big"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"
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
	if err = l.svcCtx.Redis.SetexCtx(l.ctx, codeKey(account), codeDigest(account, code), 300); err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if err = sendCode(l.ctx, l.svcCtx, account, code); err != nil {
		_, _ = l.svcCtx.Redis.DelCtx(l.ctx, codeKey(account))
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "验证码发送失败，请稍后重试")
	}
	return &apppb.NoDataResponse{}, nil
}
func smtpEnabled(s *svc.ServiceContext) bool {
	c := s.Config.SMTP
	address, err := mail.ParseAddress(c.From)
	return c.Host != "" && c.Port > 0 && c.Port <= 65535 && err == nil && address.Address == c.From && !strings.ContainsAny(c.Host+c.From, "\r\n")
}
func sendCode(ctx context.Context, s *svc.ServiceContext, to, code string) error {
	c := s.Config.SMTP
	deadline := time.Now().Add(10 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
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
	if err = conn.SetDeadline(deadline); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, c.Host)
	if err != nil {
		return err
	}
	defer client.Close()
	if !c.ImplicitTLS {
		if err = client.StartTLS(tlsConfig); err != nil {
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
	message := "From: " + c.From + "\r\nTo: " + to + "\r\nSubject: Your verification code\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nYour verification code is " + code + ". It expires in 5 minutes.\r\n"
	if _, err = writer.Write([]byte(message)); err != nil {
		return err
	}
	if err = writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
