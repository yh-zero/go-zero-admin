package middlecasbin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
)

func TestPolicyReloadObeysRequestDeadline(t *testing.T) {
	db := policyDB(t)
	_, synchronizer, err := (CasbinConf{}).newPolicySynchronizer(db, redis.RedisConf{}, time.Hour, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(synchronizer.Close)
	changePolicy(t, db, func(*gorm.DB) error { return nil })
	if err := db.Callback().Query().Before("gorm:query").Register("test:slow-policy-load", func(tx *gorm.DB) {
		if tx.Statement.Table == "casbin_rule" {
			<-tx.Statement.Context.Done()
			tx.AddError(tx.Statement.Context.Err())
		}
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	begin := time.Now()
	if err := synchronizer.Sync(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("reload ignored request deadline: %v", err)
	}
	if time.Since(begin) > 500*time.Millisecond {
		t.Fatal("reload used independent timeout instead of caller deadline")
	}
}
