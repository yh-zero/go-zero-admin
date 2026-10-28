package agentjobs

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (m *Manager) sweep(ctx context.Context) error {
	return expireRuns(m.db.WithContext(ctx).Model(&Run{}), time.Now().UTC())
}

func expireRuns(query *gorm.DB, now time.Time) error {
	return query.Where("(status=? AND queue_expires_at<=?) OR (status=? AND (lease_until IS NULL OR lease_until<=? OR deadline_at IS NULL OR deadline_at<=?))", StatusQueued, now, StatusRunning, now, now).
		Updates(map[string]any{"status": StatusInterrupted, "error_code": "interrupted", "error": "任务超时或执行中断，不会自动重试。", "text": "", "finished_at": now, "updated_at": now, "lease_until": nil}).Error
}

func (m *Manager) loop() {
	defer close(m.done)
	ticker := time.NewTicker(m.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if m.root.Err() != nil {
			return
		}
		ctx, cancel := context.WithTimeout(m.root, 3*time.Second)
		err := m.sweep(ctx)
		var run Run
		if err == nil {
			run, err = m.claim(ctx)
		}
		cancel()
		if err == nil && run.ID != "" {
			m.process(run)
			continue
		}
		select {
		case <-m.root.Done():
			return
		case <-m.wake:
		case <-ticker.C:
		}
	}
}

func (m *Manager) claim(ctx context.Context) (Run, error) {
	var run Run
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Where("status=? AND cancel_requested=? AND queue_expires_at>?", StatusQueued, false, now).Order("created_at ASC,id ASC").Take(&run).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		lease := now.Add(m.cfg.LeaseDuration)
		deadline := now.Add(m.cfg.RunTimeout)
		updated := tx.Model(&Run{}).Where("id=? AND status=? AND cancel_requested=?", run.ID, StatusQueued, false).Updates(map[string]any{"status": StatusRunning, "lease_owner": m.id, "lease_until": lease, "deadline_at": deadline, "started_at": now, "updated_at": now})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			run = Run{}
			return nil
		}
		run.Status = StatusRunning
		run.LeaseOwner = m.id
		run.LeaseUntil = &lease
		run.DeadlineAt = &deadline
		run.StartedAt = &now
		run.UpdatedAt = now
		return nil
	})
	return run, err
}

func (m *Manager) history(ctx context.Context, run Run) ([]Message, error) {
	var messages []Message
	err := m.db.WithContext(ctx).Model(&Message{}).Select("sys_ai_messages.*").Joins("JOIN sys_ai_runs previous ON previous.id=sys_ai_messages.run_id").Where("sys_ai_messages.owner_id=? AND sys_ai_messages.conversation_id=? AND previous.status=? AND previous.id<>? AND sys_ai_messages.role IN ?", run.OwnerID, run.ConversationID, StatusSucceeded, run.ID, []string{"user", "assistant"}).Order("previous.sequence DESC, CASE WHEN sys_ai_messages.role='assistant' THEN 1 ELSE 0 END DESC").Limit(12).Find(&messages).Error
	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}
	return messages, err
}

