package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// 这些回归使用隔离库的真实事务与缓存；不设置环境变量时明确跳过。
func TestReserveCallReturnsPersistedCreationTimesAcrossSecondBoundary(t *testing.T) {
	h := newCallModelHarness(t)
	conversationId := h.createConversation("预留时间一致")
	delayed := false
	h.conn.onQuery = func(query string) {
		if !delayed && strings.HasPrefix(query, "INSERT INTO ai_message") {
			delayed = true
			// 预留时间已采样后跨过完整一秒，识别依赖数据库默认时间造成的偏差。
			time.Sleep(1100 * time.Millisecond)
		}
	}
	h.conn.reset()
	pair := h.reserve(conversationId)
	if !delayed {
		t.Fatal("reservation did not exercise the delayed message insert")
	}
	if queries := h.conn.snapshot(); len(queries) != 5 {
		t.Fatalf("reservation added reads to recover message timestamps: statements=%d", len(queries))
	}
	for _, message := range []*MessageRow{pair.User, pair.Assistant} {
		var createdAt int64
		if err := h.db.QueryRowContext(h.ctx, "SELECT UNIX_TIMESTAMP(created_at) FROM ai_message WHERE id=?", message.Id).Scan(&createdAt); err != nil {
			t.Fatal("read persisted reserved message timestamp failed")
		}
		if message.CreatedAt != createdAt {
			t.Errorf("%s reservation timestamp %d differs from persisted timestamp %d", message.Role, message.CreatedAt, createdAt)
		}
	}
}

func TestFinishCallReturnsCommittedSummaryWithoutReload(t *testing.T) {
	h := newCallModelHarness(t)
	conversationId := h.createConversation("摘要回填")
	for _, status := range []string{"succeeded", "failed", "running"} {
		h.exec("INSERT INTO ai_message (conversation_id,role,content,status) VALUES (?,'assistant','history',?)", conversationId, status)
	}
	pair := h.reserve(conversationId)
	for _, id := range []int64{pair.User.Id, pair.Assistant.Id} {
		if _, err := h.messages.FindOne(h.ctx, uint64(id)); err != nil {
			t.Fatal("prime message cache failed")
		}
	}
	h.conn.reset()
	before := time.Now().Unix()
	if err := h.messages.FinishCall(h.ctx, h.ownerId, pair, "succeeded", "saved answer", `{"usage_complete":true}`, 17); err != nil {
		t.Fatalf("FinishCall: %v", err)
	}
	after := time.Now().Unix()
	summary := pair.Conversation
	if summary == nil {
		t.Fatal("successful finish did not return the committed conversation summary")
	}
	if summary.Id != conversationId || summary.UserId != h.ownerId || summary.Title != "摘要回填" || summary.MessageCount != 5 {
		t.Fatalf("unexpected summary identity/count: %+v", summary)
	}
	if summary.CreatedAt != 1577836800 || summary.UpdatedAt < before || summary.UpdatedAt > after {
		t.Fatalf("unexpected summary timestamps: created=%d updated=%d finish=[%d,%d]", summary.CreatedAt, summary.UpdatedAt, before, after)
	}
	queries := h.conn.snapshot()
	if len(queries) != 5 || countQueries(queries, "COUNT(") != 1 || !strings.Contains(strings.ToUpper(queries[0]), "FOR UPDATE") {
		t.Fatalf("finish must reuse its parent lock for the summary, got %d SQL statements and %d counts", len(queries), countQueries(queries, "COUNT("))
	}
	var updatedAt int64
	if err := h.db.QueryRowContext(h.ctx, "SELECT UNIX_TIMESTAMP(updated_at) FROM ai_conversation WHERE id=?", conversationId).Scan(&updatedAt); err != nil {
		t.Fatal("read persisted conversation timestamp failed")
	}
	if updatedAt != summary.UpdatedAt {
		t.Fatalf("summary timestamp %d differs from committed timestamp %d", summary.UpdatedAt, updatedAt)
	}
	for _, id := range []int64{pair.User.Id, pair.Assistant.Id} {
		cached, err := h.messages.FindOne(h.ctx, uint64(id))
		if err != nil || cached.Status != "succeeded" {
			t.Fatal("successful finish left stale running messages in the real cache")
		}
	}
	h.conn.reset()
	replay, err := h.messages.FindCall(h.ctx, h.ownerId, pair.Assistant.CallId)
	if err != nil {
		t.Fatalf("FindCall: %v", err)
	}
	if replay.User.Status != "succeeded" || replay.Assistant.Status != "succeeded" || replay.Assistant.Content != "saved answer" || replay.Assistant.Tokens != 17 {
		t.Fatal("replay did not return both committed messages and saved answer/usage")
	}
	if queries := h.conn.snapshot(); len(queries) != 1 || countQueries(queries, "COUNT(") != 0 {
		t.Fatal("FindCall must retrieve the original pair in one history query")
	}
	if _, err := h.messages.FindCall(h.ctx, h.ownerId+1, pair.Assistant.CallId); !errors.Is(err, ErrNotFound) {
		t.Fatalf("FindCall must reject another owner, got %v", err)
	}
}

