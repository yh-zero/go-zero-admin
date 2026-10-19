package maillogic

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

type VerifyEmailCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewVerifyEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *VerifyEmailCodeLogic {
	return &VerifyEmailCodeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// verifyScript atomically: match -> consume code+attempts, return 1.
// Mismatch -> count the attempt; after maxVerifyAttempts the code is
// invalidated so a 6-digit code cannot be brute-forced. Missing code returns
// 0 without counting (expired and wrong must be indistinguishable to callers).
const verifyScript = `
local v = redis.call('GET', KEYS[1])
if v == false then return 0 end
if v == ARGV[1] then
  redis.call('DEL', KEYS[1]); redis.call('DEL', KEYS[2]); return 1
end
local n = redis.call('INCR', KEYS[2])
if n == 1 then redis.call('EXPIRE', KEYS[2], ARGV[2]) end
if n >= tonumber(ARGV[3]) then
  redis.call('DEL', KEYS[1]); redis.call('DEL', KEYS[2])
end
return 0`

// VerifyEmailCode atomically reads and deletes the code at the Redis layer, so
// two concurrent registration attempts for the same email+scene cannot both
// pass. A successful match consumes the code, preventing replay. Wrong guesses
// are counted; after maxVerifyAttempts the code is destroyed.
func (l *VerifyEmailCodeLogic) VerifyEmailCode(in *pb.VerifyEmailCodeRequest) (*pb.VerifyEmailCodeResponse, error) {
	if in == nil {
		return &pb.VerifyEmailCodeResponse{Matched: false}, nil
	}
	address, err := normalizeEmail(in.Email)
	if err != nil {
		return &pb.VerifyEmailCodeResponse{Matched: false}, nil
	}
	if in.Code == "" {
		return &pb.VerifyEmailCodeResponse{Matched: false}, nil
	}
	scene := mailScene(in.Scene)
	key := fmt.Sprintf(emailCodePrefix, scene, address)
	attemptsKey := fmt.Sprintf(emailAttemptPref, scene, address)
	result, err := l.svcCtx.BizRedis.EvalCtx(l.ctx, verifyScript,
		[]string{key, attemptsKey}, in.Code, expireEmailCode, maxVerifyAttempts)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return &pb.VerifyEmailCodeResponse{Matched: false}, nil
		}
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	n, ok := result.(int64)
	if !ok {
		return nil, xerr.NewErrCodeMsg(900000, "验证码服务暂不可用")
	}
	return &pb.VerifyEmailCodeResponse{Matched: n == 1}, nil
}
