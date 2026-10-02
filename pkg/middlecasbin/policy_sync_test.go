package middlecasbin

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/glebarez/sqlite"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func policyDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&gormadapter.CasbinRule{}, &PolicyVersion{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&PolicyVersion{ID: 1}).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func changePolicy(t *testing.T, db *gorm.DB, change func(*gorm.DB) error) {
	t.Helper()
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := change(tx); err != nil {
			return err
		}
		return BumpPolicyVersion(tx)
	}); err != nil {
		t.Fatal(err)
	}
}

func TestMissedNotificationIsRecoveredAcrossInstances(t *testing.T) {
	db := policyDB(t)
	rule := gormadapter.CasbinRule{Ptype: "p", V0: "88", V1: "/resource", V2: "GET"}
	changePolicy(t, db, func(tx *gorm.DB) error { return tx.Create(&rule).Error })
	_, first, err := (CasbinConf{}).newPolicySynchronizer(db, redis.RedisConf{}, time.Hour, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(first.Close)
	secondEnforcer, second, err := (CasbinConf{}).newPolicySynchronizer(db, redis.RedisConf{}, 5*time.Millisecond, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(second.Close)
	allowed, err := secondEnforcer.Enforce("88", "/resource", "GET")
	if err != nil || !allowed {
		t.Fatalf("initial grant missing: %v", err)
	}
	changePolicy(t, db, func(tx *gorm.DB) error { return tx.Delete(&rule).Error })
	// No Notify call: this simulates a lost publication or a disconnected subscriber.
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		allowed, err = secondEnforcer.Enforce("88", "/resource", "GET")
		if err != nil {
			t.Fatal(err)
		}
		if !allowed {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("second instance retained revoked permission after missed notification")
}

func TestLoadFailureRetriesWithoutMarkingVersionCurrent(t *testing.T) {
	db := policyDB(t)
	rule := gormadapter.CasbinRule{Ptype: "p", V0: "88", V1: "/resource", V2: "GET"}
	changePolicy(t, db, func(tx *gorm.DB) error { return tx.Create(&rule).Error })
	enforcer, synchronizer, err := (CasbinConf{}).newPolicySynchronizer(db, redis.RedisConf{}, time.Hour, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(synchronizer.Close)
	changePolicy(t, db, func(tx *gorm.DB) error { return tx.Delete(&rule).Error })
	if err := db.Callback().Query().Before("gorm:query").Register("test:load-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == "casbin_rule" {
			tx.AddError(errors.New("injected query failure"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := synchronizer.Sync(context.Background()); err == nil {
		t.Fatal("load failure was hidden")
	}
	if err := db.Callback().Query().Remove("test:load-failure"); err != nil {
		t.Fatal(err)
	}
	if err := synchronizer.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	allowed, err := enforcer.Enforce("88", "/resource", "GET")
	if err != nil || allowed {
		t.Fatalf("recovered load retained stale grant: %v", err)
	}
}

func TestCommitDuringReloadDoesNotLoseNewerVersion(t *testing.T) {
	db := policyDB(t)
	enforcer, synchronizer, err := (CasbinConf{}).newPolicySynchronizer(db, redis.RedisConf{}, time.Hour, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(synchronizer.Close)
	changePolicy(t, db, func(tx *gorm.DB) error {
		return tx.Create(&gormadapter.CasbinRule{Ptype: "p", V0: "88", V1: "/first", V2: "GET"}).Error
	})
	var inserted atomic.Bool
	if err := db.Callback().Query().After("gorm:query").Register("test:concurrent-commit", func(tx *gorm.DB) {
		if tx.Statement.Table != "casbin_rule" || !inserted.CompareAndSwap(false, true) {
			return
		}
		changePolicy(t, db, func(writer *gorm.DB) error {
			return writer.Create(&gormadapter.CasbinRule{Ptype: "p", V0: "88", V1: "/second", V2: "GET"}).Error
		})
	}); err != nil {
		t.Fatal(err)
	}
	if err := synchronizer.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	allowed, err := enforcer.Enforce("88", "/second", "GET")
	if err != nil || !allowed {
		t.Fatalf("concurrent version was lost: %v", err)
	}
}

func TestVersionRollbackCancellationAndClose(t *testing.T) {
	db := policyDB(t)
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := BumpPolicyVersion(tx); err != nil {
			return err
		}
		return errors.New("rollback")
	}); err == nil {
		t.Fatal("expected transaction rollback")
	}
	var version PolicyVersion
	if err := db.First(&version, 1).Error; err != nil {
		t.Fatal(err)
	}
	if version.Version != 0 {
		t.Fatal("failed mutation advanced durable version")
	}
	_, synchronizer, err := (CasbinConf{}).newPolicySynchronizer(db, redis.RedisConf{}, time.Hour, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := synchronizer.Sync(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation not propagated: %v", err)
	}
	synchronizer.Close()
	synchronizer.Close()
	select {
	case <-synchronizer.done:
	default:
		t.Fatal("worker still running after Close")
	}
}

func TestMissingMigrationStopsInitialization(t *testing.T) {
	db := policyDB(t)
	if err := db.Migrator().DropTable(&PolicyVersion{}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := (CasbinConf{}).newPolicySynchronizer(db, redis.RedisConf{}, time.Hour, time.Second); err == nil {
		t.Fatal("missing version migration was accepted")
	}
}
