package agentjobs

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFinishIncludesRetainedProgressInResultBudget(t *testing.T) {
	for _, excess := range []bool{false, true} {
		name := "boundary"
		if excess {
			name = "excess"
		}
		t.Run(name, func(t *testing.T) {
			db := fixture(t)
			m := manager(t, db, nil)
			trace := `[{"name":"read","status":"completed","summary":"safe"}]`
			m.cfg.MaxResultBytes = len(trace) + 8
			run := submit(t, m, request(t))
			claimed, err := m.claim(context.Background())
			if err != nil || claimed.ID != run.ID {
				t.Fatal("claim failed", err)
			}
			if err := m.UpdateProgress(context.Background(), run.ID, trace); err != nil {
				t.Fatal(err)
			}
			answerSize := 8
			if excess {
				answerSize++
			}
			m.finish(claimed, Result{Text: strings.Repeat("x", answerSize), Provider: "test", Model: "model", InputTokens: 5, OutputTokens: 2}, nil)
			finished, err := m.Get(context.Background(), 1, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if finished.ExtraJSON != trace || finished.InputTokens != 5 || finished.OutputTokens != 2 {
				t.Fatal("retained progress or known usage changed")
			}
			var assistants int64
			if err := db.Model(&Message{}).Where("run_id=? AND role=?", run.ID, "assistant").Count(&assistants).Error; err != nil {
				t.Fatal(err)
			}
			if excess {
				if finished.Status != StatusFailed || finished.ErrorCode != "invalid_result" || finished.Text != "" || assistants != 0 {
					t.Fatal("combined answer/progress limit was bypassed")
				}
			} else if finished.Status != StatusSucceeded || len(finished.Text)+len(finished.ExtraJSON) != m.cfg.MaxResultBytes || assistants != 1 {
				t.Fatal("exact result budget boundary was rejected")
			}
		})
	}
}

func TestProgressDecoderRejectsTrailingDataAndUnknownFields(t *testing.T) {
	for _, extra := range []string{
		`[] {}`,
		`[] []`,
		`[] true`,
		`[] trailing`,
		`[{"name":"read","private":"secret"}]`,
		`[{"name":"read","summary":123}]`,
		`[{"name":"read"}]` + string([]byte{0xff}),
	} {
		if validExtra(extra) {
			t.Fatalf("invalid progress accepted: %q", extra)
		}
	}
	for _, extra := range []string{`[]`, " \n [] \t ", `[{"name":"read","status":"completed","summary":"safe","error":""}]`} {
		if !validExtra(extra) {
			t.Fatalf("valid progress rejected: %q", extra)
		}
	}
}

func TestFinishIncludesCanonicalEmptyProgressInResultBudget(t *testing.T) {
	for _, answerSize := range []int{6, 8} {
		t.Run(map[int]string{6: "boundary", 8: "excess"}[answerSize], func(t *testing.T) {
			db := fixture(t)
			m := manager(t, db, nil)
			m.cfg.MaxResultBytes = 8
			run := submit(t, m, request(t))
			claimed, err := m.claim(context.Background())
			if err != nil || claimed.ID != run.ID {
				t.Fatal("claim failed", err)
			}
			// An older record or executor can leave progress empty. Persisted
			// output still uses the public empty-array representation.
			if err := db.Model(&Run{}).Where("id=?", run.ID).Update("extra_json", "").Error; err != nil {
				t.Fatal(err)
			}
			m.finish(claimed, Result{Text: strings.Repeat("x", answerSize)}, nil)
			finished, err := m.Get(context.Background(), 1, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if finished.ExtraJSON != "[]" || len(finished.Text)+len(finished.ExtraJSON) > m.cfg.MaxResultBytes {
				t.Fatal("canonical progress bypassed persisted result limit")
			}
			var assistants int64
			if err := db.Model(&Message{}).Where("run_id=? AND role=?", run.ID, "assistant").Count(&assistants).Error; err != nil {
				t.Fatal(err)
			}
			if answerSize == 6 {
				if finished.Status != StatusSucceeded || assistants != 1 {
					t.Fatal("exact canonical result limit was rejected")
				}
			} else if finished.Status != StatusFailed || finished.ErrorCode != "invalid_result" || finished.Text != "" || assistants != 0 {
				t.Fatal("oversized result created an answer")
			}
		})
	}
}

func TestFinishBoundsRetainedProgressOnFailure(t *testing.T) {
	for _, executionErr := range []error{nil, errors.New("provider secret"), context.Canceled} {
		name := "invalid-result"
		if errors.Is(executionErr, context.Canceled) {
			name = "interrupted"
		} else if executionErr != nil {
			name = "failed"
		}
		t.Run(name, func(t *testing.T) {
			db := fixture(t)
			m := manager(t, db, nil)
			run := submit(t, m, request(t))
			claimed, err := m.claim(context.Background())
			if err != nil || claimed.ID != run.ID {
				t.Fatal("claim failed", err)
			}
			trace := `[{"name":"read","summary":"previous safe progress"}]`
			if err := m.UpdateProgress(context.Background(), run.ID, trace); err != nil {
				t.Fatal(err)
			}
			// A deployment can tighten limits while a previous worker's
			// progress is already stored in the database.
			m.cfg.MaxResultBytes = 2
			m.finish(claimed, Result{Text: "late answer", InputTokens: 7}, executionErr)
			finished, err := m.Get(context.Background(), 1, run.ID)
			if err != nil {
				t.Fatal(err)
			}
			if finished.Status == StatusSucceeded || finished.Text != "" || finished.ExtraJSON != "[]" || finished.InputTokens != 7 || strings.Contains(finished.Error, "provider secret") {
				t.Fatal("failed result retained oversized output or raw error")
			}
		})
	}
}

func TestResultBudgetAccommodatesEmptyProgress(t *testing.T) {
	db := fixture(t)
	cfg := testConfig()
	cfg.MaxResultBytes = 1
	if m, err := New(db, cfg, nil); !errors.Is(err, ErrInvalid) {
		if m != nil {
			m.Close()
		}
		t.Fatal("result budget cannot fit the public empty progress array", err)
	}
	cfg.MaxResultBytes = 2
	m, err := New(db, cfg, nil)
	if err != nil {
		t.Fatal("minimum usable result budget rejected", err)
	}
	t.Cleanup(m.Close)
	run := submit(t, m, request(t))
	canceled, err := m.Cancel(context.Background(), 1, run.ID)
	if err != nil || canceled.Status != StatusCanceled || canceled.ExtraJSON != "[]" {
		t.Fatal("minimal budget changed cancellation", err)
	}
}
