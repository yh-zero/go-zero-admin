package appaccountservicelogic

import (
	"context"
	"errors"
	"net/mail"
	"regexp"
	"strings"
	"time"

	maillogic "go-zero-admin/application/applet/rpc/internal/logic/mail"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/appauth"
	"go-zero-admin/pkg/result/xerr"
)

const appEmailScene = "app_register"

var emailCodePattern = regexp.MustCompile(`^[0-9]{6}$`)

func verifyRegistrationEmail(ctx context.Context, s *svc.ServiceContext, kind, account, code string) (*time.Time, error) {
	code = strings.TrimSpace(code)
	if kind != "email" {
		if code != "" {
			return nil, badInput("手机号注册不支持邮箱验证码")
		}
		return nil, nil
	}
	if code == "" {
		if s.Config.AppJwtAuth.EmailVerificationRequired {
			return nil, badInput("请输入邮箱验证码")
		}
		return nil, nil
	}
	if !emailCodePattern.MatchString(code) {
		return nil, badInput("邮箱验证码须为6位数字")
	}
	if s.BizRedis == nil {
		return nil, badInput("验证码服务暂不可用，请稍后重试")
	}
	result, err := maillogic.NewVerifyEmailCodeLogic(ctx, s).VerifyEmailCode(&pb.VerifyEmailCodeRequest{Email: account, Code: code, Scene: appEmailScene})
	if err != nil {
		return nil, mailError(err)
	}
	if result == nil || !result.Matched {
		return nil, badInput("邮箱验证码错误或已过期")
	}
	now := time.Now().UTC()
	return &now, nil
}
func mailEnabled(s *svc.ServiceContext) bool {
	if s == nil {
		return false
	}
	c := s.Config.Mail
	if c.Host == "" || c.Username == "" || c.Password == "" || c.From == "" || c.Port < 1 || c.Port > 65535 {
		return false
	}
	_, err := mail.ParseAddress(c.From)
	return err == nil
}
func authOptions(s *svc.ServiceContext) (*pb.AppAuthOptions, error) {
	return &pb.AppAuthOptions{EmailVerificationRequired: s.Config.AppJwtAuth.EmailVerificationRequired, EmailCodeEnabled: mailEnabled(s)}, nil
}
func sendEmailCode(ctx context.Context, s *svc.ServiceContext, in *pb.AppSendEmailCodeRequest) (*pb.NoDataResponse, error) {
	if in == nil {
		return nil, badInput("请求参数无效")
	}
	kind, address, err := appauth.NormalizeAccount(in.Account)
	if err != nil || kind != "email" {
		return nil, badInput("请输入有效邮箱地址")
	}
	if !mailEnabled(s) {
		return nil, badInput("邮件服务尚未配置，请联系管理员")
	}
	if s.BizRedis == nil {
		return nil, badInput("验证码服务暂不可用，请稍后重试")
	}
	ip := strings.TrimSpace(in.IP)
	result, err := maillogic.NewSendEmailCodeLogic(ctx, s).SendEmailCode(&pb.SendEmailCodeRequest{Email: address, Scene: appEmailScene, ClientIP: ip, IsForce: true})
	if err != nil {
		return nil, mailError(err)
	}
	return result, nil
}

// Convert the legacy mail-specific codes to the existing public business error
// envelope, preserving only messages produced by the trusted mail logic.
func mailError(err error) error {
	var coded *xerr.CodeError
	if errors.As(err, &coded) {
		switch coded.GetErrCode() {
		case 300003, 300004, 300005, 900000:
			return badInput(coded.GetErrMsg())
		}
	}
	return badInput("验证码服务暂不可用，请稍后重试")
}
