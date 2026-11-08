package svc

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/xerr"
)

type usageGovernorHarness struct {
	ctx      context.Context
	db       *sql.DB
	store    *model.UsageStore
	redis    *redis.Redis
	owner    int64
	leaseKey string
}

func newUsageGovernorHarness(t *testing.T, dailyLimit int64) *usageGovernorHarness {
	t.Helper()
	dsn, address := os.Getenv("GOZERO_AI_TEST_DSN"), os.Getenv("GOZERO_AI_TEST_REDIS")
	if dsn == "" || address == "" {
		t.Skip("需要13317/16417隔离数据库和Redis")
	}
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil || parsed.Net != "tcp" || parsed.Addr != "127.0.0.1:13317" || parsed.DBName != "gozero_admin" || address != "127.0.0.1:16417" {
		t.Fatal("用量租约回归只能写入已授权隔离环境")
	}
	sqlx.DisableLog()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("隔离数据库连接参数无效")
	}
	t.Cleanup(func() { _ = db.Close() })
	rdb, err := redis.NewRedis(redis.RedisConf{Host: address, Type: "node"})
	if err != nil || !rdb.PingCtx(ctx) {
		t.Fatal("隔离Redis不可达")
	}
	caches := cache.CacheConf{{RedisConf: redis.RedisConf{Host: address, Type: "node"}, Weight: 100}}
	h := &usageGovernorHarness{ctx: ctx, db: db, store: model.NewUsageStore(sqlx.NewSqlConnFromDB(db), caches), redis: rdb, owner: time.Now().UnixNano(), leaseKey: "test:usage:active:" + uuid.NewString()}
	if _, err := db.ExecContext(ctx, "INSERT INTO ai_usage_policy(owner_id,daily_request_limit) VALUES (?,?)", h.owner, dailyLimit); err != nil {
		t.Fatal("隔离策略夹具建立失败")
	}
	t.Logf("用量租约回归 owner_id=%d，唯一Redis key=%s", h.owner, h.leaseKey)
	t.Cleanup(func() {
		clean, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := rdb.DelCtx(clean, h.leaseKey, fmt.Sprintf("cache:gozeroAdmin:aiUsagePolicy:ownerId:%d", h.owner)); err != nil {
			t.Error("精确清理本轮用量租约/策略缓存失败")
		}
		for _, query := range []string{"DELETE FROM ai_usage_call WHERE owner_id=?", "DELETE FROM ai_usage_policy WHERE owner_id=?"} {
			if _, err := db.ExecContext(clean, query, h.owner); err != nil {
				t.Error("精确清理本轮用量账本失败")
			}
		}
		for _, table := range []string{"ai_usage_call", "ai_usage_policy"} {
			var count int
			if err := db.QueryRowContext(clean, "SELECT COUNT(*) FROM "+table+" WHERE owner_id=?", h.owner).Scan(&count); err != nil || count != 0 {
				t.Error("用量租约夹具清理后仍存在或无法核对")
			}
		}
	})
	return h
}

func (h *usageGovernorHarness) attempt() llm.UsageAttempt {
	return llm.UsageAttempt{UsageIdentity: llm.UsageIdentity{UserID: h.owner, Source: "chat", RequestID: uuid.NewString(), StepIndex: -1}, AttemptID: uuid.NewString(), Operation: "chat", Provider: "test", Model: "local-test", InputItems: 1}
}

type fixedUsageCapacity struct{ *model.UsageStore }

func (*fixedUsageCapacity) GlobalConcurrentLimit(context.Context) (int64, error) { return 1, nil }

// 只在提交后的缓存边界注入错误；真实MySQL预留和补偿仍由生产Store执行。
type committedUsageCacheFailure struct{ *model.UsageStore }

func (s *committedUsageCacheFailure) Reserve(ctx context.Context, attempt llm.UsageAttempt) error {
	if err := s.UsageStore.Reserve(ctx, attempt); err != nil {
		return err
	}
	return model.ErrUsageCommitted
}

