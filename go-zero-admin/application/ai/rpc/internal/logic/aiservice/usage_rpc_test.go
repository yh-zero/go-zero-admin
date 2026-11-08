package aiservicelogic

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/xerr"
)

func TestUsageRPCForcesSelfOwnershipAndRejectsAdmin(t *testing.T) {
	dsn, address := os.Getenv("GOZERO_AI_TEST_DSN"), os.Getenv("GOZERO_AI_TEST_REDIS")
	if dsn == "" || address == "" {
		t.Skip("需要显式13317/16417隔离环境")
	}
	parsed, err := mysql.ParseDSN(dsn)
	if err != nil || parsed.Addr != "127.0.0.1:13317" || parsed.DBName != "gozero_admin" || address != "127.0.0.1:16417" {
		t.Fatal("只允许授权隔离环境")
	}
	sqlx.DisableLog()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	owner := time.Now().UnixNano()
	other := owner + 1
	t.Logf("用量RPC夹具owner=%d/%d，结束按ID精确清理", owner, other)
	store := model.NewUsageStore(sqlx.NewSqlConnFromDB(db), cache.CacheConf{{RedisConf: redis.RedisConf{Host: address, Type: "node", NonBlock: true}, Weight: 100}})
	t.Cleanup(func() {
		clean, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		rdb := goredis.NewClient(&goredis.Options{Addr: address})
		defer rdb.Close()
		for _, id := range []int64{owner, other} {
			keys := []string{fmt.Sprintf("cache:gozeroAdmin:aiUsagePolicy:ownerId:%d", id)}
			rows, err := db.QueryContext(clean, "SELECT id,attempt_id FROM ai_usage_call WHERE owner_id=?", id)
			if err != nil {
				t.Error("无法核对本轮调用缓存键")
				continue
			}
			for rows.Next() {
				var callID int64
				var attempt string
				if err := rows.Scan(&callID, &attempt); err != nil {
					t.Error("调用缓存键读取失败")
					continue
				}
				keys = append(keys, fmt.Sprintf("cache:gozeroAdmin:aiUsageCall:id:%d", callID), "cache:gozeroAdmin:aiUsageCall:attemptId:"+attempt)
			}
			_ = rows.Close()
			for _, table := range []string{"ai_usage_call", "ai_usage_policy"} {
				if _, err := db.ExecContext(clean, "DELETE FROM "+table+" WHERE owner_id=?", id); err != nil {
					t.Error("精确用量RPC夹具清理失败")
				}
				var count int
				if err := db.QueryRowContext(clean, "SELECT COUNT(*) FROM "+table+" WHERE owner_id=?", id).Scan(&count); err != nil || count != 0 {
					t.Error("本轮RPC用量夹具清理未归零")
				}
			}
			if err := rdb.Del(clean, keys...).Err(); err != nil {
				t.Error("本轮RPC缓存键精确清理失败")
			}
		}
	})
	for _, item := range []struct{ id, total int64 }{{owner, 2}, {other, 99}} {
		attempt := llm.UsageAttempt{UsageIdentity: llm.UsageIdentity{UserID: item.id, Source: "analysis", RequestID: uuid.NewString(), StepIndex: 0}, AttemptID: uuid.NewString(), Operation: "chat", Provider: "test", Model: "local-test", InputItems: 1}
		if err := store.Reserve(ctx, attempt); err != nil {
			t.Fatal(err)
		}
		prompt, completion := int64(1), item.total-1
		if err := store.Finish(ctx, attempt.AttemptID, llm.UsageObservation{Status: "succeeded", UsageState: "complete", DispatchConfirmed: true, PromptTokens: &prompt, CompletionTokens: &completion, TotalTokens: &item.total}); err != nil {
			t.Fatal(err)
		}
	}
	s := &svc.ServiceContext{UsageStore: store}
	in := &aipb.UsageQueryRequest{Actor: &aipb.Actor{UserId: owner}, UserId: other}
	result, err := NewGetUsageSummaryLogic(ctx, s).GetUsageSummary(in)
	if err != nil || result.Summary == nil || result.Summary.ConfirmedRequests != 1 || result.Summary.KnownTotalTokens != 2 {
		t.Fatalf("本人RPC不能采用客户端user_id：result=%v err=%v", result, err)
	}
	_, err = NewAdminGetUsageSummaryLogic(ctx, s).AdminGetUsageSummary(in)
	if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 403 {
		t.Fatalf("非超管RPC必须拒绝全用户账本：%v", err)
	}
}
