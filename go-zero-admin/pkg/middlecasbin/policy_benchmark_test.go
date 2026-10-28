package middlecasbin

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/glebarez/sqlite"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type policyProfileKey struct{}
type policyProfile struct {
	start, queryStart       time.Time
	gateSetup, versionQuery time.Duration
	versionSeen             bool
}

// This isolates the policy guard/enforcement path with an in-memory database.
// It is deliberately not presented as HTTP latency or MySQL lock-wait evidence.
func BenchmarkPolicyGuard(b *testing.B) {
	for _, workers := range []int{1, 16} {
		b.Run(fmt.Sprintf("workers_%d", workers), func(b *testing.B) { benchmarkPolicyPath(b, workers, 100, false) })
	}
}
func BenchmarkPolicyReload(b *testing.B) {
	for _, rules := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprintf("rules_%d", rules), func(b *testing.B) { benchmarkPolicyPath(b, 1, rules, true) })
	}
}
func benchmarkPolicyPath(b *testing.B, workers, rules int, reload bool) {
	db, err := gorm.Open(sqlite.Open("file:policy-benchmark?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		b.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		b.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	b.Cleanup(func() { _ = sqlDB.Close() })
	if err = db.AutoMigrate(&gormadapter.CasbinRule{}, &PolicyVersion{}); err != nil {
		b.Fatal(err)
	}
	rows := make([]gormadapter.CasbinRule, rules)
	for i := range rows {
		rows[i] = gormadapter.CasbinRule{Ptype: "p", V0: "88", V1: fmt.Sprintf("/v1/benchmark/%d", i), V2: "GET"}
	}
	if err = db.CreateInBatches(rows, 100).Error; err != nil {
		b.Fatal(err)
	}
	if err = db.Create(&PolicyVersion{ID: 1}).Error; err != nil {
		b.Fatal(err)
	}
	enforcer, synchronizer, err := (CasbinConf{}).newPolicySynchronizer(db, redis.RedisConf{}, time.Hour, 30*time.Second)
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(synchronizer.Close)
	if err = db.Callback().Query().Before("gorm:query").Register("benchmark:version-before", func(tx *gorm.DB) {
		if tx.Statement.Table != "sys_policy_versions" {
			return
		}
		if sample, ok := tx.Statement.Context.Value(policyProfileKey{}).(*policyProfile); ok {
			sample.queryStart = time.Now()
			if !sample.versionSeen {
				sample.gateSetup = sample.queryStart.Sub(sample.start)
				sample.versionSeen = true
			}
		}
	}); err != nil {
		b.Fatal(err)
	}
	if err = db.Callback().Query().After("gorm:query").Register("benchmark:version-after", func(tx *gorm.DB) {
		if tx.Statement.Table != "sys_policy_versions" {
			return
		}
		if sample, ok := tx.Statement.Context.Value(policyProfileKey{}).(*policyProfile); ok {
			sample.versionQuery += time.Since(sample.queryStart)
		}
	}); err != nil {
		b.Fatal(err)
	}
	elapsed := make([]int64, b.N)
	gate := make([]int64, b.N)
	queries := make([]int64, b.N)
	var next atomic.Int64
	var failed atomic.Bool
	b.ReportAllocs()
	b.ResetTimer()
	var workersWG sync.WaitGroup
	for range workers {
		workersWG.Go(func() {
			for {
				index := int(next.Add(1)) - 1
				if index >= b.N {
					return
				}
				if reload {
					if err := db.Model(&PolicyVersion{}).Where("id = 1").UpdateColumn("version", gorm.Expr("version + 1")).Error; err != nil {
						failed.Store(true)
						return
					}
				}
				sample := &policyProfile{start: time.Now()}
				ctx := context.WithValue(context.Background(), policyProfileKey{}, sample)
				if err := synchronizer.Sync(ctx); err != nil {
					failed.Store(true)
					return
				}
				if !reload {
					allowed, err := enforcer.Enforce("88", "/v1/benchmark/0", "GET")
					if err != nil || !allowed {
						failed.Store(true)
						return
					}
				}
				elapsed[index] = time.Since(sample.start).Nanoseconds()
				gate[index] = sample.gateSetup.Nanoseconds()
				queries[index] = sample.versionQuery.Nanoseconds()
			}
		})
	}
	workersWG.Wait()
	b.StopTimer()
	if failed.Load() {
		b.Fatal("measured policy path failed")
	}
	b.ReportMetric(profileP95(elapsed), "path-p95-ns")
	b.ReportMetric(profileP95(gate), "gate-setup-p95-ns")
	b.ReportMetric(profileP95(queries), "version-query-p95-ns")
}
func profileP95(values []int64) float64 {
	if len(values) == 0 {
		return 0
	}
	slices.Sort(values)
	index := (95*len(values)+99)/100 - 1
	if index >= len(values) {
		index = len(values) - 1
	}
	return float64(values[index])
}
