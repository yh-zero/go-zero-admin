package agentjobs

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gofrs/uuid/v5"
	"go-zero-admin/pkg/audit"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Manager struct {
	db            *gorm.DB
	cfg           Config
	execute       Executor
	id            string
	root          context.Context
	cancel        context.CancelFunc
	wake          chan struct{}
	done          chan struct{}
	mu            sync.Mutex
	started       bool
	closed        bool
	currentID     string
	currentCancel context.CancelFunc
}

func New(db *gorm.DB, cfg Config, execute Executor) (*Manager, error) {
	if db == nil {
		return nil, ErrInvalid
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.RunTimeout == 0 {
		cfg.RunTimeout = 2 * time.Minute
	}
	if cfg.QueueTimeout == 0 {
		cfg.QueueTimeout = 5 * time.Minute
	}
	if cfg.LeaseDuration == 0 {
		cfg.LeaseDuration = 15 * time.Second
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 5 * time.Second
	}
	if cfg.UserActiveLimit == 0 {
		cfg.UserActiveLimit = 2
	}
	if cfg.MaxQuestionBytes == 0 {
		cfg.MaxQuestionBytes = 16 * 1024
	}
	if cfg.MaxResultBytes == 0 {
		cfg.MaxResultBytes = 512 * 1024
	}
	// Even a result without tool progress persists the public empty array [].
	if cfg.PollInterval <= 0 || cfg.RunTimeout <= 0 || cfg.QueueTimeout <= 0 || cfg.HeartbeatInterval <= 0 || cfg.LeaseDuration < 2*cfg.HeartbeatInterval || cfg.UserActiveLimit < 1 || cfg.UserActiveLimit > 32 || cfg.MaxQuestionBytes < 1 || cfg.MaxResultBytes < 2 {
		return nil, ErrInvalid
	}
	id, err := newID()
	if err != nil {
		return nil, err
	}
	root, cancel := context.WithCancel(context.Background())
	return &Manager{db: db, cfg: cfg, execute: execute, id: id, root: root, cancel: cancel, wake: make(chan struct{}, 1), done: make(chan struct{})}, nil
}

// Start verifies the installed schema. It never creates or changes schema.
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return ErrClosed
	}
	if m.started {
		return nil
	}
	if !m.cfg.Enabled {
		return ErrDisabled
	}
	if m.execute == nil {
		return ErrInvalid
	}
	ctx, cancel := context.WithTimeout(m.root, 5*time.Second)
	defer cancel()
	for _, model := range []any{&Conversation{}, &Message{}, &Run{}} {
		if err := m.db.WithContext(ctx).Limit(1).Find(model).Error; err != nil {
			return fmt.Errorf("agent schema unavailable; run migrations first: %w", err)
		}
	}
	if err := m.sweep(ctx); err != nil {
		return err
	}
	m.started = true
	go m.loop()
	return nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	if !m.closed {
		m.closed = true
		m.cancel()
	}
	started := m.started
	m.mu.Unlock()
	if started {
		<-m.done
	}
}

func (m *Manager) notify() {
	select {
	case m.wake <- struct{}{}:
	default:
	}
}

func newID() (string, error) { id, err := uuid.NewV4(); return id.String(), err }

func validID(value string) bool {
	id, err := uuid.FromString(value)
	return err == nil && id != uuid.Nil && id.String() == value
}

// A current locking read of the existing user row serializes submissions even
// when two processes create different conversations for the same owner.
func lockOwner(tx *gorm.DB, ownerID int64) error {
	var owner struct {
		ID     int64
		Enable int64
	}
	if err := tx.Table("sys_users").Clauses(clause.Locking{Strength: "UPDATE"}).Select("id,enable").Where("id=? AND deleted_at IS NULL", ownerID).Take(&owner).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	if owner.Enable != 1 {
		return ErrNotFound
	}
	return nil
}

