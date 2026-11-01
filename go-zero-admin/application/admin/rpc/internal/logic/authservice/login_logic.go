package authservicelogic

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/zeromicro/go-zero/core/limit"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/jwtx"
	"go-zero-admin/pkg/xerr"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// Login 用户登录：验证码 → 查用户 → 校验密码 → 签发 JWT
func (l *LoginLogic) Login(in *adminpb.LoginRequest) (*adminpb.LoginResponse, error) {
	// 限流：IP 维度 20 次/分钟 + 账号维度 10 次/分钟（go-zero PeriodLimit，Lua 原子）
	limiterIP := limit.NewPeriodLimit(60, 20, l.svcCtx.Redis, "admin:login:ip:")
	if code, err := limiterIP.TakeCtx(l.ctx, in.Ip); err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	} else if code == limit.OverQuota {
		return nil, xerr.NewErrCodeMsg(400, "登录过于频繁，请稍后再试")
	}
	limiterAcct := limit.NewPeriodLimit(60, 10, l.svcCtx.Redis, "admin:login:acct:")
	if code, err := limiterAcct.TakeCtx(l.ctx, in.Username); err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	} else if code == limit.OverQuota {
		return nil, xerr.NewErrCodeMsg(400, "登录尝试过多，请稍后再试")
	}

	// 1. 按服务端配置校验验证码
	if l.svcCtx.Config.Captcha.Enabled {
		if err := l.verifyCaptcha(in.CaptchaId, in.Captcha); err != nil {
			return nil, err
		}
	}

	// 2. 查用户
	user, err := l.svcCtx.SysUserModel.FindOneByUsername(l.ctx, in.Username)
	if err != nil {
		if err == sqlx.ErrNotFound {
			return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
		}
		l.Errorf("query user error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}

	// 3. 校验状态
	if user.DeletedAt.Valid {
		return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
	}
	if user.Status != 1 {
		return nil, xerr.NewErrCode(xerr.USER_DISABLED)
	}

	// 4. 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, xerr.NewErrCode(xerr.USER_PASSWORD_ERROR)
	}

	// 5. 在用户行锁内核对密码/状态快照并创建会话，避免并发改密后签发旧凭据。
	expire := l.svcCtx.Config.JwtAuth.AccessExpire
	var session *jwtx.Session
	var refresh string
	err = l.svcCtx.SysUserModel.WithLoginUser(l.ctx, user.Id, user.PasswordHash, func(ctx context.Context, current *model.SysUser) error {
		var createErr error
		session, refresh, createErr = l.svcCtx.Sessions.Create(ctx, int64(current.Id), current.Username, current.DeptId.Int64, in.Ip, in.UserAgent, time.Duration(l.svcCtx.Config.JwtAuth.RefreshExpire)*time.Second)
		if createErr != nil {
			return sessionError(createErr)
		}
		return nil
	})
	if err != nil {
		l.revokeCreatedSession(session)
		if errors.Is(err, model.ErrPasswordChanged) {
			return nil, xerr.NewErrCode(xerr.USER_PASSWORD_ERROR)
		}
		if errors.Is(err, model.ErrUserDisabled) {
			return nil, xerr.NewErrCode(xerr.USER_DISABLED)
		}
		if errors.Is(err, sqlx.ErrNotFound) {
			return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
		}
		if business, ok := xerr.AsCodeError(err); ok {
			return nil, business
		}
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	token, err := jwtx.GenerateSessionAccessToken(l.svcCtx.Config.JwtAuth.AccessSecret, session, expire)
	if err != nil {
		l.revokeCreatedSession(session)
		return nil, xerr.NewErrCode(xerr.TOKEN_GENERATE_ERROR)
	}

	// 7. 更新最后登录信息（best effort，失败不阻断登录）
	l.updateLastLogin(int64(user.Id), in.Ip)

	return &adminpb.LoginResponse{
		AccessToken:      token,
		ExpiresAt:        time.Now().Unix() + expire,
		RefreshToken:     refresh,
		RefreshExpiresAt: session.ExpiresAt,
		SessionId:        session.Id,
	}, nil
}

func (l *LoginLogic) revokeCreatedSession(session *jwtx.Session) {
	if session == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(l.ctx), time.Second)
	defer cancel()
	_ = l.svcCtx.Sessions.Revoke(ctx, session.UserId, session.Id)
}

func (l *LoginLogic) verifyCaptcha(captchaID, captcha string) error {
	if captchaID == "" || captcha == "" {
		return xerr.NewErrCode(xerr.CAPTCHA_WRONG)
	}
	key := "captcha:" + captchaID
	stored, err := l.svcCtx.Redis.GetDelCtx(l.ctx, key)
	if err != nil {
		return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if stored == "" {
		return xerr.NewErrCode(xerr.CAPTCHA_EXPIRED)
	}
	if stored != captcha {
		return xerr.NewErrCode(xerr.CAPTCHA_WRONG)
	}
	return nil
}

func sessionError(err error) error {
	if errors.Is(err, jwtx.ErrSessionInvalid) {
		return xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
}

func (l *LoginLogic) updateLastLogin(userID int64, ip string) {
	_ = l.svcCtx.SysUserModel.UpdateLastLogin(l.ctx, uint64(userID), ip)
}
