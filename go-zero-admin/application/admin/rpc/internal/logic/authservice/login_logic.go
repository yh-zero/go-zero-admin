package authservicelogic

import (
	"context"
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/zeromicro/go-zero/core/limit"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
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
	if code, err := limiterIP.TakeCtx(l.ctx, in.Ip); err == nil && code == limit.OverQuota {
		return nil, xerr.NewErrCodeMsg(400, "登录过于频繁，请稍后再试")
	}
	limiterAcct := limit.NewPeriodLimit(60, 10, l.svcCtx.Redis, "admin:login:acct:")
	if code, err := limiterAcct.TakeCtx(l.ctx, in.Username); err == nil && code == limit.OverQuota {
		return nil, xerr.NewErrCodeMsg(400, "登录尝试过多，请稍后再试")
	}

	// 1. 校验验证码
	if err := l.verifyCaptcha(in.CaptchaId, in.Captcha); err != nil {
		return nil, err
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
	if user.Status != 1 {
		return nil, xerr.NewErrCode(xerr.USER_DISABLED)
	}

	// 4. 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, xerr.NewErrCode(xerr.USER_PASSWORD_ERROR)
	}

	// 5. 提取部门 ID（处理 NULL）
	var deptID int64
	if user.DeptId.Valid {
		deptID = user.DeptId.Int64
	}

	// 6. 签发 JWT
	expire := l.svcCtx.Config.JwtAuth.AccessExpire
	token, err := jwtx.GenerateAccessToken(
		l.svcCtx.Config.JwtAuth.AccessSecret,
		int64(user.Id),
		user.Username,
		deptID,
		expire,
	)
	if err != nil {
		l.Errorf("generate token error: %v", err)
		return nil, xerr.NewErrCode(xerr.TOKEN_GENERATE_ERROR)
	}

	// 7. 更新最后登录信息（best effort，失败不阻断登录）
	l.updateLastLogin(int64(user.Id), in.Ip)

	return &adminpb.LoginResponse{
		AccessToken:      token,
		ExpiresAt:        time.Now().Unix() + expire,
		RefreshToken:     "",
		RefreshExpiresAt: 0,
	}, nil
}

func (l *LoginLogic) verifyCaptcha(captchaID, captcha string) error {
	if captchaID == "" || captcha == "" {
		return xerr.NewErrCode(xerr.CAPTCHA_WRONG)
	}
	rdb := redis.MustNewRedis(redis.RedisConf{
		Host: l.svcCtx.Config.CacheRedis[0].Host,
		Type: l.svcCtx.Config.CacheRedis[0].Type,
	})
	key := "captcha:" + captchaID
	stored, err := rdb.Get(key)
	if err != nil || stored == "" {
		return xerr.NewErrCode(xerr.CAPTCHA_EXPIRED)
	}
	_, _ = rdb.Del(key)
	if stored != captcha {
		return xerr.NewErrCode(xerr.CAPTCHA_WRONG)
	}
	return nil
}

func (l *LoginLogic) updateLastLogin(userID int64, ip string) {
	user, err := l.svcCtx.SysUserModel.FindOne(l.ctx, uint64(userID))
	if err != nil {
		return
	}
	user.LastLoginAt = sql.NullTime{Time: time.Now(), Valid: true}
	user.LastLoginIp = ip
	_ = l.svcCtx.SysUserModel.Update(l.ctx, user)
}
