// Package agentjobs stores and schedules bounded asynchronous agent runs.
// Provider calls belong to Executor; this package never retries a claimed run.
package agentjobs

import (
	"context"
	"errors"
	"time"
)

const (
	StatusQueued      = "queued"
	StatusRunning     = "running"
	StatusSucceeded   = "succeeded"
	StatusFailed      = "failed"
	StatusCanceled    = "cancelled"
	StatusInterrupted = "interrupted"
)

var (
	ErrInvalid   = errors.New("invalid agent request")
	ErrNotFound  = errors.New("agent resource not found")
	ErrConflict  = errors.New("agent request ID was reused with different content")
	ErrBusy      = errors.New("agent concurrency limit reached")
	ErrClosed    = errors.New("agent scheduler is closed")
	ErrDisabled  = errors.New("agent execution is disabled")
	ErrLeaseLost = errors.New("agent execution lease is no longer active")
)

type Config struct {
	Enabled           bool
	PollInterval      time.Duration
	RunTimeout        time.Duration
	QueueTimeout      time.Duration
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	UserActiveLimit   int
	MaxQuestionBytes  int
	MaxResultBytes    int
}

type SubmitInput struct {
	OwnerID        int64
	AuthorityID    int64
	SessionID      string
	SessionVersion int64
	RequestID      string
	Question       string
	ConversationID string
}

type Job struct {
	Run     Run
	History []Message
}

type Result struct {
	Text         string
	Provider     string
	Model        string
	InputTokens  int64
	OutputTokens int64
	ExtraJSON    string
}

// Executor must honor ctx cancellation and must not retry ambiguous requests.
type Executor func(ctx context.Context, job Job) (Result, error)

type Conversation struct {
	ID           string    `gorm:"size:36;primaryKey"`
	OwnerID      int64     `gorm:"not null;index:idx_agent_conversation_owner,priority:1"`
	Title        string    `gorm:"size:160;not null"`
	NextSequence int64     `gorm:"not null;default:0"`
	CreatedAt    time.Time `gorm:"not null"`
	UpdatedAt    time.Time `gorm:"not null;index:idx_agent_conversation_owner,priority:2"`
}

func (Conversation) TableName() string { return "sys_ai_conversations" }

type Message struct {
	ID             string    `gorm:"size:36;primaryKey"`
	OwnerID        int64     `gorm:"not null"`
	ConversationID string    `gorm:"size:36;not null;index:idx_agent_message_history,priority:1"`
	RunID          string    `gorm:"size:36;not null;uniqueIndex:uk_agent_message_run_role,priority:1"`
	Role           string    `gorm:"size:16;not null;uniqueIndex:uk_agent_message_run_role,priority:2"`
	Content        string    `gorm:"type:longtext;not null"`
	CreatedAt      time.Time `gorm:"not null;index:idx_agent_message_history,priority:2"`
}

func (Message) TableName() string { return "sys_ai_messages" }

type Run struct {
	ID              string     `gorm:"size:36;primaryKey"`
	OwnerID         int64      `gorm:"not null;uniqueIndex:uk_agent_run_request,priority:1;index:idx_agent_run_owner_status,priority:1"`
	AuthorityID     int64      `gorm:"not null"`
	SessionID       string     `gorm:"size:36;not null;default:''"`
	SessionVersion  int64      `gorm:"not null"`
	RequestID       string     `gorm:"size:36;not null;uniqueIndex:uk_agent_run_request,priority:2"`
	RequestHash     string     `gorm:"type:char(64);size:64;not null"`
	ConversationID  string     `gorm:"size:36;not null;index:idx_agent_run_conversation_status,priority:1;uniqueIndex:uk_agent_run_sequence,priority:1"`
	Sequence        int64      `gorm:"not null;uniqueIndex:uk_agent_run_sequence,priority:2"`
	Question        string     `gorm:"type:longtext;not null"`
	Status          string     `gorm:"size:16;not null;index:idx_agent_run_owner_status,priority:2;index:idx_agent_run_conversation_status,priority:2;index:idx_agent_run_claim,priority:1"`
	CancelRequested bool       `gorm:"not null;default:false"`
	Text            string     `gorm:"type:longtext;not null"`
	Provider        string     `gorm:"size:64;not null;default:''"`
	Model           string     `gorm:"size:128;not null;default:''"`
	InputTokens     int64      `gorm:"not null;default:0"`
	OutputTokens    int64      `gorm:"not null;default:0"`
	ExtraJSON       string     `gorm:"type:longtext;not null"`
	ErrorCode       string     `gorm:"size:32;not null;default:''"`
	Error           string     `gorm:"size:256;not null;default:''"`
	LeaseOwner      string     `gorm:"size:36;not null;default:''"`
	LeaseUntil      *time.Time `gorm:"index:idx_agent_run_lease"`
	QueueExpiresAt  time.Time  `gorm:"not null;index:idx_agent_run_claim,priority:2"`
	DeadlineAt      *time.Time
	StartedAt       *time.Time
	FinishedAt      *time.Time
	CreatedAt       time.Time `gorm:"not null"`
	UpdatedAt       time.Time `gorm:"not null"`
	Reused          bool      `gorm:"-"`
}

func (Run) TableName() string { return "sys_ai_runs" }

func (r Run) hasActiveLease(now time.Time) bool {
	return r.LeaseUntil != nil && r.LeaseUntil.After(now) && r.DeadlineAt != nil && r.DeadlineAt.After(now)
}

func IsTerminal(status string) bool {
	return status == StatusSucceeded || status == StatusFailed || status == StatusCanceled || status == StatusInterrupted
}