func (m *Manager) Submit(ctx context.Context, input SubmitInput) (Run, error) {
	m.mu.Lock()
	closed := m.closed
	m.mu.Unlock()
	if closed {
		return Run{}, ErrClosed
	}
	if !m.cfg.Enabled {
		return Run{}, ErrDisabled
	}
	if input.OwnerID <= 0 || input.AuthorityID <= 0 || input.SessionVersion < 0 || !validID(input.RequestID) || (input.SessionID != "" && !validID(input.SessionID)) || (input.ConversationID != "" && !validID(input.ConversationID)) || !utf8.ValidString(input.Question) || len(input.Question) > m.cfg.MaxQuestionBytes || strings.TrimSpace(input.Question) == "" {
		return Run{}, ErrInvalid
	}
	body, _ := json.Marshal(struct{ ConversationID, Question string }{input.ConversationID, input.Question})
	digest := sha256.Sum256(body)
	requestHash := hex.EncodeToString(digest[:])
	var result Run
	var transactionOptions []*sql.TxOptions
	if m.db.Dialector.Name() == "mysql" {
		// Owner locks serialize that user's decisions. READ COMMITTED keeps
		// absent request/active index reads from gap-locking unrelated owners.
		transactionOptions = append(transactionOptions, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	}
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockOwner(tx, input.OwnerID); err != nil {
			return err
		}
		if err := expireRuns(tx.Model(&Run{}).Where("owner_id=?", input.OwnerID), time.Now().UTC()); err != nil {
			return err
		}
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("owner_id=? AND request_id=?", input.OwnerID, input.RequestID).Take(&result).Error
		if err == nil {
			if result.RequestHash != requestHash {
				return ErrConflict
			}
			result.Reused = true
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		// Locking reads avoid an older REPEATABLE READ snapshot after waiting for
		// another process's user lock.
		var active []Run
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id,conversation_id").Where("owner_id=? AND status IN ?", input.OwnerID, []string{StatusQueued, StatusRunning}).Find(&active).Error; err != nil {
			return err
		}
		if len(active) >= m.cfg.UserActiveLimit {
			return ErrBusy
		}
		var conversation Conversation
		now := time.Now().UTC()
		if input.ConversationID == "" {
			id, err := newID()
			if err != nil {
				return err
			}
			title := []rune(strings.TrimSpace(input.Question))
			if len(title) > 80 {
				title = title[:80]
			}
			conversation = Conversation{ID: id, OwnerID: input.OwnerID, Title: string(title), CreatedAt: now, UpdatedAt: now}
			if err := tx.Create(&conversation).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=?", input.ConversationID, input.OwnerID).Take(&conversation).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return ErrNotFound
				}
				return err
			}
			for _, run := range active {
				if run.ConversationID == conversation.ID {
					return ErrBusy
				}
			}
		}
		id, err := newID()
		if err != nil {
			return err
		}
		result = Run{ID: id, OwnerID: input.OwnerID, AuthorityID: input.AuthorityID, SessionID: input.SessionID, SessionVersion: input.SessionVersion, RequestID: input.RequestID, RequestHash: requestHash, ConversationID: conversation.ID, Sequence: conversation.NextSequence + 1, Question: input.Question, Status: StatusQueued, ExtraJSON: "[]", QueueExpiresAt: now.Add(m.cfg.QueueTimeout), CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&result).Error; err != nil {
			return err
		}
		messageID, err := newID()
		if err != nil {
			return err
		}
		if err := tx.Create(&Message{ID: messageID, OwnerID: input.OwnerID, ConversationID: conversation.ID, RunID: id, Role: "user", Content: input.Question, CreatedAt: now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&Conversation{}).Where("id=?", conversation.ID).Updates(map[string]any{"updated_at": now, "next_sequence": result.Sequence}).Error; err != nil {
			return err
		}
		return audit.Record(ctx, tx, audit.Event{ActorID: input.OwnerID, AuthorityID: input.AuthorityID, Module: "ai-agent", Action: "submitRun", Object: id})
	}, transactionOptions...)
	if err == nil {
		m.notify()
	}
	return result, err
}

