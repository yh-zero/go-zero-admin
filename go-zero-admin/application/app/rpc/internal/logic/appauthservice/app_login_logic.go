package appauthservicelogic

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/limit"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/pkg/apptoken"
	"go-zero-admin/pkg/jwtx"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type AppLoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AppLoginLogic {
	return &AppLoginLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *AppLoginLogic) AppLogin(in *apppb.AppLoginRequest) (*apppb.AppLoginResponse, error) {
	account := strings.ToLower(strings.TrimSpace(in.Account))

	// 限流：IP 维度 20 次/分钟 + 账号维度 10 次/分钟（go-zero PeriodLimit，Lua 原子）
	limiterIP := limit.NewPeriodLimit(60, 20, l.svcCtx.Redis, "app:login:ip:")
	if code, err := limiterIP.TakeCtx(l.ctx, in.IP); err == nil && code == limit.OverQuota {
		return nil, xerr.NewErrCodeMsg(400, "登录过于频繁，请稍后再试")
	}
	limiterAcct := limit.NewPeriodLimit(60, 10, l.svcCtx.Redis, "app:login:acct:")
	if code, err := limiterAcct.TakeCtx(l.ctx, account); err == nil && code == limit.OverQuota {
		return nil, xerr.NewErrCodeMsg(400, "登录尝试过多，请稍后再试")
	}

	user, err := l.svcCtx.AppUserModel.FindOneByAccount(l.ctx, account)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(400, "账号或密码错误")
	}
	if user.Status != "active" {
		return nil, xerr.NewErrCodeMsg(400, "账号未激活或已停用")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, xerr.NewErrCodeMsg(400, "账号或密码错误")
	}

	// Phase 3: 计算会员状态并刷新缓存
	vip, planCode := apptoken.RefreshVipCache(l.ctx, l.svcCtx.Redis, l.svcCtx.DB, int64(user.Id))

	expire := l.svcCtx.Config.JwtAuth.AccessExpire
	if expire <= 0 {
		expire = 900
	}
	token, err := jwtx.GenerateAppToken(
		l.svcCtx.Config.JwtAuth.AccessSecret,
		int64(user.Id), user.Status, vip, planCode, expire,
	)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.TOKEN_GENERATE_ERROR)
	}

	// 登录成功，清除账号维度限流计数
	l.svcCtx.Redis.Del("limit:app:login:acct:" + account)

	// 更新最后登录
	u, _ := l.svcCtx.AppUserModel.FindOne(l.ctx, user.Id)
	if u != nil {
		u.LastLoginAt = sql.NullTime{Time: time.Now(), Valid: true}
		_ = l.svcCtx.AppUserModel.Update(l.ctx, u)
	}

	return &apppb.AppLoginResponse{
		AccessToken: token, ExpiresAt: time.Now().Unix() + expire,
		User: mapAppUser(user),
	}, nil
}

func mapAppUser(u *model.AppUser) *apppb.AppUserInfo {
	info := &apppb.AppUserInfo{
		Id: int64(u.Id), Account: u.Account, IdentityKind: u.IdentityKind, Nickname: u.Nickname,
		Status: u.Status, ReviewReason: u.ReviewReason,
		EmailVerified: u.EmailVerifiedAt.Valid, PhoneVerified: u.PhoneVerifiedAt.Valid,
	}
	if u.AvatarFileId.Valid {
		info.AvatarFileId = u.AvatarFileId.Int64
	}
	if u.LastLoginAt.Valid {
		info.LastLoginAt = u.LastLoginAt.Time.Unix()
	}
	return info
}
