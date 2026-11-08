package aiservicelogic

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
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
)

func TestCustomEvaluationActualCorpusOwnerAndFrozenSQLReceipt(t *testing.T) {
	dsn, address := os.Getenv("GOZERO_AI_TEST_DSN"), os.Getenv("GOZERO_AI_TEST_REDIS")
	if dsn == "" || address == "" {
		t.Skip("需要本轮隔离MySQL/Redis")
	}
	sqlx.DisableLog()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	owner := time.Now().UnixNano()
	ids := []int64{}
	var evaluationID int64
	requestID := uuid.NewString()
	rdb, err := redis.NewRedis(redis.RedisConf{Host: address, Type: "node"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if evaluationID > 0 {
			if _, err := db.ExecContext(cleanup, "DELETE FROM ai_kb_evaluation WHERE id=? AND owner_id=? AND request_id=?", evaluationID, owner, requestID); err != nil {
				t.Error("本轮自定义评测记录清理失败")
			}
			if _, err := rdb.DelCtx(cleanup, fmt.Sprintf("cache:gozeroAdmin:aiKbEvaluation:id:%d", evaluationID), fmt.Sprintf("cache:gozeroAdmin:aiKbEvaluation:ownerId:requestId:%d:%s", owner, requestID)); err != nil {
				t.Error("本轮评测缓存清理失败")
			}
			var count int
			if err := db.QueryRowContext(cleanup, "SELECT COUNT(*) FROM ai_kb_evaluation WHERE id=? AND owner_id=? AND request_id=?", evaluationID, owner, requestID).Scan(&count); err != nil || count != 0 {
				t.Error("本轮自定义评测记录未清理归零")
			}
		}
		for _, kb := range ids {
			for _, query := range []string{"DELETE FROM ai_kb_doc WHERE kb_id=?", "DELETE FROM ai_kb WHERE id=?"} {
				if _, err := db.ExecContext(cleanup, query, kb); err != nil {
					t.Error("本轮题集文档/知识库清理失败")
				}
			}
			for _, query := range []string{"SELECT COUNT(*) FROM ai_kb_doc WHERE kb_id=?", "SELECT COUNT(*) FROM ai_kb WHERE id=?"} {
				var count int
				if err := db.QueryRowContext(cleanup, query, kb).Scan(&count); err != nil || count != 0 {
					t.Error("本轮题集夹具未清理归零")
				}
			}
		}
	})
	create := func(query string, args ...any) int64 {
		t.Helper()
		result, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			t.Fatal(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	for _, who := range []int64{owner, owner + 1} {
		ids = append(ids, create("INSERT INTO ai_kb(owner_id,name) VALUES (?,'自定义冻结题集隔离回归')", who))
	}
	ownDoc := create("INSERT INTO ai_kb_doc(kb_id,title,content,status,version,embedding_model) VALUES (?,'项目说明','预算五万元。保存30天。','ready',2,'local-fixture')", ids[0])
	foreignDoc := create("INSERT INTO ai_kb_doc(kb_id,title,content,status,version,embedding_model) VALUES (?,'项目说明','预算五万元。保存30天。','ready',2,'local-fixture')", ids[1])
	conf := cache.CacheConf{{RedisConf: redis.RedisConf{Host: address, Type: "node"}, Weight: 100}}
	store := model.NewAiKbEvaluationModel(sqlx.NewSqlConnFromDB(db), conf)
	corpus, err := store.ReadCorpusOwned(ctx, owner, ids[0])
	if err != nil || len(corpus) != 1 || corpus[0].Id != ownDoc {
		t.Fatal("事实源只能读取本人当前知识库")
	}
	raw := customEvaluationFixture(t)
	wrong := strings.Replace(raw, `"source_doc_id":10`, fmt.Sprintf(`"source_doc_id":%d`, foreignDoc), 1)
	key, sha := customEvaluationIdentity(wrong)
	dataset, err := parseCustomEvaluationDataset(key, sha, wrong)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildEvaluationBindings(dataset, "project", ids[0], corpus); err == nil {
		t.Fatal("即使同标题/SHA/版本，跨owner文档ID也不能回退绑定本人同名文档")
	}
	raw = strings.Replace(raw, `"source_doc_id":10`, fmt.Sprintf(`"source_doc_id":%d`, ownDoc), 1)
	key, sha = customEvaluationIdentity(raw)
	dataset, err = parseCustomEvaluationDataset(key, sha, raw)
	if err != nil {
		t.Fatal(err)
	}
	bindings, err := buildEvaluationBindings(dataset, "project", ids[0], corpus)
	if err != nil {
		t.Fatal(err)
	}
	request := &aipb.StartKbEvaluationRequest{KbId: ids[0], RequestId: requestID, DatasetKey: key, DatasetSha256: sha, DatasetSnapshot: raw, GroupKey: "project", CaseIds: []string{"custom-facts"}}
	signature, err := evaluationRequestSignature(request)
	if err != nil {
		t.Fatal(err)
	}
	observations, _ := json.Marshal([]model.EvaluationObservation{{CaseId: "custom-facts", RequestId: uuid.NewString(), Query: "预算和保存期是什么？", Status: "waiting", Stage: "waiting", List: json.RawMessage(`[]`)}})
	draft := &model.KbEvaluationRow{OwnerId: owner, KbId: ids[0], RequestId: requestID, RequestSignature: signature, DatasetKey: key, DatasetSHA: sha, GroupKey: "project", CaseCount: 1, Snapshot: raw, Bindings: bindings, Context: "{}", Observations: string(observations)}
	created, fresh, err := store.CreateOwned(ctx, draft)
	if created != nil {
		evaluationID = created.Id
	}
	if err != nil || !fresh {
		t.Fatalf("冻结记录未保存: %v", err)
	}
	read, err := store.FindRequestOwned(ctx, owner, requestID)
	if err != nil || read.Snapshot != raw || strings.Contains(read.Bindings, `"content":`) {
		t.Fatal("SQL回读须保留完整原始snapshot，不把全文放入bindings")
	}
	if receipt, fresh, err := store.CreateOwned(ctx, draft); err != nil || fresh || receipt.Id != evaluationID {
		t.Fatal("相同快照UUID必须复用SQL批次")
	}
	changed := *draft
	request.DatasetSnapshot += "\n"
	changed.Snapshot = request.DatasetSnapshot
	changed.RequestSignature, err = evaluationRequestSignature(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.CreateOwned(ctx, &changed); !errors.Is(err, model.ErrEvaluationConflict) {
		t.Fatal("相同UUID修改原字节必须SQL幂等冲突")
	}
	t.Logf("本轮custom fixture kb_ids=%v doc_ids=%v evaluation_id=%d；均按精确ID清理，无模型或向量调用", ids, []int64{ownDoc, foreignDoc}, evaluationID)
}
