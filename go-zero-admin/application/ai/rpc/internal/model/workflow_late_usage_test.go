package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestTerminalStepUsageAfterCancellationIsBoundToOriginalExecution(t *testing.T) {
	dsn, address := os.Getenv("GOZERO_AI_TEST_DSN"), os.Getenv("GOZERO_AI_TEST_REDIS")
	if dsn == "" || address == "" {
		t.Skip("需要本轮隔离MySQL和Redis")
	}
	sqlx.DisableLog()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	owner := time.Now().UnixNano()
	expected := WorkflowStepRecord{Index: 0, Id: "analysis", Name: "分析", Tool: "kb_analyze", Input: `{"kb_id":3,"query":"目标"}`, Status: "running", StartedAt: 123456789}
	raw, _ := json.Marshal([]WorkflowStepRecord{expected})
	inserted, err := db.ExecContext(ctx, "INSERT INTO ai_task(status,input,result,created_by,next_step,step_records,title) VALUES ('running','{}','',?,0,?,'迟到计量隔离回归')", owner, string(raw))
	if err != nil {
		t.Fatal(err)
	}
	id, _ := inserted.LastInsertId()
	t.Logf("本轮late usage task_id=%d owner=%d；结束按二者精确清理", id, owner)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if _, err := db.ExecContext(cleanup, "DELETE FROM ai_task WHERE id=? AND created_by=?", id, owner); err != nil {
			t.Error("本轮迟到计量任务清理失败")
		}
	})
	c := cache.CacheConf{{RedisConf: redis.RedisConf{Host: address, Type: "node"}, Weight: 100}}
	m := NewAiTaskModel(sqlx.NewSqlConnFromDB(db), c)
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := m.(*customAiTaskModel).DelCacheCtx(cleanup, fmt.Sprintf("%s%d", cacheGozeroAdminAiTaskIdPrefix, id)); err != nil {
			t.Error("本轮迟到计量缓存清理失败")
		}
	})
	usage := WorkflowStepUsage{Scope: "chat", PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15}
	if _, err := m.StoreTerminalStepUsage(ctx, id, owner, expected, usage); !errors.Is(err, ErrTaskState) {
		t.Fatalf("运行中步骤不能通过终态方法补写: %v", err)
	}
	if err := m.CancelOwned(ctx, id, owner); err != nil {
		t.Fatal(err)
	}
	before, err := m.FindOwned(ctx, id, owner)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.FindOne(ctx, uint64(id)); err != nil {
		t.Fatal("取消状态缓存预热失败")
	}
	status, err := m.StoreTerminalStepUsage(ctx, id, owner, expected, usage)
	if err != nil || status != "cancelled" {
		t.Fatalf("计量补写须保留cancelled终态: status=%s err=%v", status, err)
	}
	after, err := m.FindOwned(ctx, id, owner)
	if err != nil {
		t.Fatal(err)
	}
	records, _ := ReadWorkflowStepRecords(after.StepRecords)
	if after.Status != before.Status || after.NextStep != before.NextStep || after.Input != before.Input || after.Result != before.Result || after.Error != before.Error || after.CompletedAt != before.CompletedAt || len(records) != 1 || records[0].Status != "cancelled" || records[0].Input != expected.Input || records[0].Output != "" || records[0].Usage == nil || *records[0].Usage != usage {
		t.Fatal("取消后只可补原步骤计量，不能恢复状态或答案")
	}
	cached, err := m.FindOne(ctx, uint64(id))
	if err != nil || cached.StepRecords.String != after.StepRecords {
		t.Fatal("补写已提交后必须失效原ID缓存")
	}
	if _, err := m.StoreTerminalStepUsage(ctx, id, owner, expected, usage); err != nil {
		t.Fatal("相同迟到计量必须幂等")
	}
	for _, mutate := range []func(*WorkflowStepRecord){
		func(r *WorkflowStepRecord) { r.Index++ },
		func(r *WorkflowStepRecord) { r.StartedAt++ },
		func(r *WorkflowStepRecord) { r.Id = "other" },
		func(r *WorkflowStepRecord) { r.Tool = "other" },
	} {
		wrong := expected
		mutate(&wrong)
		if _, err := m.StoreTerminalStepUsage(ctx, id, owner, wrong, usage); !errors.Is(err, ErrTaskState) {
			t.Fatalf("其他执行身份不得补写原计量: %v", err)
		}
	}
	if _, err := m.StoreTerminalStepUsage(ctx, id, owner+1, expected, usage); !errors.Is(err, ErrNotFound) {
		t.Fatalf("其他用户不得补写: %v", err)
	}
	changed := usage
	changed.TotalTokens = 20
	if _, err := m.StoreTerminalStepUsage(ctx, id, owner, expected, changed); !errors.Is(err, ErrTaskState) {
		t.Fatalf("已有计量不得被不同载荷覆盖: %v", err)
	}
}
