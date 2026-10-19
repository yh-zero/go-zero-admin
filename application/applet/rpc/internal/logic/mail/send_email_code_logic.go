package maillogic

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"go-zero-admin/application/applet/rpc/internal/config"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/gofrs/uuid/v5"
	"github.com/zeromicro/go-zero/core/logx"
)

type SendEmailCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	send   func(context.Context, config.MailConfig, string, string) error
	now    func() time.Time
	logx.Logger
}

func NewSendEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailCodeLogic {
	return &SendEmailCodeLogic{ctx: ctx, svcCtx: svcCtx, send: sendMail, now: time.Now, Logger: logx.WithContext(ctx)}
}

// The code, cooldown, and both daily budgets are reserved in one Redis operation.
// Cooldown/existing-code rejections do not consume the daily allowance, and a
// failed SMTP attempt retains its allowance to bound repeated sending failures.
const reserveEmailSendScript = `
if redis.call('EXISTS', KEYS[2]) == 1 then return 1 end
if redis.call('EXISTS', KEYS[1]) == 1 and ARGV[1] == '0' then
  redis.call('SET', KEYS[2], ARGV[7], 'EX', ARGV[4])
  return 2
end
local redisTime = redis.call('TIME')
if tonumber(ARGV[3]) <= tonumber(redisTime[1]) then return 4 end
local emailCount = tonumber(redis.call('GET', KEYS[3]) or '0')
local ipCount = tonumber(redis.call('GET', KEYS[4]) or '0')
if emailCount >= tonumber(ARGV[2]) or ipCount >= tonumber(ARGV[2]) then return 3 end
redis.call('INCR', KEYS[3]); redis.call('EXPIREAT', KEYS[3], ARGV[3])
redis.call('INCR', KEYS[4]); redis.call('EXPIREAT', KEYS[4], ARGV[3])
redis.call('SET', KEYS[2], ARGV[7], 'EX', ARGV[4])
redis.call('SET', KEYS[1], ARGV[5], 'EX', ARGV[6])
redis.call('DEL', KEYS[5])
return 0`

const releaseFailedEmailSendScript = `
if redis.call('GET', KEYS[1]) == ARGV[1] then redis.call('DEL', KEYS[1]) end
if redis.call('GET', KEYS[2]) == ARGV[2] then redis.call('DEL', KEYS[2]) end
return 0`

// SendEmailCode applies the same five-per-day IP and email budgets to every
// caller, including legacy admin and App flows. A valid source IP is mandatory.
func (l *SendEmailCodeLogic) SendEmailCode(in *pb.SendEmailCodeRequest) (*pb.NoDataResponse, error) {
	if in == nil {
		return nil, xerr.NewErrCodeMsg(300003, "请求无效")
	}
	address, err := normalizeEmail(in.Email)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(300003, err.Error())
	}
	ip := net.ParseIP(strings.TrimSpace(in.ClientIP))
	if ip == nil {
		return nil, xerr.NewErrCodeMsg(300003, "无法确认请求来源，请稍后重试")
	}
	if l.svcCtx == nil || l.svcCtx.BizRedis == nil {
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	cfg := l.svcCtx.Config.Mail
	if err = validateMail(cfg); err != nil {
		return nil, err
	}
	scene := mailScene(in.Scene)
	key := fmt.Sprintf(emailCodePrefix, scene, address)
	cooldownKey := fmt.Sprintf(emailCooldownPref, scene, address)
	day, expires := emailDayWindow(l.now())
	dailyKey := fmt.Sprintf(emailDailyPref, strings.ToLower(address), day)
	ipKey := fmt.Sprintf(emailIPDailyPref, ip.String(), day)
	attemptsKey := fmt.Sprintf(emailAttemptPref, scene, address)
	rds := l.svcCtx.BizRedis
	code, err := emailCode()
	if err != nil {
		return nil, err
	}
	reservation, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	force := "0"
	if in.IsForce {
		force = "1"
	}
	value, err := rds.EvalCtx(l.ctx, reserveEmailSendScript, []string{key, cooldownKey, dailyKey, ipKey, attemptsKey},
		force, dailyLimit, expires.Unix(), cooldownSeconds, code, expireEmailCode, reservation.String())
	if err != nil {
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	status, ok := value.(int64)
	if !ok {
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	switch status {
	case 0:
	case 1:
		return nil, xerr.NewErrCodeMsg(300004, "发送过于频繁，请稍后重试")
	case 2:
		return nil, xerr.NewErrCodeMsg(300004, "验证码仍有效，请稍后再发送")
	case 3:
		return nil, xerr.NewErrCodeMsg(300004, "今日验证码请求次数已达上限（每天最多5次）")
	default:
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	if err = l.send(l.ctx, cfg, address, code); err != nil {
		// Roll back only this attempt's code so a concurrent resend is not
		// corrupted, and release the cooldown so the user can retry right
		// away instead of waiting out a dead minute. The daily/IP counters
		// still absorbed this attempt, so abuse is still capped.
		_, _ = rds.EvalCtx(l.ctx, releaseFailedEmailSendScript, []string{key, cooldownKey}, code, reservation.String())
		l.Error("邮件验证码发送失败")
		return nil, xerr.NewErrCodeMsg(300005, "验证码发送失败，请检查邮件服务")
	}
	return &pb.NoDataResponse{}, nil
}