func (m *Manager) process(run Run) {
	ctx, cancel := context.WithDeadline(m.root, *run.DeadlineAt)
	m.mu.Lock()
	m.currentID = run.ID
	m.currentCancel = cancel
	m.mu.Unlock()
	defer func() { cancel(); m.mu.Lock(); m.currentID = ""; m.currentCancel = nil; m.mu.Unlock() }()
	heartbeatDone := make(chan struct{})
	go m.heartbeat(ctx, cancel, run.ID, heartbeatDone)
	history, err := m.history(ctx, run)
	if err == nil {
		err = m.checkLease(ctx, run.ID)
	}
	var result Result
	if err == nil {
		// One invocation per worker, even for an executor that violates its
		// cancellation contract. It may outlive shutdown, but blocks this worker
		// permanently rather than spawning unbounded abandoned provider calls.
		type executionResult struct {
			result Result
			err    error
		}
		completed := make(chan executionResult, 1)
		go func() {
			value, callErr := invokeExecutor(m.execute, ctx, Job{Run: run, History: history})
			completed <- executionResult{value, callErr}
		}()
		select {
		case value := <-completed:
			result, err = value.result, value.err
		case <-ctx.Done():
			err = ctx.Err()
			m.finish(run, result, err)
			cancel()
			<-heartbeatDone
			// Do not start another provider call until this one actually exits.
			select {
			case value := <-completed:
				// Keep known usage even when cancellation won the result race.
				// finish's terminal branch cannot restore answer/status/trace.
				m.finish(run, value.result, err)
			case <-m.root.Done():
			}
			return
		}
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	m.finish(run, result, err)
	cancel()
	<-heartbeatDone
}

func (m *Manager) checkLease(ctx context.Context, runID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	var active int64
	if err := m.db.WithContext(ctx).Model(&Run{}).Where("id=? AND status=? AND lease_owner=? AND cancel_requested=? AND lease_until>? AND deadline_at>?", runID, StatusRunning, m.id, false, now, now).Count(&active).Error; err != nil {
		return err
	}
	if active != 1 {
		return ErrLeaseLost
	}
	return ctx.Err()
}

func invokeExecutor(execute Executor, ctx context.Context, job Job) (result Result, err error) {
	defer func() {
		if recover() != nil {
			result = Result{}
			err = errors.New("agent executor panicked")
		}
	}()
	return execute(ctx, job)
}

func (m *Manager) heartbeat(ctx context.Context, cancel context.CancelFunc, runID string, done chan<- struct{}) {
	defer close(done)
	ticker := time.NewTicker(m.cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now().UTC()
			queryCtx, queryCancel := context.WithTimeout(ctx, 3*time.Second)
			result := m.db.WithContext(queryCtx).Model(&Run{}).Where("id=? AND lease_owner=? AND status=? AND cancel_requested=? AND deadline_at>? AND lease_until>?", runID, m.id, StatusRunning, false, now, now).Updates(map[string]any{"lease_until": now.Add(m.cfg.LeaseDuration), "updated_at": now})
			queryCancel()
			if result.Error != nil || result.RowsAffected != 1 {
				cancel()
				return
			}
		}
	}
}

func (m *Manager) validResult(result Result) bool {
	if !utf8.ValidString(result.Text) || !utf8.ValidString(result.Provider) || !utf8.ValidString(result.Model) || len(result.Text)+len(result.ExtraJSON) > m.cfg.MaxResultBytes || len(result.Provider) > 64 || len(result.Model) > 128 || result.InputTokens < 0 || result.OutputTokens < 0 {
		return false
	}
	return validExtra(result.ExtraJSON)
}

func validExtra(extra string) bool {
	extra = strings.TrimSpace(extra)
	if len(extra) == 0 || extra[0] != '[' {
		return false
	}
	if !utf8.ValidString(extra) {
		return false
	}
	var tools []struct {
		Name    string `json:"name"`
		Status  string `json:"status"`
		Error   string `json:"error"`
		Summary string `json:"summary"`
	}
	decoder := json.NewDecoder(strings.NewReader(extra))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&tools) != nil || len(tools) > 64 {
		return false
	}
	if _, err := decoder.Token(); err != io.EOF {
		return false
	}
	for _, tool := range tools {
		if len(tool.Name) > 64 || len(tool.Status) > 32 || len(tool.Error) > 256 || len(tool.Summary) > 8192 {
			return false
		}
	}
	return true
}

// UpdateProgress cannot revive a canceled, expired, or another process's run.
// A locked read also makes repeated identical progress safe with MySQL's changed
// row count semantics (an identical update may report zero changed rows).
func (m *Manager) UpdateProgress(ctx context.Context, runID string, extraJSON string) error {
	if !validID(runID) || len(extraJSON) > m.cfg.MaxResultBytes || !validExtra(extraJSON) {
		return ErrInvalid
	}
	return m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var run Run
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", runID).Take(&run).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrLeaseLost
			}
			return err
		}
		now := time.Now().UTC()
		if run.Status != StatusRunning || run.LeaseOwner != m.id || run.CancelRequested || !run.hasActiveLease(now) {
			return ErrLeaseLost
		}
		return tx.Model(&Run{}).Where("id=? AND status=? AND lease_owner=? AND cancel_requested=?", runID, StatusRunning, m.id, false).Updates(map[string]any{"extra_json": extraJSON, "updated_at": now}).Error
	})
}

