package agentlogic

import (
	"context"
	"encoding/json"
	"strings"
	"sync"

	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/aiagent"
)

// Query details stay private until execution and the final permission check
// succeed. They use the existing answer/history contract, never polling events.
type resultDisplay struct {
	mu      sync.Mutex
	blocks  []string
	seen    map[string]bool
	called  map[string]bool
	bytes   int
	limit   int
	omitted bool
}

func collectResultDisplay(tools []aiagent.Tool, maxAnswerBytes int) ([]aiagent.Tool, *resultDisplay) {
	if maxAnswerBytes <= 0 {
		maxAnswerBytes = 8192
	}
	display := &resultDisplay{seen: map[string]bool{}, called: map[string]bool{}, limit: min(4096, maxAnswerBytes/2)}
	wrapped := append([]aiagent.Tool(nil), tools...)
	for i := range wrapped {
		original := wrapped[i]
		wrapped[i].Call = func(ctx context.Context, args json.RawMessage) (aiagent.ToolOutput, error) {
			output, err := original.Call(ctx, args)
			if err == nil {
				display.capture(original.Name, output.Content)
			}
			return output, err
		}
	}
	return wrapped, display
}

func (d *resultDisplay) capture(name, content string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.called[name] = true
	// Reserve space for the explanation if several queries fill the display.
	remaining := d.limit - d.bytes - 128
	block := agenttools.FormatResult(name, content, remaining)
	if block == "" || d.seen[block] {
		if remaining < 256 {
			d.omitted = true
		}
		return
	}
	d.seen[block] = true
	d.blocks = append(d.blocks, block)
	d.bytes += len(block) + 2
}

func (d *resultDisplay) used(name string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.called[name]
}

func (d *resultDisplay) answer(answer string, maxBytes int) string {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.blocks) == 0 {
		return answer
	}
	if maxBytes <= 0 {
		maxBytes = 8192
	}
	details := "### 查询明细\n\n" + strings.Join(d.blocks, "\n\n")
	if d.omitted {
		details += "\n\n部分查询明细因展示长度限制未列出。"
	}
	if len(answer)+len(details)+2 <= maxBytes {
		return strings.TrimSpace(answer) + "\n\n" + details
	}
	// Preserve complete table rows and UTF-8. An oversized model overview must
	// not turn a successful bounded query into an oversized persisted answer.
	return "AI概述较长，已省略。以下为工具实际返回的查询明细。\n\n" + details
}
