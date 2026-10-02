package middlecasbin

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	redisclient "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type blockedPublisher struct {
	redisclient.UniversalClient
	started chan struct{}
	release chan struct{}
	calls   atomic.Int64
}

func (p *blockedPublisher) Publish(ctx context.Context, channel string, message interface{}) *redisclient.IntCmd {
	p.calls.Add(1)
	p.started <- struct{}{}
	result := redisclient.NewIntCmd(ctx)
	select {
	case <-p.release:
		result.SetVal(1)
	case <-ctx.Done():
		result.SetErr(ctx.Err())
	}
	return result
}
func (*blockedPublisher) Close() error { return nil }

func TestPolicyNotifyDoesNotWaitForSlowRedis(t *testing.T) {
	db := policyDB(t)
	_, synchronizer, err := (CasbinConf{}).newPolicySynchronizer(db, redis.RedisConf{}, time.Hour, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	publisher := &blockedPublisher{started: make(chan struct{}, 4), release: make(chan struct{})}
	synchronizer.redis = publisher
	t.Cleanup(synchronizer.Close)
	returned := make(chan struct{})
	go func() { synchronizer.Notify(); close(returned) }()
	select {
	case <-returned:
	case <-time.After(150 * time.Millisecond):
		t.Fatal("Notify blocked on Redis publication after the transaction committed")
	}
	select {
	case <-publisher.started:
	case <-time.After(time.Second):
		t.Fatal("queued policy notification was not published")
	}
	// Closing must cancel a pending publication rather than wait its full timeout.
	closed := make(chan struct{})
	go func() { synchronizer.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(300 * time.Millisecond):
		t.Fatal("Close did not cancel the pending publication")
	}
}

func TestPolicyNotifyCoalescesBurstWhilePublishing(t *testing.T) {
	db := policyDB(t)
	_, synchronizer, err := (CasbinConf{}).newPolicySynchronizer(db, redis.RedisConf{}, time.Hour, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	publisher := &blockedPublisher{started: make(chan struct{}, 4), release: make(chan struct{})}
	synchronizer.redis = publisher
	t.Cleanup(synchronizer.Close)
	synchronizer.Notify()
	select {
	case <-publisher.started:
	case <-time.After(time.Second):
		t.Fatal("initial publication did not start")
	}
	for range 1000 {
		synchronizer.Notify()
	}
	close(publisher.release)
	select {
	case <-publisher.started:
	case <-time.After(time.Second):
		t.Fatal("pending coalesced notification was lost")
	}
	select {
	case <-publisher.started:
		t.Fatal("notification burst was not coalesced")
	case <-time.After(50 * time.Millisecond):
	}
	if publisher.calls.Load() != 2 {
		t.Fatalf("want one in-flight and one pending publication, got %d", publisher.calls.Load())
	}
}
