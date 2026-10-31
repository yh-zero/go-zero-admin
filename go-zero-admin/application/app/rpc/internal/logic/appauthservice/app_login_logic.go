package appauthservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/jwtx"
	"go-zero-admin/pkg/xerr"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

type AppLoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppLoginLogic(ctx context.Context, s *svc.ServiceContext) *AppLoginLogic {
	return &AppLoginLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *AppLoginLogic) AppLogin(in *apppb.AppLoginRequest) (*apppb.AppLoginResponse, error) {
	account := strings.ToLower(strings.TrimSpace(in.Account))
	if len(account) > 128 || len(in.Password) > 64 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if err := takeLimit(l.ctx, l.svcCtx, "login:ip", in.IP, 60, 20); err != nil {
		return nil, err
	}
	if err := takeLimit(l.ctx, l.svcCtx, "login:account", account, 60, 10); err != nil {
		return nil, err
	}
	user, err := l.svcCtx.AppUserModel.FindOneByAccount(l.ctx, account)
	if errors.Is(err, model.ErrNotFound) {
		return nil, xerr.NewErrCodeMsg(400, "账号或密码错误")
	}
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if user.DeletedAt.Valid || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)) != nil {
		return nil, xerr.NewErrCodeMsg(400, "账号或密码错误")
	}
	if user.Status != "active" {
		return nil, xerr.NewErrCode(xerr.USER_DISABLED)
	}
	lifetime := l.svcCtx.Config.JwtAuth.RefreshExpire
	if lifetime <= 0 {
		lifetime = 2592000
	}
	var session *jwtx.Session
	var refresh string
	issued := false
	defer func() {
		if session != nil && !issued {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(l.ctx), 3*time.Second)
			defer cancel()
			if err := l.svcCtx.Sessions.Revoke(cleanupCtx, int64(user.Id), session.Id); err != nil {
				l.Error("App 登录失败后的会话清理失败")
			}
		}
	}()
	err = l.svcCtx.AppUserModel.WithLoginUser(l.ctx, user.Id, user.PasswordHash, func(ctx context.Context) error {
		var createErr error
		session, refresh, createErr = l.svcCtx.Sessions.Create(ctx, int64(user.Id), "", 0, in.IP, in.UserAgent, time.Duration(lifetime)*time.Second)
		if createErr != nil {
			return sessionError(createErr)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, model.ErrNotFound) || errors.Is(err, model.ErrAppLoginChanged) {
			return nil, xerr.NewErrCodeMsg(400, "账号或密码错误")
		}
		if errors.Is(err, model.ErrAppUserInactive) {
			return nil, xerr.NewErrCode(xerr.USER_DISABLED)
		}
		if business, ok := xerr.AsCodeError(err); ok {
			return nil, business
		}
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	now := time.Now().Unix()
	expire := min(accessExpire(l.svcCtx), session.ExpiresAt-now)
	if expire <= 0 {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	token, err := jwtx.GenerateAppSessionToken(l.svcCtx.Config.JwtAuth.AccessSecret, session, "active", false, "", expire)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.TOKEN_GENERATE_ERROR)
	}
	info := mapAppUser(user)
	info.LastLoginAt = now
	issued = true
	return &apppb.AppLoginResponse{AccessToken: token, ExpiresAt: now + expire, RefreshToken: refresh, RefreshExpiresAt: session.ExpiresAt, SessionId: session.Id, User: info}, nil
}
func accessExpire(s *svc.ServiceContext) int64 {
	if s.Config.JwtAuth.AccessExpire > 0 {
		return s.Config.JwtAuth.AccessExpire
	}
	return 900
}
func mapAppUser(u *model.AppUser) *apppb.AppUserInfo {
	info := &apppb.AppUserInfo{Id: int64(u.Id), Account: u.Account, IdentityKind: u.IdentityKind, Nickname: u.Nickname, Status: u.Status, ReviewReason: u.ReviewReason, EmailVerified: u.EmailVerifiedAt.Valid, PhoneVerified: u.PhoneVerifiedAt.Valid, CreatedAt: u.CreatedAt.Unix()}
	if u.AvatarFileId.Valid {
		info.AvatarFileId = u.AvatarFileId.Int64
	}
	if u.LastLoginAt.Valid {
		info.LastLoginAt = u.LastLoginAt.Time.Unix()
	}
	return info
}
