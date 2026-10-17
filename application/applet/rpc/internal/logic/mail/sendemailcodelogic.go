package maillogic

import (
	"context"
	"fmt"
	"time"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendEmailCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailCodeLogic {
	return &SendEmailCodeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// SendEmailCode enforces per-IP, per-email-per-day, and per-email cooldown
// rate limits, generates a 6-digit code, stores it in Redis, and synchronously
// delivers it over SMTP. On SMTP failure the code is removed via a Lua
// compare-and-delete so a later successful resend is not corrupted.
func (l *SendEmailCodeLogic) SendEmailCode(in *pb.SendEmailCodeRequest) (*pb.NoDataResponse, error) {
	if in == nil {
		return nil, xerr.NewErrCodeMsg(300003, "请求无效")
	}
	address, err := normalizeEmail(in.Email)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(300003, err.Error())
	}
	cfg := l.svcCtx.Config.Mail
	if err = validateMail(cfg); err != nil {
		return nil, err
	}
	scene := mailScene(in.Scene)
	key := fmt.Sprintf(emailCodePrefix, scene, address)
	cooldownKey := fmt.Sprintf(emailCooldownPref, scene, address)
	dailyKey := fmt.Sprintf(emailDailyPref, scene, address) + ":" + time.Now().UTC().Format("20060102")
	ipKey := fmt.Sprintf(emailIPHourPref, in.ClientIP) + ":" + time.Now().UTC().Format("2006010215")
	if in.ClientIP == "" {
		ipKey = ""
	}

	rds := l.svcCtx.BizRedis

	// Per-email cooldown first: it doubles as the concurrency lock, and
	// blocked attempts must NOT consume the daily/IP budget.
	allowed, err := rds.SetnxExCtx(l.ctx, cooldownKey, "1", cooldownSeconds)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	if !allowed {
		return nil, xerr.NewErrCodeMsg(300004, "发送过于频繁，请稍后重试")
	}

	old, err := rds.GetCtx(l.ctx, key)
	if err != nil {
		_, _ = rds.DelCtx(l.ctx, cooldownKey)
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	if old != "" && !in.IsForce {
		// Keep the cooldown held: releasing it here would let a client probe
		// with IsForce=false to clear the lock and then immediately force a
		// resend, bypassing the 60s rate limit.
		return nil, xerr.NewErrCodeMsg(300004, "验证码仍有效，请稍后再发送")
	}

	// Per-IP hourly limit (only when ClientIP supplied; protects against
	// address-spraying that would otherwise slip past per-email limits).
	if ipKey != "" {
		n, err := rds.IncrCtx(l.ctx, ipKey)
		if err != nil {
			_, _ = rds.DelCtx(l.ctx, cooldownKey)
			return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
		}
		if n == 1 {
			_ = rds.ExpireCtx(l.ctx, ipKey, 3600)
		}
		if n > int64(ipHourlyLimit) {
			_, _ = rds.DelCtx(l.ctx, cooldownKey)
			return nil, xerr.NewErrCodeMsg(300004, "请求过于频繁，请稍后再试")
		}
	}

	// Per-email-per-day counter.
	n, err := rds.IncrCtx(l.ctx, dailyKey)
	if err != nil {
		_, _ = rds.DelCtx(l.ctx, cooldownKey)
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	if n == 1 {
		_ = rds.ExpireCtx(l.ctx, dailyKey, 86400)
	}
	if n > int64(dailyLimit) {
		_, _ = rds.DelCtx(l.ctx, cooldownKey)
		return nil, xerr.NewErrCodeMsg(300004, "今日验证码请求次数已达上限")
	}

	code, err := emailCode()
	if err != nil {
		return nil, err
	}
	if err = rds.SetexCtx(l.ctx, key, code, expireEmailCode); err != nil {
		return nil, xerr.NewErrCodeMsg(900000, "验证码保存失败")
	}
	if err = sendMail(l.ctx, cfg, address, code); err != nil {
		// Roll back only this attempt's code so a concurrent resend is not
		// corrupted, and release the cooldown so the user can retry right
		// away instead of waiting out a dead minute. The daily/IP counters
		// still absorbed this attempt, so abuse is still capped.
		_, _ = rds.EvalCtx(l.ctx, "if redis.call('get',KEYS[1]) == ARGV[1] then return redis.call('del',KEYS[1]) else return 0 end", []string{key}, code)
		_, _ = rds.DelCtx(l.ctx, cooldownKey)
		l.Error("邮件验证码发送失败")
		return nil, xerr.NewErrCodeMsg(300005, "验证码发送失败，请检查邮件服务")
	}
	return &pb.NoDataResponse{}, nil
}