func requireUsageCode(t *testing.T, err error, want uint32) {
	t.Helper()
	code, ok := xerr.AsCodeError(err)
	if !ok || code.GetErrCode() != want {
		t.Fatalf("准入错误码不符合预期：类型=%T，期望=%d", err, want)
	}
}

func TestUsageGovernorGlobalLeaseSpansCompleteCall(t *testing.T) {
	h := newUsageGovernorHarness(t, 10)
	governor := NewUsageGovernor(&fixedUsageCapacity{h.store}, h.redis, h.leaseKey)
	first, second := h.attempt(), h.attempt()
	if err := governor.Begin(h.ctx, first); err != nil {
		t.Fatal(err)
	}
	if active, err := governor.Active(h.ctx); err != nil || active != 1 {
		t.Fatal("供应商调用活跃期间必须持有租约")
	}
	requireUsageCode(t, governor.Begin(h.ctx, second), 429)
	if err := governor.Dispatched(h.ctx, first.AttemptID); err != nil {
		t.Fatal(err)
	}
	if err := governor.Finish(h.ctx, first.AttemptID, llm.UsageObservation{Status: "failed", UsageState: "unknown", DispatchConfirmed: true}); err != nil {
		t.Fatal(err)
	}
	if active, err := governor.Active(h.ctx); err != nil || active != 0 {
		t.Fatal("失败结束也必须精确释放活跃租约")
	}
	if err := governor.Begin(h.ctx, second); err != nil {
		t.Fatal("前一调用结束后应释放容量")
	}
	if err := governor.Finish(h.ctx, second.AttemptID, llm.UsageObservation{Status: "not_sent", UsageState: "unknown"}); err != nil {
		t.Fatal(err)
	}
}

