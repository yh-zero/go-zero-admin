package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/pkg/llm"
)

type usageStoreHarness struct {
	ctx   context.Context
	db    *sql.DB
	store *UsageStore
	owner int64
}

func newUsageStoreHarness(t *testing.T, dailyLimit int64) *usageStoreHarness {
	t.Helper()
	dsn, redisAddress := os.Getenv("GOZERO_AI_TEST_DSN"), os.Getenv("GOZERO_AI_TEST_REDIS")
	if dsn == "" || redisAddress == "" {
		t.Skip("需要显式隔离 GOZERO_AI_TEST_DSN / GOZERO_AI_TEST_REDIS")
	}
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil || parsed.Net != "tcp" || parsed.Addr != "127.0.0.1:13317" || parsed.DBName != "gozero_admin" || redisAddress != "127.0.0.1:16417" {
		t.Fatal("用量写入回归仅允许已授权的13317/16417隔离环境")
	}
	sqlx.DisableLog()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("隔离数据库连接参数无效")
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = db.PingContext(ctx); err != nil {
		t.Fatal("隔离数据库不可达")
	}
	caches := cache.CacheConf{{RedisConf: redis.RedisConf{Host: redisAddress, Type: "node"}, Weight: 100}}
	h := &usageStoreHarness{ctx: ctx, db: db, store: NewUsageStore(sqlx.NewSqlConnFromDB(db), caches), owner: time.Now().UnixNano()}
	if _, err = db.ExecContext(ctx, "INSERT INTO ai_usage_policy(owner_id,daily_request_limit,daily_token_warning,global_concurrent_limit,version) VALUES (?,?,0,8,1)", h.owner, dailyLimit); err != nil {
		t.Fatal("隔离用量策略夹具未建立，请先通过来源DDL初始化隔离库")
	}
	t.Logf("用量回归 owner_id=%d，数据库/缓存仅13317/16417，结束后精确清理", h.owner)
	t.Cleanup(func() {
		clean, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, query := range []string{"DELETE FROM ai_usage_call WHERE owner_id=?", "DELETE FROM ai_usage_policy WHERE owner_id=?"} {
			if _, err := db.ExecContext(clean, query, h.owner); err != nil {
				t.Error("本轮用量夹具精确清理失败")
			}
		}
		if err := h.store.cached.DelCacheCtx(clean, fmt.Sprintf("%s%d", cacheGozeroAdminAiUsagePolicyOwnerIdPrefix, h.owner)); err != nil {
			t.Error("本轮策略缓存清理失败")
		}
		for _, table := range []string{"ai_usage_call", "ai_usage_policy"} {
			var count int
			if err := db.QueryRowContext(clean, "SELECT COUNT(*) FROM "+table+" WHERE owner_id=?", h.owner).Scan(&count); err != nil || count != 0 {
				t.Error("本轮用量夹具清理后仍存在或无法核对")
			}
		}
	})
	return h
}

func (h *usageStoreHarness) attempt() llm.UsageAttempt {
	return llm.UsageAttempt{UsageIdentity: llm.UsageIdentity{UserID: h.owner, Source: "analysis", RequestID: uuid.NewString(), SourceID: 7, StepIndex: 0}, AttemptID: uuid.NewString(), Operation: "chat", Provider: "test", Model: "local-test", InputItems: 1}
}

func TestUsageStoreSerializesConcurrentDailyQuota(t *testing.T) {
	h := newUsageStoreHarness(t, 1)
	var admitted, denied atomic.Int64
	var wg sync.WaitGroup
	start := make(chan struct{})
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			err := h.store.Reserve(h.ctx, h.attempt())
			switch {
			case err == nil:
				admitted.Add(1)
			case errors.Is(err, ErrUsageQuota):
				denied.Add(1)
			default:
				var databaseError *mysql.MySQLError
				if errors.As(err, &databaseError) {
					t.Errorf("并发预留数据库错误码=%d", databaseError.Number)
				} else {
					t.Errorf("并发预留出现额度以外错误，类型=%T", err)
				}
			}
		}()
	}
	close(start)
	wg.Wait()
	if admitted.Load() != 1 || denied.Load() != 11 {
		t.Fatalf("额度1应恰好准入一次：准入=%d，拒绝=%d", admitted.Load(), denied.Load())
	}
	var charged int
	if err := h.db.QueryRowContext(h.ctx, "SELECT COUNT(*) FROM ai_usage_call WHERE owner_id=? AND quota_charged=1", h.owner).Scan(&charged); err != nil || charged != 1 {
		t.Fatal("数据库保留的额度与准入次数不一致")
	}
}

