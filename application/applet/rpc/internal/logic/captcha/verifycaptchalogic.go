package captchalogic

import (
	"context"
	"errors"
	"fmt"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type VerifyCaptchaLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewVerifyCaptchaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyCaptchaLogic {
	return &VerifyCaptchaLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// verifyScript atomically reads and deletes the captcha at the Redis layer.
// A successful match returns 1, a missing-or-wrong id returns 0. The id and
// answer are always validated before talking to Redis so a forged request
// cannot consume an unrelated key.
const verifyCaptchaScript = `local v = redis.call('GET', KEYS[1])
if v == false then return 0 end
if v == ARGV[1] then redis.call('DEL', KEYS[1]); return 1 end
return 0`

// VerifyCaptcha returns Matched=true exactly once per captcha. Invalid input
// (bad id, wrong length, missing entry) collapses to Matched=false so callers
// cannot distinguish the reason; the failure reason is logged so brute-force
// attempts remain observable.
func (l *VerifyCaptchaLogic) VerifyCaptcha(in *pb.VerifyCaptchaRequest) (*pb.VerifyCaptchaResponse, error) {
	if in == nil {
		return &pb.VerifyCaptchaResponse{Matched: false}, nil
	}
	if !validCaptchaID(in.Id) {
		l.Infof("captcha verify rejected: malformed id len=%d", len(in.Id))
		return &pb.VerifyCaptchaResponse{Matched: false}, nil
	}
	if !validCaptchaAnswer(in.Answer) {
		l.Infof("captcha verify rejected: wrong answer length id=%s", in.Id)
		return &pb.VerifyCaptchaResponse{Matched: false}, nil
	}
	key := fmt.Sprintf(prefixCaptcha, in.Id)
	result, err := l.svcCtx.BizRedis.EvalCtx(l.ctx, verifyCaptchaScript, []string{key}, in.Answer)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &pb.VerifyCaptchaResponse{Matched: false}, nil
		}
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	n, ok := result.(int64)
	if !ok {
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	if n != 1 {
		l.Infof("captcha verify mismatched or expired id=%s", in.Id)
	}
	return &pb.VerifyCaptchaResponse{Matched: n == 1}, nil
}
