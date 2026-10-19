package appaccountservicelogic

import (
	"context"
	"errors"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/pkg/appauth"
	"time"
)

const credentialLimitScript = `local n = redis.call('INCR', KEYS[1]); if n == 1 then redis.call('EXPIRE', KEYS[1], ARGV[1]) end; return n`

func credentialLimit(ctx context.Context, s *svc.ServiceContext, action, account, ip string) error {
	if s == nil || s.BizRedis == nil {
		return errors.New("App credential rate limiter unavailable")
	}
	ttl := int64(15 * time.Minute / time.Second)
	accountMax, ipMax := int64(10), int64(50)
	if action == "register" {
		ttl = 3600
		accountMax = 5
		ipMax = 20
	}
	for _, item := range []struct {
		kind, value string
		max         int64
	}{{"ip", ip, ipMax}, {"account", account, accountMax}} {
		key := "app:credential:" + action + ":" + item.kind + ":" + appauth.Digest(item.value)
		value, err := s.BizRedis.EvalCtx(ctx, credentialLimitScript, []string{key}, ttl)
		if err != nil {
			return errors.New("App credential rate limiter unavailable")
		}
		count, ok := value.(int64)
		if !ok {
			return errors.New("App credential rate limiter unavailable")
		}
		if count > item.max {
			return badInput("尝试次数过多，请稍后再试")
		}
	}
	return nil
}
