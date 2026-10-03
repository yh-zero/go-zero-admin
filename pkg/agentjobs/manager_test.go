package agentjobs

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go-zero-admin/pkg/audit"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type fixtureOwner struct {
	ID        int64 `gorm:"primaryKey"`
	Enable    int64
	DeletedAt *time.Time
}

func (fixtureOwner) TableName() string { return "sys_users" }

func fixture(t *testing.T) *gorm.DB {
	t.Helper()
	// A canceled SQLite query may discard its connection. A file fixture keeps
	// the installed schema when that connection is the pool's only connection.
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "agent.sqlite")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	connection, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = connection.Close() })
	if err := db.AutoMigrate(&Conversation{}, &Message{}, &Run{}, &fixtureOwner{}, &audit.Event{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&[]fixtureOwner{{ID: 1, Enable: 1}, {ID: 2, Enable: 1}}).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func testConfig() Config {
	return Config{Enabled: true, PollInterval: 10 * time.Millisecond, RunTimeout: time.Second, QueueTimeout: time.Second, LeaseDuration: 300 * time.Millisecond, HeartbeatInterval: 50 * time.Millisecond, UserActiveLimit: 2}
}
func manager(t *testing.T, db *gorm.DB, execute Executor) *Manager {
	t.Helper()
	m, err := New(db, testConfig(), execute)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	return m
}
func request(t *testing.T) SubmitInput {
	t.Helper()
	id, err := newID()
	if err != nil {
		t.Fatal(err)
	}
	return SubmitInput{OwnerID: 1, AuthorityID: 1, SessionVersion: 1, RequestID: id, Question: "请检查当前任务"}
}
func submit(t *testing.T, m *Manager, input SubmitInput) Run {
	t.Helper()
	run, err := m.Submit(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	return run
}
func waitRun(t *testing.T, m *Manager, id, status string) Run {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		run, err := m.Get(context.Background(), 1, id)
		if err != nil {
			t.Fatal(err)
		}
		if run.Status == status {
			return run
		}
		time.Sleep(5 * time.Millisecond)
	}
	run, _ := m.Get(context.Background(), 1, id)
	t.Fatalf("run status=%s, want %s", run.Status, status)
	return Run{}
}

func TestSubmissionDedupOwnershipAndAuditRollback(t *testing.T) {
	db := fixture(t)
	m := manager(t, db, nil)
	input := request(t)
	original := submit(t, m, input)
	reused := submit(t, m, input)
	if !reused.Reused || reused.ID != original.ID || reused.ConversationID != original.ConversationID {
		t.Fatal("duplicate request created another run")
	}
	changed := input
	changed.Question += "!"
	if _, err := m.Submit(context.Background(), changed); !errors.Is(err, ErrConflict) {
		t.Fatal("different question reused request ID", err)
	}
	changed = input
	changed.ConversationID = original.ConversationID
	if _, err := m.Submit(context.Background(), changed); !errors.Is(err, ErrConflict) {
		t.Fatal("different conversation reused request ID", err)
	}
	if _, err := m.Get(context.Background(), 2, original.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("another owner could read run", err)
	}
	if _, err := m.Cancel(context.Background(), 2, original.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("another owner could cancel run", err)
	}
	if _, _, err := m.ListMessages(context.Background(), 2, original.ConversationID, 1, 20); !errors.Is(err, ErrNotFound) {
		t.Fatal("another owner could read history", err)
	}
	if err := db.Callback().Create().Before("gorm:create").Register("test:reject-agent-audit", func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_audit_logs" {
			tx.AddError(errors.New("audit unavailable"))
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Create().Remove("test:reject-agent-audit") })
	if _, err := m.Submit(context.Background(), request(t)); err == nil {
		t.Fatal("submit succeeded without audit")
	}
	var runs, messages, conversations int64
	db.Model(&Run{}).Count(&runs)
	db.Model(&Message{}).Count(&messages)
	db.Model(&Conversation{}).Count(&conversations)
	if runs != 1 || messages != 1 || conversations != 1 {
		t.Fatal("audit failure left partial submission", runs, messages, conversations)
	}
	if _, err := m.Cancel(context.Background(), 1, original.ID); err == nil {
		t.Fatal("cancel succeeded without audit")
	}
	unchanged, _ := m.Get(context.Background(), 1, original.ID)
	if unchanged.Status != StatusQueued {
		t.Fatal("cancel audit failure did not roll back")
	}
}

