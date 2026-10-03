package agentjobs

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestCancellationBetweenClaimAndExecutionSkipsProvider(t *testing.T) {
	db := fixture(t)
	var calls atomic.Int64
	m := manager(t, db, func(context.Context, Job) (Result, error) {
		calls.Add(1)
		return Result{Text: "unnecessary provider result"}, nil
	})
	run := submit(t, m, request(t))
	claimed, err := m.claim(context.Background())
	if err != nil || claimed.ID != run.ID {
		t.Fatal("claim failed", err)
	}
	other := manager(t, db, nil)
	if _, err := other.Cancel(context.Background(), 1, run.ID); err != nil {
		t.Fatal(err)
	}
	m.process(claimed)
	if calls.Load() != 0 {
		t.Fatal("already-cancelled claimed run called the provider")
	}
	final, err := m.Get(context.Background(), 1, run.ID)
	if err != nil || final.Status != StatusCanceled || final.Text != "" {
		t.Fatal("cancelled run changed", err)
	}
}

func TestSubmissionExpiresOwnStaleRunsBeforeConcurrencyCheck(t *testing.T) {
	db := fixture(t)
	m := manager(t, db, nil)
	first := submit(t, m, request(t))
	second := submit(t, m, request(t))
	past := time.Now().UTC().Add(-time.Second)
	if err := db.Model(&Run{}).Where("id IN ?", []string{first.ID, second.ID}).Update("queue_expires_at", past).Error; err != nil {
		t.Fatal(err)
	}
	next := request(t)
	next.ConversationID = first.ConversationID
	if _, err := m.Submit(context.Background(), next); err != nil {
		t.Fatal("expired queue still occupied owner/conversation limit", err)
	}
	var outdated []Run
	if err := db.Where("id IN ?", []string{first.ID, second.ID}).Find(&outdated).Error; err != nil {
		t.Fatal(err)
	}
	for _, run := range outdated {
		if run.Status != StatusInterrupted {
			t.Fatal("expiry was not persisted atomically")
		}
	}
}

func TestCancelledRunRetainsKnownUsageWithoutLateAnswer(t *testing.T) {
	db := fixture(t)
	m := manager(t, db, nil)
	run := submit(t, m, request(t))
	claimed, err := m.claim(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	trace := `[{"name":"read","status":"completed","summary":"safe"}]`
	if err := m.UpdateProgress(context.Background(), run.ID, trace); err != nil {
		t.Fatal(err)
	}
	other := manager(t, db, nil)
	if _, err := other.Cancel(context.Background(), 1, run.ID); err != nil {
		t.Fatal(err)
	}
	m.finish(claimed, Result{Text: "late private answer", Provider: "test", Model: "model", InputTokens: 9, OutputTokens: 4, ExtraJSON: `[{"name":"read","summary":"late private trace"}]`}, context.Canceled)
	finished, err := m.Get(context.Background(), 1, run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if finished.Status != StatusCanceled || finished.Text != "" || finished.ExtraJSON != trace || finished.InputTokens != 9 || finished.OutputTokens != 4 || finished.Provider != "test" || finished.Model != "model" {
		t.Fatal("cancel lost known usage or accepted late result", finished)
	}
	m.finish(claimed, Result{InputTokens: 3, OutputTokens: 2}, context.Canceled)
	other.finish(claimed, Result{InputTokens: 50, OutputTokens: 50}, context.Canceled)
	finished, err = m.Get(context.Background(), 1, run.ID)
	if err != nil || finished.InputTokens != 9 || finished.OutputTokens != 4 {
		t.Fatal("known usage was duplicated/reduced or another lease wrote it", err)
	}
	var assistants int64
	if err := db.Model(&Message{}).Where("run_id=? AND role=?", run.ID, "assistant").Count(&assistants).Error; err != nil || assistants != 0 {
		t.Fatal("cancel wrote assistant message", err)
	}
}

func TestWorkerCancellationRetainsProviderReportedUsage(t *testing.T) {
	db := fixture(t)
	entered := make(chan struct{})
	m := manager(t, db, func(ctx context.Context, _ Job) (Result, error) {
		close(entered)
		<-ctx.Done()
		return Result{Text: "late answer", Provider: "test", Model: "model", InputTokens: 7, OutputTokens: 3}, ctx.Err()
	})
	run := submit(t, m, request(t))
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("worker did not enter")
	}
	if _, err := m.Cancel(context.Background(), 1, run.ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		current, err := m.Get(context.Background(), 1, run.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.InputTokens == 7 && current.OutputTokens == 3 {
			if current.Status != StatusCanceled || current.Text != "" {
				t.Fatal("late response changed cancellation")
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("worker discarded provider-reported usage after cancellation")
}