func TestFinishCallSummaryTimeIncludesParentLockWait(t *testing.T) {
	h := newCallModelHarness(t)
	conversationId := h.createConversation("锁等待")
	pair := h.reserve(conversationId)
	blocker, err := h.db.BeginTx(h.ctx, nil)
	if err != nil {
		t.Fatal("begin parent lock transaction failed")
	}
	defer blocker.Rollback()
	if _, err := blocker.ExecContext(h.ctx, "UPDATE ai_conversation SET title=title WHERE id=?", conversationId); err != nil {
		t.Fatal("lock parent conversation failed")
	}
	h.conn.reset()
	started := make(chan struct{}, 1)
	h.conn.onQuery = func(query string) {
		if strings.Contains(query, "ai_conversation") && strings.Contains(query, "FOR UPDATE") {
			select {
			case started <- struct{}{}:
			default:
			}
		}
	}
	finished := make(chan error, 1)
	go func() { finished <- h.messages.FinishCall(h.ctx, h.ownerId, pair, "succeeded", "answer", "{}", 1) }()
	select {
	case <-started:
	case <-h.ctx.Done():
		t.Fatal("finish never attempted the parent lock")
	}
	// 跨过完整一秒，能识别在等待锁前采样完成时间的回归。
	time.Sleep(1100 * time.Millisecond)
	if _, err := blocker.ExecContext(h.ctx, "INSERT INTO ai_message (conversation_id,role,content,status) VALUES (?,'assistant','concurrent history','succeeded')", conversationId); err != nil {
		t.Fatal("insert history before releasing parent lock failed")
	}
	releasedAt := time.Now().Unix()
	if err := blocker.Commit(); err != nil {
		t.Fatal("release parent lock failed")
	}
	if err := <-finished; err != nil {
		t.Fatalf("FinishCall after lock wait: %v", err)
	}
	summary := pair.Conversation
	if summary == nil || summary.UpdatedAt < releasedAt {
		t.Fatalf("summary predates completed parent lock wait: %+v release=%d", summary, releasedAt)
	}
	if summary.MessageCount != 3 {
		t.Fatalf("summary missed history committed before the parent lock was acquired: count=%d", summary.MessageCount)
	}
}