func TestUsageStoreReleasesNotSentButKeepsUnknownCharged(t *testing.T) {
	h := newUsageStoreHarness(t, 1)
	first := h.attempt()
	if err := h.store.Reserve(h.ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := h.store.Finish(h.ctx, first.AttemptID, llm.UsageObservation{Status: "not_sent", UsageState: "unknown", ErrorCode: "sdk_validation"}); err != nil {
		t.Fatal(err)
	}
	second := h.attempt()
	if err := h.store.Reserve(h.ctx, second); err != nil {
		t.Fatal("未dispatch失败必须释放预留额度")
	}
	if err := h.store.Finish(h.ctx, second.AttemptID, llm.UsageObservation{Status: "unknown", UsageState: "unknown", ErrorCode: "interrupted"}); err != nil {
		t.Fatal(err)
	}
	if err := h.store.Reserve(h.ctx, h.attempt()); !errors.Is(err, ErrUsageQuota) {
		t.Fatal("未知调用不得按免费释放额度")
	}
	var status string
	var charged, dispatched int
	if err := h.db.QueryRowContext(h.ctx, "SELECT status,quota_charged,dispatch_confirmed FROM ai_usage_call WHERE attempt_id=?", second.AttemptID).Scan(&status, &charged, &dispatched); err != nil || status != "unknown" || charged != 1 || dispatched != 0 {
		t.Fatal("未知预留必须保留额度，且不能冒充已确认provider调用")
	}
}

func TestUsageStorePreservesKnownUsageForFailedResponseAndRejectsLateRefund(t *testing.T) {
	h := newUsageStoreHarness(t, 1)
	a := h.attempt()
	if err := h.store.Reserve(h.ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := h.store.MarkDispatched(h.ctx, a.AttemptID); err != nil {
		t.Fatal(err)
	}
	prompt, completion, total := int64(4), int64(2), int64(6)
	observation := llm.UsageObservation{Status: "failed", UsageState: "complete", PromptTokens: &prompt, CompletionTokens: &completion, TotalTokens: &total, HTTPStatus: 200, ProviderRequestID: "req-local-1", ErrorCode: "response_validation", DispatchConfirmed: true}
	if err := h.store.Finish(h.ctx, a.AttemptID, observation); err != nil {
		t.Fatal(err)
	}
	if err := h.store.Finish(h.ctx, a.AttemptID, observation); err != nil {
		t.Fatal("相同终态重写必须幂等")
	}
	_ = h.store.Finish(h.ctx, a.AttemptID, llm.UsageObservation{Status: "not_sent", UsageState: "unknown"})
	var state, usage string
	var known, charged, confirmed int64
	if err := h.db.QueryRowContext(h.ctx, "SELECT status,usage_state,total_tokens,quota_charged,dispatch_confirmed FROM ai_usage_call WHERE attempt_id=?", a.AttemptID).Scan(&state, &usage, &known, &charged, &confirmed); err != nil || state != "failed" || usage != "complete" || known != 6 || charged != 1 || confirmed != 1 {
		t.Fatal("业务失败不能抹掉已知用量，迟到not_sent也不能退款")
	}
}

func TestUsageStoreOmitsUnsafeProviderRequestID(t *testing.T) {
	h := newUsageStoreHarness(t, 1)
	a := h.attempt()
	if err := h.store.Reserve(h.ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := h.store.Finish(h.ctx, a.AttemptID, llm.UsageObservation{Status: "failed", UsageState: "unknown", ProviderRequestID: "unsafe\n" + strings.Repeat("x", 200), ErrorCode: "http_failure", DispatchConfirmed: true}); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := h.db.QueryRowContext(h.ctx, "SELECT provider_request_id FROM ai_usage_call WHERE attempt_id=?", a.AttemptID).Scan(&id); err != nil || id != "" {
		t.Fatal("供应商request_id非法字符或长度不能进入可回显账本")
	}
}

func TestUsagePeriodUsesShanghaiDaysAndBoundsHistory(t *testing.T) {
	now := time.Date(2026, 11, 7, 19, 30, 0, 0, time.UTC)
	from, to, _, _, err := usagePeriod("", "", now)
	if err != nil || from != "2026-11-02" || to != "2026-11-08" {
		t.Fatalf("默认7日必须按上海日历计算，from=%s to=%s err=%v", from, to, err)
	}
	if _, _, _, _, err := usagePeriod("2026-11-01", "2026-12-01", now); err != nil {
		t.Fatal("31个上海自然日应允许")
	}
	for _, pair := range [][2]string{{"2026-11-01", "2026-12-02"}, {"2026-11-09", "2026-11-08"}, {"2026-02-30", "2026-03-01"}, {"2026-1-01", "2026-01-02"}} {
		if _, _, _, _, err := usagePeriod(pair[0], pair[1], now); err == nil {
			t.Error("超31日、倒序或无效日期必须拒绝")
		}
	}
}

func TestUsageFilterRequiresExplicitOwnerOrAdminScope(t *testing.T) {
	if _, _, _, err := usageWhere(UsageFilter{}); !errors.Is(err, ErrUsageInvalid) {
		t.Fatal("未声明用户范围不能退化为查询全部账本")
	}
	if _, _, _, err := usageWhere(UsageFilter{OwnerId: 7, Source: "chat' OR 1=1"}); !errors.Is(err, ErrUsageInvalid) {
		t.Fatal("来源筛选必须来自固定枚举")
	}
	if _, _, _, err := usageWhere(UsageFilter{AllUsers: true, PageSize: 101}); !errors.Is(err, ErrUsageInvalid) {
		t.Fatal("超出分页边界必须拒绝")
	}
}

func TestUsageStoreSummarySeparatesKnownUnknownAndNotSent(t *testing.T) {
	h := newUsageStoreHarness(t, 10)
	prompt, completion, chatTotal, embedTotal := int64(4), int64(2), int64(6), int64(5)
	observations := []llm.UsageObservation{
		{Status: "failed", UsageState: "complete", PromptTokens: &prompt, CompletionTokens: &completion, TotalTokens: &chatTotal, DispatchConfirmed: true},
		{Status: "succeeded", UsageState: "complete", PromptTokens: &embedTotal, TotalTokens: &embedTotal, DispatchConfirmed: true},
		{Status: "failed", UsageState: "unknown", HTTPStatus: 503, DispatchConfirmed: true},
		{Status: "not_sent", UsageState: "unknown"},
	}
	for i, observation := range observations {
		a := h.attempt()
		if i == 1 || i == 2 {
			a.Operation, a.Source, a.BatchIndex = "embedding", "kb_ingest", int64(i-1)
		}
		if err := h.store.Reserve(h.ctx, a); err != nil {
			t.Fatal(err)
		}
		if err := h.store.Finish(h.ctx, a.AttemptID, observation); err != nil {
			t.Fatal(err)
		}
	}
	summary, err := h.store.Summary(h.ctx, UsageFilter{OwnerId: h.owner})
	if err != nil {
		t.Fatal(err)
	}
	if summary.ConfirmedRequests != 3 || summary.NotSentRequests != 1 || summary.CompleteRequests != 2 || summary.UnknownRequests != 1 || summary.FailedRequests != 2 || summary.KnownPromptTokens != 4 || summary.KnownCompletionTokens != 2 || summary.KnownEmbeddingTokens != 5 || summary.KnownTotalTokens != 11 || len(summary.Groups) != 2 {
		t.Fatalf("账本汇总不能把not_sent当已调用或把unknown当已知零：%+v", summary.UsageCounters)
	}
	rows, total, err := h.store.List(h.ctx, UsageFilter{OwnerId: h.owner, Operation: "embedding"})
	if err != nil || total != 2 || len(rows) != 2 || rows[0].CompletionTokens != nil || rows[0].TotalTokens != nil || rows[1].CompletionTokens != nil {
		t.Fatal("Embedding未知字段必须省略，不能伪造输出token或把失败批次填零")
	}
	if rows, count, err := h.store.List(h.ctx, UsageFilter{OwnerId: h.owner + 1}); err != nil || count != 0 || len(rows) != 0 {
		t.Fatal("本人列表不能跨用户读取账本")
	}
}

func TestUsageStoreUnconfirmedCancellationCountsAsUncertain(t *testing.T) {
	h := newUsageStoreHarness(t, 10)
	for _, status := range []string{"cancelled", "not_sent", "reserved"} {
		a := h.attempt()
		if err := h.store.Reserve(h.ctx, a); err != nil {
			t.Fatal(err)
		}
		if status != "reserved" {
			if err := h.store.Finish(h.ctx, a.AttemptID, llm.UsageObservation{Status: status, UsageState: "unknown"}); err != nil {
				t.Fatal(err)
			}
		}
	}
	summary, err := h.store.Summary(h.ctx, UsageFilter{OwnerId: h.owner})
	if err != nil {
		t.Fatal(err)
	}
	if summary.ConfirmedRequests != 0 || summary.UncertainRequests != 1 || summary.UnknownRequests != 1 || summary.CancelledRequests != 1 || summary.NotSentRequests != 1 || summary.ReservedRequests != 1 || summary.KnownTotalTokens != 0 || len(summary.Groups) != 1 || summary.Groups[0].UncertainRequests != 1 || summary.Groups[0].UnknownRequests != 1 {
		t.Errorf("未确认dispatch的保守扣额终态必须进入不确定与未知提醒，not_sent/预留另列：%+v", summary.UsageCounters)
	}
	policy, err := h.store.GetPolicy(h.ctx, h.owner)
	if err != nil || policy.UsedRequests != 2 || policy.RemainingRequests != 8 || policy.IncompleteRequests != 1 {
		t.Errorf("本人策略必须保留未知取消与预留额度，仅未知取消计入不完整：policy=%+v err=%v", policy, err)
	}
}

func TestUsageStorePolicyCASAndWarningUseCommittedLedger(t *testing.T) {
	h := newUsageStoreHarness(t, 10)
	a := h.attempt()
	if err := h.store.Reserve(h.ctx, a); err != nil {
		t.Fatal(err)
	}
	prompt, completion, total := int64(7), int64(4), int64(11)
	if err := h.store.Finish(h.ctx, a.AttemptID, llm.UsageObservation{Status: "succeeded", UsageState: "complete", PromptTokens: &prompt, CompletionTokens: &completion, TotalTokens: &total, DispatchConfirmed: true}); err != nil {
		t.Fatal(err)
	}
	warning := int64(10)
	policy, err := h.store.SetPolicy(h.ctx, h.owner, 1, UsagePolicyPatch{DailyTokenWarning: &warning})
	if err != nil || policy.Version != 2 || !policy.WarningReached || policy.KnownTokens != 11 || policy.UsedRequests != 1 || policy.RemainingRequests != 9 {
		t.Fatalf("策略提醒必须读取真实账本：policy=%+v err=%v", policy, err)
	}
	if _, err := h.store.SetPolicy(h.ctx, h.owner, 1, UsagePolicyPatch{DailyTokenWarning: &warning}); !errors.Is(err, ErrUsageConflict) {
		t.Fatal("旧策略版本必须冲突")
	}
	global, err := h.store.GetPolicy(h.ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	policy, err = h.store.SetPolicy(h.ctx, h.owner, 2, UsagePolicyPatch{InheritDefault: true})
	if err != nil || !policy.Inherited || policy.DailyRequestLimit != global.DailyRequestLimit || policy.Version != 3 {
		t.Fatal("恢复默认必须清除用户覆盖并保存新版本")
	}
}

func TestUsageStoreExpiredReservationRemainsChargedAndUnconfirmed(t *testing.T) {
	h := newUsageStoreHarness(t, 1)
	a := h.attempt()
	if err := h.store.Reserve(h.ctx, a); err != nil {
		t.Fatal(err)
	}
	if _, err := h.db.ExecContext(h.ctx, "UPDATE ai_usage_call SET lease_until=DATE_SUB(NOW(3),INTERVAL 1 SECOND) WHERE attempt_id=? AND owner_id=?", a.AttemptID, h.owner); err != nil {
		t.Fatal(err)
	}
	if err := h.store.expire(h.ctx, h.owner); err != nil {
		t.Fatal(err)
	}
	policy, err := h.store.GetPolicy(h.ctx, h.owner)
	if err != nil || policy.UsedRequests != 1 || policy.IncompleteRequests != 1 {
		t.Fatal("崩溃预留必须保守保留额度和未知用量")
	}
	var state string
	var confirmed int
	if err := h.db.QueryRowContext(h.ctx, "SELECT status,dispatch_confirmed FROM ai_usage_call WHERE attempt_id=?", a.AttemptID).Scan(&state, &confirmed); err != nil || state != "unknown" || confirmed != 0 {
		t.Fatal("过期预留不能冒充已确认dispatch")
	}
}
