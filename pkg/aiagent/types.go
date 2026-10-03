package aiagent

import (
	"context"
	"encoding/json"
	"errors"
)

var (
	ErrDisabled       = errors.New("AI agent is disabled")
	ErrInvalidRequest = errors.New("invalid AI agent request")
	ErrUnknownTool    = errors.New("AI agent requested an unavailable tool")
	ErrToolArguments  = errors.New("invalid AI agent tool arguments")
	ErrToolFailed     = errors.New("AI agent tool execution failed")
	ErrModelFailed    = errors.New("AI model request failed")
	ErrBudgetExceeded = errors.New("AI agent budget exceeded")
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Tool callbacks are trusted server-side capabilities, never request fields.
// Call must validate its typed arguments and recheck the actor's current session,
// Casbin policy, and applicable data scope each time. Only an authorized subset
// of the built-in definitions should be injected into a request.
type Tool struct {
	ToolDefinition
	Call func(context.Context, json.RawMessage) (ToolOutput, error)
}

// Content must be bounded JSON business data, without passwords, tokens,
// signed URLs, internal credentials or raw audit request bodies. Count is safe
// execution metadata; Content is passed to the model but never copied to events.
type ToolOutput struct {
	Content string
	Summary string
	Count   int
}

type Request struct {
	History []Message
	Tools   []Tool
}

type RunInput = Request

type Usage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

type ToolSummary struct {
	Name       string `json:"name"`
	Summary    string `json:"summary"`
	Success    bool   `json:"success"`
	Count      int    `json:"count"`
	DurationMS int64  `json:"durationMs"`
}

// Event intentionally has no model reasoning, arguments, tool contents or
// provider error text. Callbacks are synchronous, serialized and may return an
// error to abort the run when persisting its trace fails.
type Event struct {
	Kind  string      `json:"kind"`
	Step  int         `json:"step"`
	Tool  ToolSummary `json:"tool"`
	Usage Usage       `json:"usage"`
}

type EventHandler func(context.Context, Event) error

type Result struct {
	FinalAnswer   string        `json:"finalAnswer"`
	ToolSummaries []ToolSummary `json:"toolSummaries"`
	Usage         Usage         `json:"usage"`
	Provider      string        `json:"provider"`
	Model         string        `json:"model"`
}

type Runner interface {
	Run(context.Context, Request, EventHandler) (Result, error)
}
