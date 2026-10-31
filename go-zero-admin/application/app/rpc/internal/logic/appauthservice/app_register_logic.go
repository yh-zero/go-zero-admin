package appauthservicelogic

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"github.com/zeromicro/go-zero/core/limit"
	"github.com/zeromicro/go-zero/core/logx"
	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
	"golang.org/x/crypto/bcrypt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

type AppRegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppRegisterLogic(ctx context.Context, s *svc.ServiceContext) *AppRegisterLogic {
	return &AppRegisterLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *AppRegisterLogic) AppRegister(in *apppb.AppRegisterRequest) (*apppb.NoDataResponse, error) {
	account, err := emailAccount(in.Account)
	if err != nil {
		return nil, err
	}
	if err = validatePasswordStrength(in.Password); err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(in.Nickname) > 64 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if err = takeLimit(l.ctx, l.svcCtx, "register:ip", in.Ip, 3600, 10); err != nil {
		return nil, err
	}
	if err = takeLimit(l.ctx, l.svcCtx, "register:account", account, 3600, 5); err != nil {
		return nil, err
	}
	_, err = l.svcCtx.AppUserModel.FindOneByAccount(l.ctx, account)
	if err == nil {
		return nil, xerr.NewErrCodeMsg(400, "账号已存在")
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	verified := false
	if l.svcCtx.Config.EmailVerificationRequired {
		if !smtpEnabled(l.svcCtx) {
			return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "邮箱验证服务未配置")
		}
		expected, readErr := l.svcCtx.Redis.GetDelCtx(l.ctx, codeKey(account))
		if readErr != nil {
			return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
		actual := codeDigest(account, in.EmailCode)
		if expected == "" || subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) != 1 {
			return nil, xerr.NewErrCodeMsg(400, "验证码错误或已过期")
		}
		verified = true
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	now := time.Now()
	user := &model.AppUser{Account: account, IdentityKind: "email", PasswordHash: string(hash), Nickname: in.Nickname, Status: "pending", CreatedAt: now, UpdatedAt: now, EmailVerifiedAt: sql.NullTime{Time: now, Valid: verified}}
	if _, err = l.svcCtx.AppUserModel.Insert(l.ctx, user); err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &apppb.NoDataResponse{}, nil
}
func emailAccount(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(value)
	if err != nil || address.Address != value || len(value) > 128 || strings.ContainsAny(value, "\r\n") {
		return "", xerr.NewErrCodeMsg(400, "请输入有效邮箱")
	}
	return value, nil
}
func codeKey(account string) string { return "app:email-code:" + codeDigest(account, "") }
func codeDigest(account, code string) string {
	sum := sha256.Sum256([]byte(account + ":" + code))
	return hex.EncodeToString(sum[:])
}
func takeLimit(ctx context.Context, s *svc.ServiceContext, scope, key string, period, quota int) error {
	limiter := limit.NewPeriodLimit(period, quota, s.Redis, "app:"+scope+":")
	code, err := limiter.TakeCtx(ctx, codeDigest(key, ""))
	if err != nil {
		return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if code == limit.OverQuota {
		return xerr.NewErrCodeMsg(400, "操作过于频繁，请稍后再试")
	}
	return nil
}
func validatePasswordStrength(password string) error {
	if len(password) < 8 || len(password) > 64 {
		return xerr.NewErrCodeMsg(400, "密码长度需为 8-64 位")
	}
	var letter, digit bool
	for _, c := range password {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			letter = true
		}
		if c >= '0' && c <= '9' {
			digit = true
		}
	}
	if !letter || !digit {
		return xerr.NewErrCodeMsg(400, "密码需同时包含字母和数字")
	}
	return nil
}