func TestUsageGovernorRedisLossCannotResetDailyQuota(t *testing.T) {
	h := newUsageGovernorHarness(t, 1)
	governor := NewUsageGovernor(h.store, h.redis, h.leaseKey)
	first := h.attempt()
	if err := governor.Begin(h.ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := governor.Finish(h.ctx, first.AttemptID, llm.UsageObservation{Status: "unknown", UsageState: "unknown"}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.redis.DelCtx(h.ctx, h.leaseKey); err != nil {
		t.Fatal(err)
	}
	requireUsageCode(t, governor.Begin(h.ctx, h.attempt()), 429)
	if active, err := governor.Active(h.ctx); err != nil || active != 0 {
		t.Fatal("SQL日额度拒绝后不能泄漏本轮新取得的Redis租约")
	}
}

func TestUsageGovernorNotSentRefundAndCacheFailureCompensation(t *testing.T) {
	h := newUsageGovernorHarness(t, 1)
	governor := NewUsageGovernor(h.store, h.redis, h.leaseKey)
	first := h.attempt()
	if err := governor.Begin(h.ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := governor.Finish(h.ctx, first.AttemptID, llm.UsageObservation{Status: "not_sent", UsageState: "unknown"}); err != nil {
		t.Fatal(err)
	}
	badGovernor := NewUsageGovernor(&committedUsageCacheFailure{h.store}, h.redis, h.leaseKey)
	second := h.attempt()
	requireUsageCode(t, badGovernor.Begin(h.ctx, second), 503)
	var status string
	var charged int
	if err := h.db.QueryRowContext(h.ctx, "SELECT status,quota_charged FROM ai_usage_call WHERE attempt_id=? AND owner_id=?", second.AttemptID, h.owner).Scan(&status, &charged); err != nil || status != "not_sent" || charged != 0 {
		t.Fatal("预留已提交但缓存失败时，尚未发出调用必须补偿为not_sent并退款")
	}
	if active, err := governor.Active(h.ctx); err != nil || active != 0 {
		t.Fatal("预留失败补偿后不能泄漏活跃租约")
	}
}

func TestUsageGovernorRedisLossCannotResetGlobalCapacity(t *testing.T) {
	h := newUsageGovernorHarness(t, 10)
	other := newUsageGovernorHarness(t, 10)
	var previousLimit, active int64
	if err := h.db.QueryRowContext(h.ctx, "SELECT global_concurrent_limit FROM ai_usage_policy WHERE owner_id=0").Scan(&previousLimit); err != nil {
		t.Fatal(err)
	}
	if err := h.db.QueryRowContext(h.ctx, "SELECT COUNT(*) FROM ai_usage_call WHERE status IN ('reserved','running') AND lease_until>NOW(3)").Scan(&active); err != nil || active != 0 {
		t.Fatal("全局并发回归需要隔离库无其他活跃调用，不能清理他人夹具")
	}
	if _, err := h.db.ExecContext(h.ctx, "UPDATE ai_usage_policy SET global_concurrent_limit=1 WHERE owner_id=0 AND global_concurrent_limit=?", previousLimit); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		clean, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := h.db.ExecContext(clean, "UPDATE ai_usage_policy SET global_concurrent_limit=? WHERE owner_id=0 AND global_concurrent_limit=1", previousLimit); err != nil {
			t.Error("恢复本轮临时全局并发策略失败")
		}
	})
	governor := NewUsageGovernor(h.store, h.redis, h.leaseKey)
	first, second := h.attempt(), other.attempt()
	if err := governor.Begin(h.ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := governor.Dispatched(h.ctx, first.AttemptID); err != nil {
		t.Fatal(err)
	}
	// 只删除本轮UUID活跃key，模拟Redis丢失，保留真实MySQL未结束调用。
	if _, err := h.redis.DelCtx(h.ctx, h.leaseKey); err != nil {
		t.Fatal(err)
	}
	if count, err := governor.Active(h.ctx); err != nil || count != 1 {
		t.Errorf("Redis丢key不能把已知数据库活跃调用报告为0：count=%d err=%v", count, err)
	}
	err := governor.Begin(h.ctx, second)
	if code, ok := xerr.AsCodeError(err); !ok || code.GetErrCode() != 429 {
		t.Errorf("Redis丢key后不同用户仍须受全局上限1约束：err=%v", err)
		if err == nil {
			_ = governor.Finish(h.ctx, second.AttemptID, llm.UsageObservation{Status: "not_sent", UsageState: "unknown"})
		}
	}
	if err := governor.Finish(h.ctx, first.AttemptID, llm.UsageObservation{Status: "failed", UsageState: "unknown", DispatchConfirmed: true}); err != nil {
		t.Fatal(err)
	}
	if count, err := governor.Active(h.ctx); err != nil || count != 0 {
		t.Fatal("调用结束后数据库活跃容量必须归零")
	}
	third := other.attempt()
	if err := governor.Begin(h.ctx, third); err != nil {
		t.Fatal("真实活跃调用结束后应可继续准入", err)
	}
	if err := governor.Finish(h.ctx, third.AttemptID, llm.UsageObservation{Status: "not_sent", UsageState: "unknown"}); err != nil {
		t.Fatal(err)
	}
	// 不同Redis key模拟快速挡板完全失联；不同用户同时争抢仍只有MySQL可最终准入。
	type outcome struct {
		governor *UsageGovernor
		attempt  llm.UsageAttempt
		err      error
	}
	results := make(chan outcome, 6)
	start := make(chan struct{})
	var workers sync.WaitGroup
	for i := range 6 {
		facts := h
		if i%2 == 1 {
			facts = other
		}
		g := NewUsageGovernor(facts.store, facts.redis, facts.leaseKey)
		a := facts.attempt()
		workers.Add(1)
		go func() { defer workers.Done(); <-start; results <- outcome{g, a, g.Begin(h.ctx, a)} }()
	}
	close(start)
	workers.Wait()
	close(results)
	var admitted int
	for result := range results {
		if result.err == nil {
			admitted++
			if err := result.governor.Finish(h.ctx, result.attempt.AttemptID, llm.UsageObservation{Status: "not_sent", UsageState: "unknown"}); err != nil {
				t.Fatal(err)
			}
		} else {
			requireUsageCode(t, result.err, 429)
		}
	}
	if admitted != 1 {
		t.Fatalf("不同用户/Redis租约key竞争上限1，MySQL只能准入1次：%d", admitted)
	}
}