func TestFinishCallDeletedConversationStillSettlesUsage(t *testing.T) {
	h := newCallModelHarness(t)
	conversationId := h.createConversation("删除后结算")
	pair := h.reserve(conversationId)
	if err := h.conversations.DeleteOwned(h.ctx, conversationId, h.ownerId); err != nil {
		t.Fatalf("DeleteOwned: %v", err)
	}
	if err := h.messages.SaveCallUsage(h.ctx, h.ownerId, pair, 9, `{"checkpoint":true}`); err != nil {
		t.Fatalf("SaveCallUsage after deletion: %v", err)
	}
	if err := h.messages.FinishCall(h.ctx, h.ownerId, pair, "succeeded", "answer", `{"usage_complete":true}`, 12); err != nil {
		t.Fatalf("FinishCall after deletion: %v", err)
	}
	if pair.Conversation != nil {
		t.Fatal("finish must not restore a deleted conversation through its summary")
	}
	var row struct {
		Status  string `db:"status"`
		Tokens  int64  `db:"tokens"`
		Deleted bool   `db:"deleted"`
	}
	if err := h.raw.QueryRowCtx(h.ctx, &row, "SELECT m.status,m.tokens,c.deleted_at IS NOT NULL AS deleted FROM ai_message m JOIN ai_conversation c ON c.id=m.conversation_id WHERE m.id=?", pair.Assistant.Id); err != nil {
		t.Fatal("read deleted call settlement failed")
	}
	if row.Status != "succeeded" || row.Tokens != 12 || !row.Deleted {
		t.Fatalf("deleted call did not settle without restoration: %+v", row)
	}
}

func TestSaveCallUsageKeepsLightweightParentLock(t *testing.T) {
	h := newCallModelHarness(t)
	pair := h.reserve(h.createConversation("用量检查点"))
	h.conn.reset()
	if err := h.messages.SaveCallUsage(h.ctx, h.ownerId, pair, 8, `{"checkpoint":true}`); err != nil {
		t.Fatalf("SaveCallUsage: %v", err)
	}
	queries := h.conn.snapshot()
	if len(queries) != 3 || countQueries(queries, "COUNT(") != 0 || !strings.HasPrefix(strings.ToUpper(queries[0]), "SELECT ID FROM AI_CONVERSATION") {
		t.Fatalf("usage checkpoint added expensive summary reads: statements=%d counts=%d", len(queries), countQueries(queries, "COUNT("))
	}
	if pair.Conversation != nil {
		t.Fatal("usage checkpoint must not expose a terminal conversation summary")
	}
}

func TestFinishCallFailureClearsPartialAnswerAndRejectsReplay(t *testing.T) {
	h := newCallModelHarness(t)
	pair := h.reserve(h.createConversation("失败结算"))
	if err := h.messages.FinishCall(h.ctx, h.ownerId, pair, "failed", "partial untrusted answer", `{"error_code":503}`, 6); err != nil {
		t.Fatalf("FinishCall failed status: %v", err)
	}
	replay, err := h.messages.FindCall(h.ctx, h.ownerId, pair.Assistant.CallId)
	if err != nil {
		t.Fatalf("FindCall failed result: %v", err)
	}
	if replay.User.Status != "failed" || replay.Assistant.Status != "failed" || replay.Assistant.Content != "" || replay.Assistant.Tokens != 6 {
		t.Fatal("failed terminal pair did not preserve atomic status, empty answer and usage")
	}
	if err := h.messages.FinishCall(h.ctx, h.ownerId, pair, "succeeded", "late answer", "{}", 99); !errors.Is(err, ErrCallState) {
		t.Fatalf("terminal replay must be rejected, got %v", err)
	}
	if pair.Conversation != nil {
		t.Fatal("rejected terminal mutation exposed a stale summary")
	}
}

func TestFinishCallRollbackAndOwnershipDoNotExposeSummary(t *testing.T) {
	h := newCallModelHarness(t)
	pair := h.reserve(h.createConversation("原子回滚"))
	if err := h.messages.FinishCall(h.ctx, h.ownerId+1, pair, "succeeded", "answer", "{}", 1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong owner must be rejected, got %v", err)
	}
	h.exec("UPDATE ai_message SET status='failed' WHERE id=?", pair.Assistant.Id)
	if err := h.messages.FinishCall(h.ctx, h.ownerId, pair, "succeeded", "answer", "{}", 1); !errors.Is(err, ErrCallState) {
		t.Fatalf("incomplete running pair must be rejected, got %v", err)
	}
	if pair.Conversation != nil {
		t.Fatal("rolled back finish exposed a conversation summary")
	}
	var userStatus string
	if err := h.db.QueryRowContext(h.ctx, "SELECT status FROM ai_message WHERE id=?", pair.User.Id).Scan(&userStatus); err != nil || userStatus != "running" {
		t.Fatal("rejected finish partially mutated the user message")
	}
}

