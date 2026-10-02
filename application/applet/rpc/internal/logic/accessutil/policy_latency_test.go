package accessutil_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	casbinlogic "go-zero-admin/application/applet/rpc/internal/logic/casbin"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/middlecasbin"
	"gorm.io/gorm"
)

func TestCommittedPolicyDoesNotWaitForCacheReload(t *testing.T) {
	s := service(t)
	must(t, s.DB.Create(&middlecasbin.PolicyVersion{ID: 1}).Error)
	rule := gormadapter.CasbinRule{Ptype: "p", V0: "88", V1: "/revocable", V2: "GET"}
	must(t, s.DB.Create(&rule).Error)
	enforcer, synchronizer, err := (middlecasbin.CasbinConf{}).NewPolicySynchronizer(s.DB.DB, redis.RedisConf{})
	must(t, err)
	s.Casbin, s.PolicySync = enforcer, synchronizer
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(func() { unblock(); synchronizer.Close() })
	must(t, s.DB.Callback().Query().Before("gorm:query").Register("test:blocked-policy-reload", func(tx *gorm.DB) {
		if tx.Statement.Table != "casbin_rule" {
			return
		}
		select {
		case started <- struct{}{}:
		default:
		}
		select {
		case <-release:
		case <-tx.Statement.Context.Done():
			tx.AddError(tx.Statement.Context.Err())
		}
	}))
	result := make(chan error, 1)
	go func() {
		result <- accessutil.PolicyTransaction(context.Background(), s, func(tx *gorm.DB) error { return tx.Delete(&rule).Error })
	}()
	select {
	case err := <-result:
		must(t, err)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("committed permission write waited for the background cache reload")
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("background policy reload did not start")
	}
	var version middlecasbin.PolicyVersion
	must(t, s.DB.First(&version, 1).Error)
	if version.Version != 1 {
		t.Fatal("permission mutation did not commit its durable version")
	}
	// A blocked local cache refresh must neither retain the write gate nor allow
	// enforcement from its old snapshot.
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancelWrite()
	must(t, accessutil.PolicyTransaction(writeCtx, s, func(*gorm.DB) error { return nil }))
	checkCtx, cancelCheck := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancelCheck()
	_, err = casbinlogic.NewEnforceLogic(checkCtx, s).Enforce(&pb.EnforceRequest{AuthorityId: "88", Path: "/revocable", Method: "GET"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("enforcement used a stale cache instead of failing closed: %v", err)
	}
	unblock()
	check, err := casbinlogic.NewEnforceLogic(context.Background(), s).Enforce(&pb.EnforceRequest{AuthorityId: "88", Path: "/revocable", Method: "GET"})
	must(t, err)
	if check.Pass {
		t.Fatal("revoked permission survived cache recovery")
	}
}
