package model

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type workflowRevisionCache struct {
	cache.Cache
	fail bool
}

func (c *workflowRevisionCache) DelCtx(context.Context, ...string) error {
	if c.fail {
		return errors.New("isolated workflow cache failure")
	}
	return nil
}

type workflowRevisionFixture struct {
	ctx       context.Context
	db        *sql.DB
	conn      sqlx.SqlConn
	model     *customAiWorkflowModel
	cache     *workflowRevisionCache
	id, owner int64
}

func newWorkflowRevisionFixture(t *testing.T) *workflowRevisionFixture {
	t.Helper()
	dsn := os.Getenv("GOZERO_AI_TEST_DSN")
	if dsn == "" {
		t.Skip("requires isolated GOZERO_AI_TEST_DSN")
	}
	sqlx.DisableLog()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("隔离工作流版本数据库初始化失败")
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	f := &workflowRevisionFixture{ctx: ctx, db: db, conn: sqlx.NewSqlConnFromDB(db), cache: &workflowRevisionCache{}, owner: time.Now().UnixNano()}
	f.model = &customAiWorkflowModel{conn: f.conn, defaultAiWorkflowModel: &defaultAiWorkflowModel{CachedConn: sqlc.NewConnWithCache(f.conn, f.cache)}}
	result, err := db.ExecContext(ctx, "INSERT INTO ai_workflow(name,definition,created_by) VALUES ('初始版本',?,?)", `{"steps":[{"name":"时间","tool":"get_current_time","input":{}}]}`, f.owner)
	if err != nil {
		t.Fatal("隔离工作流版本数据创建失败")
	}
	f.id, _ = result.LastInsertId()
	t.Cleanup(func() {
		clean, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		for _, item := range []struct {
			query string
			args  []any
		}{
			{"DELETE FROM ai_task WHERE workflow_id=? AND created_by=?", []any{f.id, f.owner}},
			{"DELETE FROM ai_workflow_revision WHERE workflow_id=? AND owner_id=?", []any{f.id, f.owner}},
			{"DELETE FROM ai_workflow WHERE id=? AND created_by=?", []any{f.id, f.owner}},
		} {
			if _, err := db.ExecContext(clean, item.query, item.args...); err != nil {
				t.Error("本轮工作流版本数据精确清理失败")
			}
		}
		var remaining int
		if err := db.QueryRowContext(clean, "SELECT (SELECT COUNT(*) FROM ai_workflow WHERE id=? AND created_by=?)+(SELECT COUNT(*) FROM ai_workflow_revision WHERE workflow_id=? AND owner_id=?)", f.id, f.owner, f.id, f.owner).Scan(&remaining); err != nil || remaining != 0 {
			t.Error("工作流版本数据清理后仍存在或不可核对")
		}
	})
	return f
}