func TestFinishCallCacheFailureReportsCommittedWithoutSummary(t *testing.T) {
	h := newCallModelHarness(t)
	pair := h.reserve(h.createConversation("缓存失败"))
	broken := *h.messages
	defaults := *h.messages.defaultAiMessageModel
	defaults.CachedConn = sqlc.NewConnWithCache(h.conn, rejectedCallCache{})
	broken.defaultAiMessageModel = &defaults
	if err := broken.FinishCall(h.ctx, h.ownerId, pair, "succeeded", "committed answer", "{}", 4); !errors.Is(err, ErrCallCommitted) {
		t.Fatalf("cache failure must report committed state, got %v", err)
	}
	if pair.Conversation != nil {
		t.Fatal("cache invalidation failure exposed a successful summary")
	}
	var count int64
	if err := h.db.QueryRowContext(h.ctx, "SELECT COUNT(*) FROM ai_message WHERE id IN (?,?) AND status='succeeded'", pair.User.Id, pair.Assistant.Id).Scan(&count); err != nil || count != 2 {
		t.Fatal("cache failure did not retain the atomically committed message pair")
	}
}

func TestFinishCallWriteFailureRollsBackBothMessages(t *testing.T) {
	h := newCallModelHarness(t)
	pair := h.reserve(h.createConversation("写入失败回滚"))
	// TEXT 超过列宽会在用户消息更新成功后的助手消息写入处失败。
	err := h.messages.FinishCall(h.ctx, h.ownerId, pair, "succeeded", strings.Repeat("x", 65536), "{}", 15)
	if err == nil || errors.Is(err, ErrCallCommitted) {
		t.Fatal("oversized answer must fail before the terminal transaction commits")
	}
	if pair.Conversation != nil {
		t.Fatal("write rollback exposed a terminal summary")
	}
	replay, err := h.messages.FindCall(h.ctx, h.ownerId, pair.Assistant.CallId)
	if err != nil {
		t.Fatalf("FindCall after rollback: %v", err)
	}
	if replay.User.Status != "running" || replay.Assistant.Status != "running" || replay.Assistant.Content != "" || replay.Assistant.Tokens != 0 {
		t.Fatal("failed assistant write did not roll back the entire terminal pair")
	}
}

func TestFinishCallConcurrentSettlementsCommitOnePair(t *testing.T) {
	h := newCallModelHarness(t)
	pair := h.reserve(h.createConversation("并发终态"))
	first, second := *pair, *pair
	start := make(chan struct{})
	type outcome struct {
		pair *CallPair
		err  error
	}
	results := make(chan outcome, 2)
	go func() {
		<-start
		results <- outcome{&first, h.messages.FinishCall(h.ctx, h.ownerId, &first, "succeeded", "winner answer", "{}", 7)}
	}()
	go func() {
		<-start
		results <- outcome{&second, h.messages.FinishCall(h.ctx, h.ownerId, &second, "failed", "partial", "{}", 3)}
	}()
	close(start)
	winner, loser := <-results, <-results
	if winner.err != nil {
		winner, loser = loser, winner
	}
	if winner.err != nil || !errors.Is(loser.err, ErrCallState) {
		t.Fatalf("concurrent settlement must have one winner: first=%v second=%v", winner.err, loser.err)
	}
	if winner.pair.Conversation == nil || loser.pair.Conversation != nil {
		t.Fatal("only the committed terminal mutation may expose a summary")
	}
	replay, err := h.messages.FindCall(h.ctx, h.ownerId, pair.Assistant.CallId)
	if err != nil {
		t.Fatalf("FindCall after concurrent settlement: %v", err)
	}
	if replay.User.Status != replay.Assistant.Status || (replay.Assistant.Status != "succeeded" && replay.Assistant.Status != "failed") {
		t.Fatal("concurrent settlement produced mixed terminal message states")
	}
	if replay.Assistant.Status == "succeeded" && (replay.Assistant.Content != "winner answer" || replay.Assistant.Tokens != 7) {
		t.Fatal("succeeded settlement did not preserve its own answer and usage")
	}
	if replay.Assistant.Status == "failed" && (replay.Assistant.Content != "" || replay.Assistant.Tokens != 3) {
		t.Fatal("failed settlement did not clear its answer and preserve its own usage")
	}
}