func (m *Manager) Get(ctx context.Context, ownerID int64, runID string) (Run, error) {
	if ownerID <= 0 || !validID(runID) {
		return Run{}, ErrInvalid
	}
	var run Run
	err := m.db.WithContext(ctx).Where("id=? AND owner_id=?", runID, ownerID).Take(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = ErrNotFound
	}
	if err == nil {
		now := time.Now().UTC()
		expired := run.Status == StatusQueued && !run.QueueExpiresAt.After(now)
		expired = expired || (run.Status == StatusRunning && !run.hasActiveLease(now))
		if expired {
			if err := expireRuns(m.db.WithContext(ctx).Model(&Run{}).Where("id=? AND owner_id=?", runID, ownerID), now); err != nil {
				return Run{}, err
			}
			err = m.db.WithContext(ctx).Where("id=? AND owner_id=?", runID, ownerID).Take(&run).Error
		}
	}
	return run, err
}

func (m *Manager) Cancel(ctx context.Context, ownerID int64, runID string) (Run, error) {
	if ownerID <= 0 || !validID(runID) {
		return Run{}, ErrInvalid
	}
	var run Run
	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := lockOwner(tx, ownerID); err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND owner_id=?", runID, ownerID).Take(&run).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrNotFound
			}
			return err
		}
		if IsTerminal(run.Status) {
			return nil
		}
		now := time.Now().UTC()
		if err := tx.Model(&Run{}).Where("id=? AND status IN ?", runID, []string{StatusQueued, StatusRunning}).Updates(map[string]any{"status": StatusCanceled, "cancel_requested": true, "finished_at": now, "updated_at": now, "lease_until": nil}).Error; err != nil {
			return err
		}
		run.Status = StatusCanceled
		run.CancelRequested = true
		run.FinishedAt = &now
		run.UpdatedAt = now
		run.LeaseUntil = nil
		return audit.Record(ctx, tx, audit.Event{ActorID: ownerID, AuthorityID: run.AuthorityID, Module: "ai-agent", Action: "cancelRun", Object: runID})
	})
	if err == nil {
		m.mu.Lock()
		if m.currentID == runID && m.currentCancel != nil {
			m.currentCancel()
		}
		m.mu.Unlock()
		m.notify()
	}
	return run, err
}

func pagination(page, size int) (int, error) {
	if page < 1 || size < 1 || size > 100 || page > 1000000 {
		return 0, ErrInvalid
	}
	return (page - 1) * size, nil
}

func (m *Manager) ListConversations(ctx context.Context, ownerID int64, page, size int) ([]Conversation, int64, error) {
	offset, err := pagination(page, size)
	if err != nil || ownerID <= 0 {
		return nil, 0, ErrInvalid
	}
	query := m.db.WithContext(ctx).Model(&Conversation{}).Where("owner_id=?", ownerID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var values []Conversation
	err = query.Order("updated_at DESC,id DESC").Offset(offset).Limit(size).Find(&values).Error
	return values, total, err
}

func (m *Manager) ListMessages(ctx context.Context, ownerID int64, conversationID string, page, size int) ([]Message, int64, error) {
	offset, err := pagination(page, size)
	if err != nil || ownerID <= 0 || !validID(conversationID) {
		return nil, 0, ErrInvalid
	}
	var conversation Conversation
	if err := m.db.WithContext(ctx).Where("id=? AND owner_id=?", conversationID, ownerID).Take(&conversation).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrNotFound
		}
		return nil, 0, err
	}
	query := m.db.WithContext(ctx).Model(&Message{}).Joins("JOIN sys_ai_runs ordered_run ON ordered_run.id=sys_ai_messages.run_id").Where("sys_ai_messages.owner_id=? AND sys_ai_messages.conversation_id=?", ownerID, conversationID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var values []Message
	err = query.Select("sys_ai_messages.*").Order("ordered_run.sequence DESC, CASE WHEN sys_ai_messages.role='assistant' THEN 1 ELSE 0 END DESC").Offset(offset).Limit(size).Find(&values).Error
	return values, total, err
}