func TestWorkflowRevisionConcurrentUpdateAndRestore(t *testing.T) {
	f := newWorkflowRevisionFixture(t)
	initial, err := f.model.FindOwned(f.ctx, f.id, f.owner)
	if err != nil || initial.RevisionSeq != 1 {
		t.Fatalf("初始版本必须为1: %v", err)
	}
	task, _, err := f.model.CreateTaskIdempotent(f.ctx, f.id, f.owner, `{}`, "11111111-1111-4111-8111-111111111111")
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var waiting sync.WaitGroup
	for _, name := range []string{"修改甲", "修改乙"} {
		waiting.Add(1)
		go func(name string) {
			defer waiting.Done()
			<-start
			_, err := f.model.UpdateOwned(f.ctx, f.id, f.owner, 1, WorkflowPatch{Name: &name})
			results <- err
		}(name)
	}
	close(start)
	waiting.Wait()
	close(results)
	succeeded, conflicts := 0, 0
	for err := range results {
		if err == nil {
			succeeded++
		} else if errors.Is(err, ErrWorkflowRevisionConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if succeeded != 1 || conflicts != 1 {
		t.Fatalf("同基线并发只能成功一次: success=%d conflict=%d", succeeded, conflicts)
	}
	current, err := f.model.FindOwned(f.ctx, f.id, f.owner)
	if err != nil || current.RevisionSeq != 2 {
		t.Fatalf("当前版本必须为2: %v", err)
	}
	winner := current.Name
	history, total, err := f.model.ListRevisionsOwned(f.ctx, f.id, f.owner, 1, 20)
	if err != nil || total != 1 || len(history) != 1 || history[0].RevisionSeq != 1 || history[0].Name != "初始版本" || history[0].Definition != "" {
		t.Fatalf("历史须归档旧摘要且不读取正文: total=%d rows=%+v err=%v", total, history, err)
	}
	old, err := f.model.FindRevisionOwned(f.ctx, f.id, f.owner, 1)
	if err != nil || !strings.Contains(old.Definition, "get_current_time") {
		t.Fatal("历史详情须保留完整旧定义")
	}
	sameDefinition := `{"steps":[{"name":"时间","tool":"get_current_time","input":{}}]}`
	unchanged, err := f.model.UpdateOwned(f.ctx, f.id, f.owner, 2, WorkflowPatch{Name: &winner, Definition: &sameDefinition})
	if err != nil || unchanged.RevisionSeq != 2 {
		t.Fatalf("同值patch不得增加版本: %v", err)
	}
	if _, err := f.model.UpdateOwned(f.ctx, f.id, f.owner, 1, WorkflowPatch{}); !errors.Is(err, ErrWorkflowRevisionConflict) {
		t.Fatal("过期空patch也必须冲突")
	}
	restored, err := f.model.RestoreRevisionOwned(f.ctx, f.id, f.owner, 1, 2)
	if err != nil || restored.RevisionSeq != 3 || restored.Name != "初始版本" {
		t.Fatalf("恢复必须成为新版本3: %+v %v", restored, err)
	}
	if _, err := f.model.UpdateOwned(f.ctx, f.id, f.owner, 1, WorkflowPatch{Name: &winner}); !errors.Is(err, ErrWorkflowRevisionConflict) {
		t.Fatal("A到B再恢复A后旧请求不能通过")
	}
	if _, err := f.model.RestoreRevisionOwned(f.ctx, f.id, f.owner, 1, 2); !errors.Is(err, ErrWorkflowRevisionConflict) {
		t.Fatal("迟到恢复不能覆盖新版本")
	}
	history, total, err = f.model.ListRevisionsOwned(f.ctx, f.id, f.owner, 1, 1)
	if err != nil || total != 2 || len(history) != 1 || history[0].RevisionSeq != 2 || history[0].Name != winner {
		t.Fatal("恢复前版本须归档，分页按最新在前")
	}
	oldAgain, err := f.model.FindRevisionOwned(f.ctx, f.id, f.owner, 1)
	if err != nil || oldAgain.Name != old.Name || oldAgain.Definition != old.Definition {
		t.Fatal("恢复不能改写既有历史")
	}
	var snapshot string
	if err := f.db.QueryRowContext(f.ctx, "SELECT CAST(workflow_snapshot AS CHAR) FROM ai_task WHERE id=? AND created_by=?", task.Id, f.owner).Scan(&snapshot); err != nil || snapshot != initial.Definition {
		t.Fatal("修改及恢复不能覆盖原任务快照")
	}
	sameContent, err := f.model.RestoreRevisionOwned(f.ctx, f.id, f.owner, 1, 3)
	if err != nil || sameContent.RevisionSeq != 4 {
		t.Fatal("显式恢复相同内容仍须归档并创建新版本")
	}
}

func TestWorkflowRevisionOwnershipAndSoftDelete(t *testing.T) {
	f := newWorkflowRevisionFixture(t)
	name := "下一版本"
	if _, err := f.model.UpdateOwned(f.ctx, f.id, f.owner, 1, WorkflowPatch{Name: &name}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.model.FindRevisionOwned(f.ctx, f.id, f.owner+1, 1); !errors.Is(err, ErrNotFound) {
		t.Fatal("跨owner历史读取须404")
	}
	if _, _, err := f.model.ListRevisionsOwned(f.ctx, f.id, f.owner+1, 1, 20); !errors.Is(err, ErrNotFound) {
		t.Fatal("跨owner历史列表须404")
	}
	if _, err := f.model.RestoreRevisionOwned(f.ctx, f.id, f.owner+1, 1, 2); !errors.Is(err, ErrNotFound) {
		t.Fatal("跨owner恢复须404")
	}
	if _, err := f.model.FindRevisionOwned(f.ctx, f.id, f.owner, 99); !errors.Is(err, ErrNotFound) {
		t.Fatal("不存在的历史须404")
	}
	if err := f.model.DeleteOwned(f.ctx, f.id, f.owner); err != nil {
		t.Fatal(err)
	}
	if _, err := f.model.FindRevisionOwned(f.ctx, f.id, f.owner, 1); !errors.Is(err, ErrNotFound) {
		t.Fatal("软删后历史不能继续读取")
	}
	if _, _, err := f.model.ListRevisionsOwned(f.ctx, f.id, f.owner, 1, 20); !errors.Is(err, ErrNotFound) {
		t.Fatal("软删后历史列表须404")
	}
	if _, err := f.model.RestoreRevisionOwned(f.ctx, f.id, f.owner, 1, 2); !errors.Is(err, ErrNotFound) {
		t.Fatal("历史恢复不能复活软删工作流")
	}
}

func TestWorkflowRevisionOmittedSchedulePreservesLegacyMetadata(t *testing.T) {
	f := newWorkflowRevisionFixture(t)
	if _, err := f.db.ExecContext(f.ctx, "UPDATE ai_workflow SET interval_seconds=60,next_run_at=DATE_ADD(NOW(),INTERVAL 1 HOUR) WHERE id=? AND created_by=?", f.id, f.owner); err != nil {
		t.Fatal(err)
	}
	before, err := f.model.FindOwned(f.ctx, f.id, f.owner)
	if err != nil || before.NextRunAt == 0 {
		t.Fatal("历史时间夹具未建立")
	}
	name := "仅改名称"
	after, err := f.model.UpdateOwned(f.ctx, f.id, f.owner, 1, WorkflowPatch{Name: &name})
	if err != nil || after.IntervalSeconds != 60 || after.NextRunAt != before.NextRunAt {
		t.Fatal("省略调度字段时不能改变历史元数据")
	}
}

type workflowRevisionFailConn struct{ sqlx.SqlConn }
type workflowRevisionFailSession struct{ sqlx.Session }

func (c workflowRevisionFailConn) TransactCtx(ctx context.Context, action func(context.Context, sqlx.Session) error) error {
	return c.SqlConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return action(ctx, workflowRevisionFailSession{session})
	})
}
func (s workflowRevisionFailSession) ExecCtx(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if strings.HasPrefix(query, "UPDATE ai_workflow SET") {
		return nil, errors.New("isolated workflow update failure")
	}
	return s.Session.ExecCtx(ctx, query, args...)
}

func TestWorkflowRevisionRollbackAndCommittedCacheError(t *testing.T) {
	f := newWorkflowRevisionFixture(t)
	name := "不能部分提交"
	f.model.conn = workflowRevisionFailConn{f.conn}
	if _, err := f.model.UpdateOwned(f.ctx, f.id, f.owner, 1, WorkflowPatch{Name: &name}); err == nil {
		t.Fatal("数据库故障必须失败")
	}
	current, err := f.model.FindOwned(f.ctx, f.id, f.owner)
	if err != nil || current.Name != "初始版本" || current.RevisionSeq != 1 {
		t.Fatal("失败更新必须回滚当前版本")
	}
	_, total, err := f.model.ListRevisionsOwned(f.ctx, f.id, f.owner, 1, 20)
	if err != nil || total != 0 {
		t.Fatal("失败更新必须同时回滚已插入历史")
	}
	f.model.conn = f.conn
	f.cache.fail = true
	committed, err := f.model.UpdateOwned(f.ctx, f.id, f.owner, 1, WorkflowPatch{Name: &name})
	if !errors.Is(err, ErrWorkflowCommitted) || committed == nil || committed.RevisionSeq != 2 {
		t.Fatal("提交后缓存故障不能冒充回滚")
	}
	current, err = f.model.FindOwned(f.ctx, f.id, f.owner)
	if err != nil || current.Name != name || current.RevisionSeq != 2 {
		t.Fatal("缓存故障不能抹去已提交版本")
	}
	_, total, err = f.model.ListRevisionsOwned(f.ctx, f.id, f.owner, 1, 20)
	if err != nil || total != 1 {
		t.Fatal("缓存故障后历史也必须已提交")
	}
	f.cache.fail = false
	f.model.conn = workflowRevisionFailConn{f.conn}
	if _, err := f.model.RestoreRevisionOwned(f.ctx, f.id, f.owner, 1, 2); err == nil {
		t.Fatal("恢复更新故障必须失败")
	}
	current, err = f.model.FindOwned(f.ctx, f.id, f.owner)
	if err != nil || current.Name != name || current.RevisionSeq != 2 {
		t.Fatal("失败恢复必须回滚当前版本")
	}
	_, total, err = f.model.ListRevisionsOwned(f.ctx, f.id, f.owner, 1, 20)
	if err != nil || total != 1 {
		t.Fatal("失败恢复不能留下半条历史")
	}
	f.model.conn = f.conn
	f.cache.fail = true
	committed, err = f.model.RestoreRevisionOwned(f.ctx, f.id, f.owner, 1, 2)
	if !errors.Is(err, ErrWorkflowCommitted) || committed == nil || committed.RevisionSeq != 3 || committed.Name != "初始版本" {
		t.Fatal("恢复提交后缓存故障须保留新版本")
	}
}