func TestConversationCheckOwnedUsesNoMessageCount(t *testing.T) {
	h := newCallModelHarness(t)
	conversationId := h.createConversation("轻量归属")
	h.conn.reset()
	if err := h.conversations.CheckOwned(h.ctx, conversationId, h.ownerId); err != nil {
		t.Fatalf("CheckOwned: %v", err)
	}
	queries := h.conn.snapshot()
	if len(queries) != 1 || countQueries(queries, "COUNT(") != 0 || strings.Contains(strings.ToUpper(queries[0]), "AI_MESSAGE") {
		t.Fatal("ownership check unnecessarily reads conversation history")
	}
	if err := h.conversations.CheckOwned(h.ctx, conversationId, h.ownerId+1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("CheckOwned must reject another owner, got %v", err)
	}
	if err := h.conversations.DeleteOwned(h.ctx, conversationId, h.ownerId); err != nil {
		t.Fatalf("DeleteOwned: %v", err)
	}
	if err := h.conversations.CheckOwned(h.ctx, conversationId, h.ownerId); !errors.Is(err, ErrNotFound) {
		t.Fatalf("CheckOwned must reject deleted conversation, got %v", err)
	}
}

type callModelHarness struct {
	t             *testing.T
	ctx           context.Context
	db            *sql.DB
	raw           sqlx.SqlConn
	conn          *countedCallConn
	messages      *customAiMessageModel
	conversations AiConversationModel
	ownerId       int64
	prefix        string
	ids           []int64
	pairs         []*CallPair
}

func newCallModelHarness(t *testing.T) *callModelHarness {
	t.Helper()
	dsn, redisHost := os.Getenv("GOZERO_AI_TEST_DSN"), os.Getenv("GOZERO_AI_TEST_REDIS")
	if dsn == "" || redisHost == "" {
		t.Skip("requires isolated MySQL/Redis: GOZERO_AI_TEST_DSN and GOZERO_AI_TEST_REDIS")
	}
	sqlx.DisableLog()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	db, err := sql.Open("mysql", dsn)
	if err != nil || db.PingContext(ctx) != nil {
		t.Fatal("isolated MySQL connection unavailable")
	}
	t.Cleanup(func() { _ = db.Close() })
	raw := sqlx.NewSqlConnFromDB(db)
	conn := &countedCallConn{SqlConn: raw}
	conf := cache.CacheConf{{RedisConf: redis.RedisConf{Host: redisHost, Type: "node"}, Weight: 100}}
	if _, err := redis.NewRedis(conf[0].RedisConf); err != nil {
		t.Fatal("isolated Redis connection unavailable")
	}
	h := &callModelHarness{
		t: t, ctx: ctx, db: db, raw: raw, conn: conn,
		messages:      NewAiMessageModel(conn, conf).(*customAiMessageModel),
		conversations: NewAiConversationModel(conn, conf),
		ownerId:       998811, prefix: "ai-model-" + uuid.NewString(),
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		for _, pair := range h.pairs {
			if err := h.messages.invalidatePair(cleanupCtx, pair); err != nil {
				t.Error("cleanup precise model cache keys failed")
			}
		}
		for _, id := range h.ids {
			if _, err := db.ExecContext(cleanupCtx, "DELETE FROM ai_message WHERE conversation_id=?", id); err != nil {
				t.Error("cleanup exact conversation messages failed")
			}
			if _, err := db.ExecContext(cleanupCtx, "DELETE FROM ai_conversation WHERE id=? AND user_id=?", id, h.ownerId); err != nil {
				t.Error("cleanup exact conversation failed")
			}
		}
	})
	return h
}

