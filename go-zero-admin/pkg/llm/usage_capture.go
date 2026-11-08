package llm

import (
	"context"
	"sync"

	"github.com/cloudwego/eino/schema"
)

type usageCaptureKey struct{}

// UsageCapture 只保留每次HTTP尝试的数字用量；不保存或提供模型正文。
type UsageCapture struct {
	mu       sync.Mutex
	attempts map[string]*capturedUsage
}

type capturedUsage struct {
	prompt, completion, total *int64
	invalid, incomplete       bool
}

func withUsageCapture(ctx context.Context) (context.Context, *UsageCapture) {
	capture := &UsageCapture{attempts: make(map[string]*capturedUsage)}
	return context.WithValue(ctx, usageCaptureKey{}, capture), capture
}

func (c *UsageCapture) register(id string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attempts[id] = &capturedUsage{}
}

func (c *UsageCapture) observe(id string, usage *providerUsage, valid bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.attempts[id]
	if entry == nil {
		return
	}
	if !valid {
		entry.invalid = true
		return
	}
	keep := func(old **int64, next *int64) {
		if next == nil {
			return
		}
		if *old != nil && **old > *next {
			entry.invalid = true
			return
		}
		value := *next
		*old = &value
	}
	keep(&entry.prompt, usage.Prompt)
	keep(&entry.completion, usage.Completion)
	keep(&entry.total, usage.Total)
}

func (c *UsageCapture) finish(id string, incomplete bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry := c.attempts[id]; entry != nil {
		entry.incomplete = entry.incomplete || incomplete
	}
}

// Snapshot 跨HTTP尝试合计已知值，同一尝试的重复帧不重复相加；缺失不推算为零费用。
func (c *UsageCapture) Snapshot() (*schema.TokenUsage, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var prompt, completion, total int64
	known, complete := false, len(c.attempts) > 0
	maxInt := int64(^uint(0) >> 1)
	for _, entry := range c.attempts {
		for _, field := range []struct {
			target *int64
			value  *int64
		}{{&prompt, entry.prompt}, {&completion, entry.completion}, {&total, entry.total}} {
			if field.value != nil {
				if *field.value > maxInt-*field.target {
					return nil, false
				}
				*field.target += *field.value
				known = true
			}
		}
		complete = complete && !entry.invalid && !entry.incomplete && entry.prompt != nil && entry.completion != nil && entry.total != nil && *entry.prompt+*entry.completion == *entry.total
	}
	if !known {
		return nil, false
	}
	return &schema.TokenUsage{PromptTokens: int(prompt), CompletionTokens: int(completion), TotalTokens: int(total)}, complete
}
