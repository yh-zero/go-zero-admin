package model

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type evaluationStoreHarness struct {
	ctx       context.Context
	db        *sql.DB
	conn      sqlx.SqlConn
	store     AiKbEvaluationModel
	owner, kb int64
}

func newEvaluationStoreHarness(t *testing.T) *evaluationStoreHarness {
	t.Helper()
	dsn, address := os.Getenv("GOZERO_AI_TEST_DSN"), os.Getenv("GOZERO_AI_TEST_REDIS")
	if dsn == "" || address == "" {
		t.Skip("requires isolated GOZERO_AI_TEST_DSN / GOZERO_AI_TEST_REDIS")
	}
	sqlx.DisableLog()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err = db.PingContext(ctx); err != nil {
		t.Fatal("隔离MySQL不可达")
	}
	conf := cache.CacheConf{{RedisConf: redis.RedisConf{Host: address, Type: "node"}, Weight: 100}}
	rdb, err := redis.NewRedis(conf[0].RedisConf)
	if err != nil || !rdb.PingCtx(ctx) {
		t.Fatal("隔离Redis不可达")
	}
	conn := sqlx.NewSqlConnFromDB(db)
	h := &evaluationStoreHarness{ctx: ctx, db: db, conn: conn, store: NewAiKbEvaluationModel(conn, conf), owner: time.Now().UnixNano()}
	result, err := db.ExecContext(ctx, "INSERT INTO ai_kb(owner_id,name,description) VALUES (?,?,'')", h.owner, "质量验收隔离-"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	h.kb, _ = result.LastInsertId()
	t.Cleanup(func() {
		clean, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		var rows []struct {
			Id        int64  `db:"id"`
			RequestId string `db:"request_id"`
		}
		if err := conn.QueryRowsCtx(clean, &rows, "SELECT id,request_id FROM ai_kb_evaluation WHERE owner_id=? AND kb_id=?", h.owner, h.kb); err != nil {
			t.Error("精确清理清单读取失败")
			return
		}
		for _, row := range rows {
			if _, err := db.ExecContext(clean, "DELETE FROM ai_kb_evaluation WHERE id=? AND owner_id=? AND kb_id=?", row.Id, h.owner, h.kb); err != nil {
				t.Error("清理本轮评测失败")
			}
			for _, key := range []string{fmt.Sprintf("cache:gozeroAdmin:aiKbEvaluation:id:%d", row.Id), fmt.Sprintf("cache:gozeroAdmin:aiKbEvaluation:ownerId:requestId:%d:%s", h.owner, row.RequestId)} {
				if _, err := rdb.DelCtx(clean, key); err != nil {
					t.Error("清理本轮缓存失败")
				}
			}
		}
		if _, err := db.ExecContext(clean, "DELETE FROM ai_kb_doc WHERE kb_id=?", h.kb); err != nil {
			t.Error("清理本轮文档失败")
		}
		if _, err := db.ExecContext(clean, "DELETE FROM ai_kb WHERE id=? AND owner_id=?", h.kb, h.owner); err != nil {
			t.Error("清理本轮知识库失败")
		}
	})
	return h
}
func (h *evaluationStoreHarness) draft(t *testing.T) *KbEvaluationRow {
	rows, raw := evaluationStateFixture(t)
	for i := range rows {
		rows[i].RequestId = uuid.NewString()
	}
	encoded, _ := json.Marshal(rows)
	_ = raw
	return &KbEvaluationRow{OwnerId: h.owner, KbId: h.kb, RequestId: uuid.NewString(), RequestSignature: strings.Repeat("a", 64), DatasetKey: "fixture", DatasetSHA: strings.Repeat("b", 64), GroupKey: "ops", CaseCount: 2, Snapshot: `{"version":1}`, Bindings: `{"documents":[],"inventory":[]}`, Context: `{}`, Observations: string(encoded)}
}
func TestEvaluationStoreIdempotencyStopAndLateWrites(t *testing.T) {
	h := newEvaluationStoreHarness(t)
	draft := h.draft(t)
	row, created, err := h.store.CreateOwned(h.ctx, draft)
	if err != nil || !created || row.Status != "pending" {
		t.Fatalf("创建失败: %v", err)
	}
	replay, newRow, err := h.store.CreateOwned(h.ctx, draft)
	if err != nil || newRow || replay.Id != row.Id {
		t.Fatal("相同UUID不能再创建")
	}
	changed := *draft
	changed.RequestSignature = strings.Repeat("c", 64)
	if _, _, err = h.store.CreateOwned(h.ctx, &changed); !errors.Is(err, ErrEvaluationConflict) {
		t.Fatal("相同UUID不同内容应冲突")
	}
	if _, err = h.store.FindOwned(h.ctx, h.owner+1, h.kb, row.Id); !errors.Is(err, ErrNotFound) {
		t.Fatal("跨用户不能读取")
	}
	if _, err = h.store.FindOwned(h.ctx, h.owner, h.kb+1, row.Id); !errors.Is(err, ErrNotFound) {
		t.Fatal("错知识库不能读取")
	}
	if _, err = h.store.FindOne(h.ctx, uint64(row.Id)); err != nil {
		t.Fatal(err)
	}
	token := uuid.NewString()
	row, err = h.store.ActivateOwned(h.ctx, h.owner, h.kb, row.Id, token)
	if err != nil {
		t.Fatal(err)
	}
	cached, err := h.store.FindOneByOwnerIdRequestId(h.ctx, uint64(h.owner), draft.RequestId)
	if err != nil || cached.Status != "running" {
		t.Fatal("激活后缓存应保持一致")
	}
	row, err = h.store.TryStartCase(h.ctx, h.owner, h.kb, row.Id, 0, token)
	if err != nil {
		t.Fatal(err)
	}
	row, err = h.store.StopOwned(h.ctx, h.owner, h.kb, row.Id)
	if err != nil || !row.StopRequested || row.Status != "running" {
		t.Fatal("停止只挡后续，当前项可继续")
	}
	if _, err = h.store.TryStartCase(h.ctx, h.owner, h.kb, row.Id, 1, token); !errors.Is(err, ErrEvaluationStopped) {
		t.Fatal("停止后不能领取后续题")
	}
	items, _ := decodeEvaluationObservations(row.Observations)
	next := items[0]
	next.TaskId = 123
	next.Stage = "polling"
	row, err = h.store.SaveCase(h.ctx, h.owner, h.kb, row.Id, 0, token, next)
	if err != nil {
		t.Fatal(err)
	}
	next.Status = "succeeded"
	next.Stage = "completed"
	next.Analysis = json.RawMessage(`{"version":1,"type":"kb_analysis"}`)
	row, err = h.store.SaveCase(h.ctx, h.owner, h.kb, row.Id, 0, token, next)
	if err != nil || row.CompletedCount != 1 {
		t.Fatal("已领取项应可写完成")
	}
	row, err = h.store.FinishOwned(h.ctx, h.owner, h.kb, row.Id, token, "cancelled", false, "已停止")
	if err != nil || row.Status != "cancelled" || row.CompletedCount != 1 || row.FailedCount != 0 {
		t.Fatal("停止计数应保留")
	}
	if _, err = h.store.SaveCase(h.ctx, h.owner, h.kb, row.Id, 0, token, next); !errors.Is(err, ErrEvaluationState) {
		t.Fatal("迟到写不能恢复终态")
	}
	if err = h.store.HeartbeatOwned(h.ctx, h.owner, h.kb, row.Id, token); !errors.Is(err, ErrEvaluationState) {
		t.Fatal("终态不能续租")
	}
	list, total, err := h.store.ListOwned(h.ctx, h.owner, h.kb, 1, 10, "all", draft.RequestId)
	if err != nil || total != 1 || len(list) != 1 || list[0].Snapshot != "" || list[0].Observations != "" {
		t.Fatal("列表须有界并支持原UUID核对")
	}
}
func TestEvaluationStoreExpiredLeaseCannotResume(t *testing.T) {
	h := newEvaluationStoreHarness(t)
	row, _, err := h.store.CreateOwned(h.ctx, h.draft(t))
	if err != nil {
		t.Fatal(err)
	}
	token := uuid.NewString()
	row, err = h.store.ActivateOwned(h.ctx, h.owner, h.kb, row.Id, token)
	if err != nil {
		t.Fatal(err)
	}
	row, err = h.store.TryStartCase(h.ctx, h.owner, h.kb, row.Id, 0, token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.db.ExecContext(h.ctx, "UPDATE ai_kb_evaluation SET lease_expires_at=DATE_SUB(NOW(),INTERVAL 1 SECOND) WHERE id=? AND owner_id=?", row.Id, h.owner); err != nil {
		t.Fatal(err)
	}
	if err = h.store.HeartbeatOwned(h.ctx, h.owner, h.kb, row.Id, token); !errors.Is(err, ErrEvaluationState) {
		t.Fatal("过期执行者不能自行续租")
	}
	row, err = h.store.ExpireOwned(h.ctx, h.owner, h.kb, row.Id)
	if err != nil || row.Status != "unknown" || row.FailedCount != 1 {
		t.Fatalf("过期应标未知: %+v %v", row, err)
	}
	if _, err = h.store.ActivateOwned(h.ctx, h.owner, h.kb, row.Id, uuid.NewString()); !errors.Is(err, ErrEvaluationState) {
		t.Fatal("未知不能自动重领")
	}
	items, _ := decodeEvaluationObservations(row.Observations)
	if items[0].Status != "unknown" || items[1].Status != "waiting" {
		t.Fatal("过期仅标当前项，后续仍未执行")
	}
}
func TestEvaluationStoreCommittedCacheError(t *testing.T) {
	h := newEvaluationStoreHarness(t)
	broken := &customAiKbEvaluationModel{conn: h.conn, defaultAiKbEvaluationModel: &defaultAiKbEvaluationModel{CachedConn: sqlc.NewConnWithCache(h.conn, &failedKbCache{})}}
	draft := h.draft(t)
	row, created, err := broken.CreateOwned(h.ctx, draft)
	if !created || row == nil || !errors.Is(err, ErrEvaluationCommitted) {
		t.Fatal("已提交与缓存失败必须区分")
	}
	replay, newRow, err := broken.CreateOwned(h.ctx, draft)
	if err != nil || newRow || replay.Id != row.Id {
		t.Fatal("缓存故障后的相同请求不能重建")
	}
	token := uuid.NewString()
	_, err = broken.ActivateOwned(h.ctx, h.owner, h.kb, row.Id, token)
	if !errors.Is(err, ErrEvaluationCommitted) {
		t.Fatal("激活提交后的缓存故障应保留")
	}
	actual, err := h.store.FindOwned(h.ctx, h.owner, h.kb, row.Id)
	if err != nil || actual.Status != "running" || actual.LeaseToken != token {
		t.Fatal("实际已提交的执行权不能抹掉")
	}
}

func TestEvaluationStoreLeaseExpiryDuringWriteRejectsLateCommit(t *testing.T) {
	h := newEvaluationStoreHarness(t)
	row, _, err := h.store.CreateOwned(h.ctx, h.draft(t))
	if err != nil {
		t.Fatal(err)
	}
	token := uuid.NewString()
	row, err = h.store.ActivateOwned(h.ctx, h.owner, h.kb, row.Id, token)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = h.db.ExecContext(h.ctx, "UPDATE ai_kb_evaluation SET lease_expires_at=DATE_ADD(NOW(),INTERVAL 1 SECOND) WHERE id=? AND owner_id=?", row.Id, h.owner); err != nil {
		t.Fatal(err)
	}
	custom := h.store.(*customAiKbEvaluationModel)
	beganValid := false
	_, err = custom.mutateEvaluation(h.ctx, h.owner, h.kb, row.Id, token, func(current *KbEvaluationRow) (bool, error) {
		if !evaluationLeaseValid(current, token) {
			return false, ErrEvaluationState
		}
		beganValid = true
		current.Error = "迟到进度不应提交"
		time.Sleep(2100 * time.Millisecond)
		return true, nil
	})
	if !beganValid {
		t.Fatal("回归前提未满足：事务必须在租约有效时进入，再跨过过期点")
	}
	if !errors.Is(err, ErrEvaluationState) {
		t.Fatalf("租约在事务内过期也必须拒绝迟到提交: %v", err)
	}
	actual, err := h.store.FindOwned(h.ctx, h.owner, h.kb, row.Id)
	if err != nil || actual.Error != "" {
		t.Fatal("迟到事务应回滚，不能覆盖已有状态")
	}
}