func (h *callModelHarness) exec(query string, args ...any) sql.Result {
	h.t.Helper()
	result, err := h.db.ExecContext(h.ctx, query, args...)
	if err != nil {
		h.t.Fatalf("integration fixture write failed: %v", err)
	}
	return result
}

func (h *callModelHarness) createConversation(title string) int64 {
	h.t.Helper()
	result := h.exec("INSERT INTO ai_conversation (user_id,title,created_at,updated_at) VALUES (?,?,FROM_UNIXTIME(1577836800),FROM_UNIXTIME(1577836800))", h.ownerId, title)
	id, err := result.LastInsertId()
	if err != nil {
		h.t.Fatal("read fixture conversation ID failed")
	}
	h.ids = append(h.ids, id)
	return id
}

func (h *callModelHarness) reserve(conversationId int64) *CallPair {
	h.t.Helper()
	pair, fresh, err := h.messages.ReserveCall(h.ctx, h.ownerId, conversationId, h.prefix+fmt.Sprint(len(h.pairs)), "input", "{}")
	if err != nil || !fresh {
		h.t.Fatalf("reserve fixture call: fresh=%v err=%v", fresh, err)
	}
	h.pairs = append(h.pairs, pair)
	return pair
}

// 包装真实 sqlx 连接，记录业务实际发出的 SQL；事务继续使用真实 session。
type countedCallConn struct {
	sqlx.SqlConn
	mu      sync.Mutex
	queries []string
	onQuery func(string)
}

func (c *countedCallConn) record(query string) {
	c.mu.Lock()
	c.queries = append(c.queries, query)
	hook := c.onQuery
	c.mu.Unlock()
	if hook != nil {
		hook(query)
	}
}

func (c *countedCallConn) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.queries = nil
}

func (c *countedCallConn) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.queries...)
}

func (c *countedCallConn) QueryRowCtx(ctx context.Context, v any, query string, args ...any) error {
	c.record(query)
	return c.SqlConn.QueryRowCtx(ctx, v, query, args...)
}

func (c *countedCallConn) QueryRowsCtx(ctx context.Context, v any, query string, args ...any) error {
	c.record(query)
	return c.SqlConn.QueryRowsCtx(ctx, v, query, args...)
}

func (c *countedCallConn) TransactCtx(ctx context.Context, fn func(context.Context, sqlx.Session) error) error {
	return c.SqlConn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		return fn(ctx, countedCallSession{Session: session, conn: c})
	})
}

type countedCallSession struct {
	sqlx.Session
	conn *countedCallConn
}

func (s countedCallSession) QueryRowCtx(ctx context.Context, v any, query string, args ...any) error {
	s.conn.record(query)
	return s.Session.QueryRowCtx(ctx, v, query, args...)
}

func (s countedCallSession) QueryRowsCtx(ctx context.Context, v any, query string, args ...any) error {
	s.conn.record(query)
	return s.Session.QueryRowsCtx(ctx, v, query, args...)
}

func (s countedCallSession) ExecCtx(ctx context.Context, query string, args ...any) (sql.Result, error) {
	s.conn.record(query)
	return s.Session.ExecCtx(ctx, query, args...)
}

func countQueries(queries []string, fragment string) int {
	count := 0
	for _, query := range queries {
		if strings.Contains(strings.ToUpper(query), fragment) {
			count++
		}
	}
	return count
}

type rejectedCallCache struct{ cache.Cache }

func (rejectedCallCache) DelCtx(context.Context, ...string) error {
	return errors.New("injected cache invalidation failure")
}