func TestConversationAndOwnerLimitsReleaseOnCancellation(t *testing.T) {
	db := fixture(t)
	m := manager(t, db, nil)
	first := submit(t, m, request(t))
	other := request(t)
	other.ConversationID = first.ConversationID
	if _, err := m.Submit(context.Background(), other); !errors.Is(err, ErrBusy) {
		t.Fatal("parallel run in same conversation", err)
	}
	submit(t, m, request(t))
	if _, err := m.Submit(context.Background(), request(t)); !errors.Is(err, ErrBusy) {
		t.Fatal("owner active run limit exceeded", err)
	}
	cancelled, err := m.Cancel(context.Background(), 1, first.ID)
	if err != nil || cancelled.Status != StatusCanceled {
		t.Fatal("cancel failed", err)
	}
	if _, err := m.Cancel(context.Background(), 1, first.ID); err != nil {
		t.Fatal("repeat cancel failed", err)
	}
	submit(t, m, other)
	invalid := request(t)
	invalid.Question = string([]byte{0xff})
	if _, err := m.Submit(context.Background(), invalid); !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid UTF8 accepted", err)
	}
	var logs []audit.Event
	if err := db.Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	for _, event := range logs {
		if strings.Contains(event.Params, "当前任务") || event.Object == invalid.Question {
			t.Fatal("question leaked into audit")
		}
	}
}

func TestTwoSchedulersClaimOnceAndCancellationWins(t *testing.T) {
	db := fixture(t)
	var calls atomic.Int64
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	execute := func(ctx context.Context, _ Job) (Result, error) {
		calls.Add(1)
		started <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
		}
		return Result{Text: "late answer"}, nil
	}
	first := manager(t, db, execute)
	second := manager(t, db, execute)
	run := submit(t, first, request(t))
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	if err := second.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("worker did not claim")
	}
	active := waitRun(t, first, run.ID, StatusRunning)
	owner, other := first, second
	if active.LeaseOwner == second.id {
		owner, other = second, first
	}
	progress := `[{"name":"getMenus","status":"running","error":"","summary":""}]`
	if err := owner.UpdateProgress(context.Background(), run.ID, progress); err != nil {
		t.Fatal(err)
	}
	if err := owner.UpdateProgress(context.Background(), run.ID, progress); err != nil {
		t.Fatal("identical progress failed", err)
	}
	if err := other.UpdateProgress(context.Background(), run.ID, progress); !errors.Is(err, ErrLeaseLost) {
		t.Fatal("another manager updated lease", err)
	}
	if _, err := other.Cancel(context.Background(), 1, run.ID); err != nil {
		t.Fatal(err)
	}
	if err := owner.UpdateProgress(context.Background(), run.ID, progress); !errors.Is(err, ErrLeaseLost) {
		t.Fatal("progress revived canceled run", err)
	}
	close(release)
	time.Sleep(80 * time.Millisecond)
	final := waitRun(t, first, run.ID, StatusCanceled)
	if calls.Load() != 1 || final.Text != "" || final.ExtraJSON != progress {
		t.Fatal("duplicate execution or late completion overwrote cancellation", calls.Load(), final.Text)
	}
	var assistants int64
	db.Model(&Message{}).Where("run_id=? AND role=?", run.ID, "assistant").Count(&assistants)
	if assistants != 0 {
		t.Fatal("canceled run got assistant message")
	}
}

