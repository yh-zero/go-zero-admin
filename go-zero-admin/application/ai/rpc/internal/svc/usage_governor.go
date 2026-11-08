package svc

import (
	"context"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/xerr"
)

type usageFacts interface {
	Reserve(context.Context, llm.UsageAttempt) error
	MarkDispatched(context.Context, string) error
	Finish(context.Context, string, llm.UsageObservation) error
	Expire(context.Context) error
	GlobalConcurrentLimit(context.Context) (int64, error)
	Active(context.Context) (int64, error)
}

type UsageGovernor struct {
	facts usageFacts
	redis *redis.Redis
	key   string
}

var _ llm.UsageHook = (*UsageGovernor)(nil)

func NewUsageGovernor(facts usageFacts, rdb *redis.Redis, key ...string) *UsageGovernor {
	leaseKey := "ai:usage:active"
	if len(key) > 0 && key[0] != "" {
		leaseKey = key[0]
	}
	return &UsageGovernor{facts: facts, redis: rdb, key: leaseKey}
}

const usageAcquireLease = `
local now=redis.call('TIME'); local ms=tonumber(now[1])*1000+math.floor(tonumber(now[2])/1000)
redis.call('ZREMRANGEBYSCORE',KEYS[1],'-inf',ms)
if redis.call('ZSCORE',KEYS[1],ARGV[1]) then return 2 end
if redis.call('ZCARD',KEYS[1])>=tonumber(ARGV[2]) then return 0 end
redis.call('ZADD',KEYS[1],ms+120000,ARGV[1]);redis.call('PEXPIRE',KEYS[1],120000);return 1`

func usageUnavailable() error {
	return xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "用量账本或容量服务暂不可用，未启动新的供应商调用")
}

func (g *UsageGovernor) Begin(ctx context.Context, attempt llm.UsageAttempt) error {
	if g == nil || g.facts == nil || g.redis == nil {
		return usageUnavailable()
	}
	if err := g.facts.Expire(ctx); err != nil {
		return usageUnavailable()
	}
	limit, err := g.facts.GlobalConcurrentLimit(ctx)
	if err != nil || limit < 1 || limit > 64 {
		return usageUnavailable()
	}
	result, err := g.redis.EvalCtx(ctx, usageAcquireLease, []string{g.key}, attempt.AttemptID, limit)
	if err != nil {
		return usageUnavailable()
	}
	code, ok := result.(int64)
	if !ok || code == 2 {
		return usageUnavailable()
	}
	if code == 0 {
		return xerr.NewErrCodeMsg(429, "AI供应商请求容量已满，请稍后重试")
	}
	if err := g.facts.Reserve(ctx, attempt); err != nil {
		if errors.Is(err, model.ErrUsageCommitted) {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			if cleanupErr := g.facts.Finish(cleanup, attempt.AttemptID, llm.UsageObservation{Status: "not_sent", UsageState: "unknown", ErrorCode: "admission_cache"}); cleanupErr != nil && !errors.Is(cleanupErr, model.ErrUsageCommitted) && !errors.Is(cleanupErr, model.ErrNotFound) {
				logx.WithContext(ctx).Error("用量准入补偿未确认，预留将按未知状态核对")
			}
			cancel()
		}
		if cleanupErr := g.release(ctx, attempt.AttemptID); cleanupErr != nil {
			logx.WithContext(ctx).Error("用量准入失败后的容量租约未确认释放，将等待过期")
		}
		if errors.Is(err, model.ErrUsageQuota) {
			return xerr.NewErrCodeMsg(429, "每日供应商请求额度已用完")
		}
		if errors.Is(err, model.ErrUsageCapacity) {
			return xerr.NewErrCodeMsg(429, "AI供应商请求容量已满，请稍后重试")
		}
		return usageUnavailable()
	}
	return nil
}

func (g *UsageGovernor) Dispatched(ctx context.Context, attempt string) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := g.facts.MarkDispatched(cleanup, attempt); err != nil {
		return usageUnavailable()
	}
	return nil
}

func (g *UsageGovernor) Finish(ctx context.Context, attempt string, observation llm.UsageObservation) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	err := g.facts.Finish(cleanup, attempt, observation)
	cancel()
	// 本进程请求已结束可释放容量；落库失败仍保留DB预留，不释放每日额度。
	releaseErr := g.release(ctx, attempt)
	if err != nil || releaseErr != nil {
		return usageUnavailable()
	}
	return nil
}

func (g *UsageGovernor) release(ctx context.Context, attempt string) error {
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	_, err := g.redis.ZremCtx(cleanup, g.key, attempt)
	return err
}

func (g *UsageGovernor) Active(ctx context.Context) (int64, error) {
	if g == nil || g.facts == nil {
		return 0, usageUnavailable()
	}
	count, err := g.facts.Active(ctx)
	if err != nil {
		return 0, usageUnavailable()
	}
	return count, nil
}