func (m *Manager) finish(run Run, result Result, executionErr error) {
	status, errorCode, errorText := StatusSucceeded, "", ""
	if executionErr != nil {
		status, errorCode, errorText = StatusFailed, "execution_failed", "AI任务执行失败。"
		if errors.Is(executionErr, context.Canceled) || errors.Is(executionErr, context.DeadlineExceeded) {
			status, errorCode, errorText = StatusInterrupted, "interrupted", "任务超时或执行中断，不会自动重试。"
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current Run
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=?", run.ID).Take(&current).Error; err != nil {
			return err
		}
		if current.LeaseOwner != m.id {
			return nil
		}
		if current.Status != StatusRunning {
			if current.Status == StatusCanceled || current.Status == StatusInterrupted {
				return preserveKnownUsage(tx, current, result, m.id)
			}
			return nil
		}
		if current.CancelRequested {
			return nil
		}
		if result.ExtraJSON == "" {
			result.ExtraJSON = current.ExtraJSON
		}
		if result.ExtraJSON == "" {
			result.ExtraJSON = "[]"
		}
		// Validate the complete persisted result, including progress retained
		// when an executor leaves ExtraJSON empty and its canonical empty array.
		if executionErr == nil && !m.validResult(result) {
			status, errorCode, errorText = StatusFailed, "invalid_result", "AI返回结果无效或超出限制。"
		}
		now := time.Now().UTC()
		if !current.hasActiveLease(now) {
			status, errorCode, errorText = StatusInterrupted, "interrupted", "任务超时或执行中断，不会自动重试。"
		}
		if status != StatusSucceeded {
			result.Text = ""
			result.ExtraJSON = current.ExtraJSON
			// A new deployment can tighten limits below already stored progress.
			if len(result.ExtraJSON) > m.cfg.MaxResultBytes || !validExtra(result.ExtraJSON) {
				result.ExtraJSON = "[]"
			}
			if len(result.Provider) > 64 || !utf8.ValidString(result.Provider) {
				result.Provider = ""
			}
			if len(result.Model) > 128 || !utf8.ValidString(result.Model) {
				result.Model = ""
			}
			if result.InputTokens < 0 {
				result.InputTokens = 0
			}
			if result.OutputTokens < 0 {
				result.OutputTokens = 0
			}
		}
		updates := map[string]any{"status": status, "text": result.Text, "provider": result.Provider, "model": result.Model, "input_tokens": result.InputTokens, "output_tokens": result.OutputTokens, "extra_json": result.ExtraJSON, "error_code": errorCode, "error": errorText, "lease_until": nil, "finished_at": now, "updated_at": now}
		changed := tx.Model(&Run{}).Where("id=? AND status=? AND lease_owner=? AND cancel_requested=?", run.ID, StatusRunning, m.id, false).Updates(updates)
		if changed.Error != nil {
			return changed.Error
		}
		if changed.RowsAffected != 1 {
			return nil
		}
		if status == StatusSucceeded {
			id, err := newID()
			if err != nil {
				return err
			}
			if err := tx.Create(&Message{ID: id, OwnerID: run.OwnerID, ConversationID: run.ConversationID, RunID: run.ID, Role: "assistant", Content: result.Text, CreatedAt: now}).Error; err != nil {
				return err
			}
		}
		return tx.Model(&Conversation{}).Where("id=? AND owner_id=?", run.ConversationID, run.OwnerID).Update("updated_at", now).Error
	})
}

func preserveKnownUsage(tx *gorm.DB, current Run, result Result, leaseOwner string) error {
	updates := map[string]any{}
	if result.InputTokens > current.InputTokens {
		updates["input_tokens"] = result.InputTokens
	}
	if result.OutputTokens > current.OutputTokens {
		updates["output_tokens"] = result.OutputTokens
	}
	if current.Provider == "" && result.Provider != "" && len(result.Provider) <= 64 && utf8.ValidString(result.Provider) {
		updates["provider"] = result.Provider
	}
	if current.Model == "" && result.Model != "" && len(result.Model) <= 128 && utf8.ValidString(result.Model) {
		updates["model"] = result.Model
	}
	if len(updates) == 0 {
		return nil
	}
	updates["updated_at"] = time.Now().UTC()
	return tx.Model(&Run{}).Where("id=? AND status=? AND lease_owner=?", current.ID, current.Status, leaseOwner).Updates(updates).Error
}
