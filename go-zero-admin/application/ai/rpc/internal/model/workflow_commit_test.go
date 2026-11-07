package model

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// 默认Redis缓存会异步补偿；此用例覆盖真正返回错误的自定义cache边界。
func TestWorkflowCommittedCacheErrorPreservesData(t *testing.T) {
	dsn := os.Getenv("GOZERO_AI_TEST_DSN")
	if dsn == "" {
		t.Skip("requires isolated GOZERO_AI_TEST_DSN")
	}
	sqlx.DisableLog()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	conn := sqlx.NewSqlConnFromDB(db)
	cached := sqlc.NewConnWithCache(conn, &failedKbCache{})
	w := &customAiWorkflowModel{conn: conn, defaultAiWorkflowModel: &defaultAiWorkflowModel{CachedConn: cached}}
	tasks := &customAiTaskModel{conn: conn, defaultAiTaskModel: &defaultAiTaskModel{CachedConn: cached}}
	owner := time.Now().UnixNano()
	result, err := db.ExecContext(ctx, "INSERT INTO ai_workflow(name,definition,created_by) VALUES ('提交边界回归',?,?)", `{"steps":[{"name":"占位","tool":"get_current_time","input":{}}]}`, owner)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := result.LastInsertId()
	t.Cleanup(func() {
		clean, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, table := range []string{"ai_task", "ai_workflow"} {
			if _, err := db.ExecContext(clean, "DELETE FROM "+table+" WHERE created_by=?", owner); err != nil {
				t.Error("清理本轮隔离提交边界数据失败")
			}
		}
	})
	request := uuid.NewString()
	task, created, err := w.CreateTaskIdempotent(ctx, id, owner, `{}`, request)
	if !errors.Is(err, ErrTaskCommitted) || !created || task == nil {
		t.Fatalf("缓存故障须保留已提交任务: %+v %v %v", task, created, err)
	}
	if replay, newTask, err := w.CreateTaskIdempotent(ctx, id, owner, `{}`, request); err != nil || newTask || replay.Id != task.Id {
		t.Fatal("已提交任务须按request_id复用")
	}
	records := []WorkflowStepRecord{{Index: 0, Id: "one", Name: "待批准", Tool: "get_current_time", Input: `{}`, Status: "pending_approval"}}
	if err := tasks.AdvanceRecorded(ctx, task.Id, owner, 0, 0, "pending_approval", "", "", records); !errors.Is(err, ErrTaskCommitted) {
		t.Fatal("进度提交与缓存故障须区分")
	}
	if err := tasks.ClaimApproval(ctx, task.Id, owner, 0); !errors.Is(err, ErrTaskCommitted) {
		t.Fatal("批准已领取执行权必须明确返回已提交错误")
	}
	records[0].Status = "succeeded"
	records[0].Output = "已完成"
	if err := tasks.AdvanceRecorded(ctx, task.Id, owner, 0, 1, "succeeded", "已完成", "", records); !errors.Is(err, ErrTaskCommitted) {
		t.Fatal(err)
	}
	if row, err := tasks.FindOwned(ctx, task.Id, owner); err != nil || row.Status != "succeeded" || row.NextStep != 1 {
		t.Fatal("缓存故障不得回滚或抹去已提交进度")
	}
	other, _, err := w.CreateTaskIdempotent(ctx, id, owner, `{}`, uuid.NewString())
	if !errors.Is(err, ErrTaskCommitted) {
		t.Fatal(err)
	}
	if err := tasks.CancelOwned(ctx, other.Id, owner); !errors.Is(err, ErrTaskCommitted) {
		t.Fatal(err)
	}
	if row, err := tasks.FindOwned(ctx, other.Id, owner); err != nil || row.Status != "cancelled" {
		t.Fatal("取消事务已经提交，不能报告回滚")
	}
}