func TestHistoryUsesOnlyRecentCompletedMessages(t *testing.T) {
	db := fixture(t)
	m := manager(t, db, nil)
	first := submit(t, m, request(t))
	now := time.Now().UTC().Add(-time.Minute)
	if err := db.Model(&Run{}).Where("id=?", first.ID).Update("sequence", 100).Error; err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 8; index++ {
		id, _ := newID()
		requestID, _ := newID()
		status := StatusSucceeded
		if index == 0 {
			status = StatusFailed
		}
		previous := Run{ID: id, OwnerID: 1, ConversationID: first.ConversationID, Sequence: int64(index + 1), RequestID: requestID, Status: status, QueueExpiresAt: now, ExtraJSON: "[]", CreatedAt: now, UpdatedAt: now}
		if err := db.Create(&previous).Error; err != nil {
			t.Fatal(err)
		}
		for _, role := range []string{"user", "assistant"} {
			messageID, _ := newID()
			if err := db.Create(&Message{ID: messageID, OwnerID: 1, ConversationID: first.ConversationID, RunID: id, Role: role, Content: role, CreatedAt: now}).Error; err != nil {
				t.Fatal(err)
			}
			now = now.Add(time.Second)
		}
	}
	m.execute = func(_ context.Context, job Job) (Result, error) {
		if len(job.History) != 12 {
			t.Errorf("history length=%d", len(job.History))
		}
		for _, message := range job.History {
			if message.RunID == first.ID {
				t.Error("current question duplicated in history")
			}
		}
		for index, message := range job.History {
			if (index%2 == 0 && message.Role != "user") || (index%2 == 1 && message.Role != "assistant") {
				t.Error("history role order is unstable")
			}
		}
		return Result{Text: "完成", Provider: "test", Model: "mock", InputTokens: 12, OutputTokens: 2, ExtraJSON: "[]"}, nil
	}
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	final := waitRun(t, m, first.ID, StatusSucceeded)
	if final.Text != "完成" || final.InputTokens != 12 || final.OutputTokens != 2 {
		t.Fatal("result or usage missing")
	}
	values, total, err := m.ListMessages(context.Background(), 1, first.ConversationID, 1, 100)
	if err != nil || total != 18 || len(values) != 18 {
		t.Fatal("history pagination failed", len(values), total, err)
	}
	if values[0].RunID != first.ID || values[0].Role != "assistant" || values[1].RunID != first.ID || values[1].Role != "user" {
		t.Fatal("latest run is not first in descending history")
	}
	newest, _, err := m.ListMessages(context.Background(), 1, first.ConversationID, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	previous, _, err := m.ListMessages(context.Background(), 1, first.ConversationID, 2, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(newest) != 1 || len(previous) != 1 || newest[0].RunID != first.ID || previous[0].RunID != first.ID || newest[0].Role != "assistant" || previous[0].Role != "user" {
		t.Fatal("pagination split newest pair in the wrong order")
	}
	// Reversing DESC pages restores user before assistant, including a pair
	// split over adjacent one-item pages.
	chronological := []Message{previous[0], newest[0]}
	if chronological[0].Role != "user" || chronological[1].Role != "assistant" {
		t.Fatal("descending pair cannot be reversed for display")
	}
	conversations, count, err := m.ListConversations(context.Background(), 1, 1, 20)
	if err != nil || count != 1 || len(conversations) != 1 {
		t.Fatal("conversation pagination failed", err)
	}
}

func TestExpiredRunsAreInterruptedWithoutReplay(t *testing.T) {
	db := fixture(t)
	m := manager(t, db, nil)
	queued := submit(t, m, request(t))
	running := submit(t, m, request(t))
	past := time.Now().UTC().Add(-time.Second)
	db.Model(&Run{}).Where("id=?", queued.ID).Update("queue_expires_at", past)
	db.Model(&Run{}).Where("id=?", running.ID).Updates(map[string]any{"status": StatusRunning, "lease_owner": "dead-process", "lease_until": past, "deadline_at": past})
	var calls atomic.Int64
	m.execute = func(context.Context, Job) (Result, error) { calls.Add(1); return Result{}, nil }
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	waitRun(t, m, queued.ID, StatusInterrupted)
	waitRun(t, m, running.ID, StatusInterrupted)
	time.Sleep(30 * time.Millisecond)
	if calls.Load() != 0 {
		t.Fatal("expired run replayed")
	}
}

func TestTimeoutAndCloseCancelExecutorWithoutReplay(t *testing.T) {
	for _, closeEarly := range []bool{false, true} {
		t.Run(map[bool]string{false: "timeout", true: "close"}[closeEarly], func(t *testing.T) {
			db := fixture(t)
			cfg := testConfig()
			cfg.RunTimeout = 70 * time.Millisecond
			var calls atomic.Int64
			entered := make(chan struct{})
			m, err := New(db, cfg, func(ctx context.Context, _ Job) (Result, error) {
				calls.Add(1)
				close(entered)
				<-ctx.Done()
				return Result{}, ctx.Err()
			})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(m.Close)
			run := submit(t, m, request(t))
			if err := m.Start(); err != nil {
				t.Fatal(err)
			}
			<-entered
			if closeEarly {
				done := make(chan struct{})
				go func() { m.Close(); close(done) }()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Fatal("Close did not cancel worker")
				}
			}
			waitRun(t, m, run.ID, StatusInterrupted)
			other := manager(t, db, func(context.Context, Job) (Result, error) { calls.Add(1); return Result{}, nil })
			if err := other.Start(); err != nil {
				t.Fatal(err)
			}
			time.Sleep(30 * time.Millisecond)
			if calls.Load() != 1 {
				t.Fatal("interrupted run replayed")
			}
		})
	}
}

func TestDisabledConstructionAndSchemaGuard(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	connection, _ := db.DB()
	t.Cleanup(func() { _ = connection.Close() })
	m, err := New(db, Config{}, nil)
	if err != nil {
		t.Fatal("disabled manager needs schema", err)
	}
	t.Cleanup(m.Close)
	if err := m.Start(); !errors.Is(err, ErrDisabled) {
		t.Fatal(err)
	}
	if _, err := m.Submit(context.Background(), request(t)); !errors.Is(err, ErrDisabled) {
		t.Fatal(err)
	}
	enabled := manager(t, db, func(context.Context, Job) (Result, error) { return Result{}, nil })
	if err := enabled.Start(); err == nil {
		t.Fatal("scheduler started without migration")
	}
}

func TestExecutorFailureAndPanicDoNotExposeRawErrors(t *testing.T) {
	for _, panics := range []bool{false, true} {
		t.Run(map[bool]string{false: "error", true: "panic"}[panics], func(t *testing.T) {
			db := fixture(t)
			m := manager(t, db, func(context.Context, Job) (Result, error) {
				if panics {
					panic("provider secret")
				}
				return Result{Text: "secret late answer", ExtraJSON: "not valid json", InputTokens: 4}, errors.New("provider secret")
			})
			run := submit(t, m, request(t))
			if err := m.Start(); err != nil {
				t.Fatal(err)
			}
			result := waitRun(t, m, run.ID, StatusFailed)
			if result.ErrorCode != "execution_failed" || strings.Contains(result.Error, "secret") || result.Text != "" {
				t.Fatal("raw provider error exposed")
			}
			if !panics && result.InputTokens != 4 {
				t.Fatal("known failure usage was lost")
			}
		})
	}
}

func TestInvalidSuccessfulResultPreservesKnownUsage(t *testing.T) {
	db := fixture(t)
	m := manager(t, db, func(context.Context, Job) (Result, error) {
		return Result{Text: "answer exceeds cap", Provider: "test", Model: "model", InputTokens: 5, OutputTokens: 2, ExtraJSON: "[]"}, nil
	})
	m.cfg.MaxResultBytes = 8
	run := submit(t, m, request(t))
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	failed := waitRun(t, m, run.ID, StatusFailed)
	if failed.ErrorCode != "invalid_result" || failed.Text != "" || failed.InputTokens != 5 || failed.OutputTokens != 2 || failed.Provider != "test" {
		t.Fatal("invalid answer leaked or known usage was discarded")
	}
	var assistants int64
	if err := db.Model(&Message{}).Where("run_id=? AND role=?", run.ID, "assistant").Count(&assistants).Error; err != nil || assistants != 0 {
		t.Fatal("failed run wrote assistant", err)
	}
}

func TestDeadlinePreservesProgressAndStableMessageOrder(t *testing.T) {
	db := fixture(t)
	cfg := testConfig()
	cfg.RunTimeout = 60 * time.Millisecond
	var m *Manager
	m, err := New(db, cfg, func(ctx context.Context, job Job) (Result, error) {
		if err := m.UpdateProgress(ctx, job.Run.ID, `[{"name":"read","status":"running"}]`); err != nil {
			t.Error(err)
		}
		<-ctx.Done()
		return Result{}, ctx.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	run := submit(t, m, request(t))
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	expired := waitRun(t, m, run.ID, StatusInterrupted)
	if !strings.Contains(expired.ExtraJSON, "read") {
		t.Fatal("timeout erased progress")
	}
	// UUID lexicographic order must not reverse user/assistant at datetime(3).
	now := time.Now().UTC()
	db.Model(&Run{}).Where("id=?", run.ID).Update("status", StatusSucceeded)
	assistantID := "00000000-0000-4000-8000-000000000001"
	if err := db.Create(&Message{ID: assistantID, OwnerID: 1, ConversationID: run.ConversationID, RunID: run.ID, Role: "assistant", Content: "answer", CreatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&Message{}).Where("run_id=?", run.ID).Update("created_at", now).Error; err != nil {
		t.Fatal(err)
	}
	messages, _, err := m.ListMessages(context.Background(), 1, run.ConversationID, 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(messages) != 2 || messages[0].Role != "assistant" || messages[1].Role != "user" {
		t.Fatal("same-timestamp messages reordered")
	}
}

func TestLateResultAndDisabledReadCannotReviveExpiredRun(t *testing.T) {
	db := fixture(t)
	m := manager(t, db, nil)
	run := submit(t, m, request(t))
	past := time.Now().UTC().Add(-time.Second)
	trace := `[{"name":"read","status":"completed","summary":"ok"}]`
	if err := db.Model(&Run{}).Where("id=?", run.ID).Updates(map[string]any{"status": StatusRunning, "lease_owner": m.id, "lease_until": past, "deadline_at": past, "extra_json": trace}).Error; err != nil {
		t.Fatal(err)
	}
	m.finish(run, Result{Text: "late private answer", Provider: "test", Model: "model", InputTokens: 4, ExtraJSON: "[]"}, nil)
	finished, err := m.Get(context.Background(), 1, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != StatusInterrupted || finished.Text != "" || finished.ExtraJSON != trace || finished.InputTokens != 4 {
		t.Fatal("late response leaked or erased usage/trace", finished)
	}
	queued := submit(t, m, request(t))
	if err := db.Model(&Run{}).Where("id=?", queued.ID).Update("queue_expires_at", past).Error; err != nil {
		t.Fatal(err)
	}
	disabled, err := New(db, Config{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(disabled.Close)
	expired, err := disabled.Get(context.Background(), 1, queued.ID)
	if err != nil || expired.Status != StatusInterrupted {
		t.Fatal("disabled read left expired run pending", err)
	}
}

func TestNonCooperativeExecutorDoesNotSpawnMoreCalls(t *testing.T) {
	db := fixture(t)
	cfg := testConfig()
	cfg.RunTimeout = 50 * time.Millisecond
	release := make(chan struct{})
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	var calls atomic.Int64
	m, err := New(db, cfg, func(context.Context, Job) (Result, error) { calls.Add(1); <-release; return Result{Text: "late"}, nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Close)
	first := submit(t, m, request(t))
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	waitRun(t, m, first.ID, StatusInterrupted)
	second := submit(t, m, request(t))
	time.Sleep(80 * time.Millisecond)
	if calls.Load() != 1 {
		t.Fatal("timed-out executor produced unbounded new invocations")
	}
	current, err := m.Get(context.Background(), 1, second.ID)
	if err != nil || current.Status != StatusQueued {
		t.Fatal("worker did not wait for previous invocation", err)
	}
	m.Close()
	once.Do(func() { close(release) })
}
