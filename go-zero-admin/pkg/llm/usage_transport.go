package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go-zero-admin/pkg/xerr"
)

type usageStateKey struct{}
type providerUsage struct {
	Prompt     *int64 `json:"prompt_tokens"`
	Completion *int64 `json:"completion_tokens"`
	Total      *int64 `json:"total_tokens"`
}
type usageEnvelope struct {
	ID    string         `json:"id"`
	Usage *providerUsage `json:"usage"`
}

type usageCall struct {
	mu                                                        sync.Mutex
	hook                                                      UsageHook
	attempt                                                   UsageAttempt
	ctx                                                       context.Context
	observation                                               UsageObservation
	started, stream, sawDone, truncated, autoFinish, finished bool
	transportCount                                            int
	buffer                                                    []byte
	finishErr                                                 error
	cancel                                                    context.CancelFunc
	apiKey                                                    string
	metadata                                                  *UsageCapture
	last                                                      *usageCall
}

func beginUsage(ctx context.Context, hook UsageHook, provider, model, operation string, batch, items int64, apiKey string) (context.Context, *usageCall, error) {
	identity, ok := UsageIdentityFrom(ctx)
	var cancel context.CancelFunc
	if hook != nil {
		if !ok {
			return ctx, nil, xerr.NewErrCodeMsg(503, "模型调用缺少计量身份")
		}
		ctx, cancel = context.WithTimeout(ctx, 90*time.Second)
	}
	call := &usageCall{hook: hook, ctx: ctx, cancel: cancel, apiKey: apiKey, attempt: UsageAttempt{UsageIdentity: identity, AttemptID: uuid.NewString(), Operation: operation, Provider: provider, Model: model, BatchIndex: batch, InputItems: items}, observation: UsageObservation{UsageState: "unknown"}}
	if hook != nil {
		if err := hook.Begin(ctx, call.attempt); err != nil {
			cancel()
			return ctx, nil, err
		}
	}
	call.metadata, _ = ctx.Value(usageCaptureKey{}).(*UsageCapture)
	call.metadata.register(call.attempt.AttemptID)
	return context.WithValue(ctx, usageStateKey{}, call), call, nil
}

type usageTransport struct{ base http.RoundTripper }

func (t *usageTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	call, _ := req.Context().Value(usageStateKey{}).(*usageCall)
	if call == nil {
		return base.RoundTrip(req)
	}
	call.mu.Lock()
	call.transportCount++
	count := call.transportCount
	call.mu.Unlock()
	if count > 1 {
		// HTTP重定向亦是新请求，不能沿用前一跳的额度或重复覆盖同一账本。
		ctx, next, err := beginUsage(req.Context(), call.hook, call.attempt.Provider, call.attempt.Model, call.attempt.Operation, call.attempt.BatchIndex, call.attempt.InputItems, call.apiKey)
		if err != nil {
			return nil, err
		}
		// 实际req继续使用含HTTP客户端deadline的ctx；结算取消状态跟随调用者ctx，
		// 不把net/http在Body.Close时释放客户端计时器视为供应商调用被取消。
		next.ctx = call.ctx
		// net/http 的下一跳继承最初请求ctx。只有最终调用结束才能取消整条链；
		// 中间响应体结束仅结算该attempt，最终跳同时释放自身和逻辑调用ctx。
		logicalCancel, attemptCancel := call.cancel, next.cancel
		next.cancel = func() {
			if attemptCancel != nil {
				attemptCancel()
			}
			if logicalCancel != nil {
				logicalCancel()
			}
		}
		call.mu.Lock()
		call.last = next
		call.mu.Unlock()
		call = next
		req = req.WithContext(ctx)
	}
	call.mu.Lock()
	call.started = true
	call.mu.Unlock()
	trace := &httptrace.ClientTrace{WroteRequest: func(info httptrace.WroteRequestInfo) {
		if info.Err == nil {
			call.markDispatched()
		}
	}}
	response, err := base.RoundTrip(req.WithContext(httptrace.WithClientTrace(req.Context(), trace)))
	if err != nil {
		if call.autoFinish {
			_ = call.finish(err, false)
		}
		return nil, err
	}
	call.markDispatched()
	call.mu.Lock()
	call.observation.HTTPStatus = int64(response.StatusCode)
	call.observation.ProviderRequestID = safeProviderID(response.Header.Get("X-Request-Id"), call.apiKey)
	call.stream = strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/event-stream")
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		call.autoFinish = true
	}
	call.mu.Unlock()
	response.Body = &usageBody{ReadCloser: response.Body, call: call, redirect: response.StatusCode >= 300 && response.StatusCode < 400}
	return response, nil
}

func (c *usageCall) markDispatched() {
	c.mu.Lock()
	if c.observation.DispatchConfirmed {
		c.mu.Unlock()
		return
	}
	c.observation.DispatchConfirmed = true
	c.mu.Unlock()
	if c.hook == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.ctx), 3*time.Second)
	defer cancel()
	// 最终结算会再次带上已确认标记；暂态写错不能使已发出的请求被退款。
	_ = c.hook.Dispatched(ctx, c.attempt.AttemptID)
}

type usageBody struct {
	io.ReadCloser
	call     *usageCall
	redirect bool
}

func (b *usageBody) finish(cause error, eof bool) error {
	if b.redirect {
		return b.call.finishAttempt(cause, eof)
	}
	return b.call.finish(cause, eof)
}

func (b *usageBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		b.call.capture(p[:n])
	}
	if err != nil {
		b.call.mu.Lock()
		stream, auto := b.call.stream, b.call.autoFinish
		b.call.mu.Unlock()
		if stream || auto {
			if finishErr := b.finish(err, true); finishErr != nil {
				return n, finishErr
			}
		}
	}
	return n, err
}
func (b *usageBody) Close() error {
	err := b.ReadCloser.Close()
	b.call.mu.Lock()
	stream, auto := b.call.stream, b.call.autoFinish
	b.call.mu.Unlock()
	if stream || auto {
		if finishErr := b.finish(err, false); finishErr != nil {
			return finishErr
		}
	}
	return err
}

func (c *usageCall) capture(part []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stream {
		for _, value := range part {
			if value == '\n' {
				if !c.truncated {
					c.observeLine(c.buffer)
				}
				c.buffer = nil
				c.truncated = false
				continue
			}
			if len(c.buffer) < 256<<10 {
				c.buffer = append(c.buffer, value)
			} else {
				c.truncated = true
			}
		}
		return
	}
	// Embedding复用其既有解析缓冲，不额外保存大向量响应；聊天仅保留有界临时解析缓冲。
	if c.attempt.Operation == "embedding" || c.truncated {
		return
	}
	if len(c.buffer)+len(part) > 1<<20 {
		c.buffer = nil
		c.truncated = true
		return
	}
	c.buffer = append(c.buffer, part...)
}

func (c *usageCall) observeLine(line []byte) {
	line = bytes.TrimSpace(line)
	if !bytes.HasPrefix(line, []byte("data:")) {
		return
	}
	data := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
	if bytes.Equal(data, []byte("[DONE]")) {
		c.sawDone = true
		return
	}
	var envelope usageEnvelope
	if json.Unmarshal(data, &envelope) == nil {
		c.observeEnvelope(envelope)
	}
}
func (c *usageCall) observeEnvelope(envelope usageEnvelope) {
	if c.observation.ProviderRequestID == "" {
		c.observation.ProviderRequestID = safeProviderID(envelope.ID, c.apiKey)
	}
	if envelope.Usage == nil {
		return
	}
	u := envelope.Usage
	valid := func(v *int64) bool { return v == nil || *v >= 0 && *v <= 9007199254740991 }
	if !valid(u.Prompt) || !valid(u.Completion) || !valid(u.Total) {
		c.observation.UsageState = "invalid"
		c.metadata.observe(c.attempt.AttemptID, u, false)
		return
	}
	c.metadata.observe(c.attempt.AttemptID, u, true)
	c.observation.PromptTokens = u.Prompt
	c.observation.CompletionTokens = u.Completion
	c.observation.TotalTokens = u.Total
	complete := u.Prompt != nil && u.Total != nil
	if c.attempt.Operation == "embedding" {
		complete = complete && *u.Prompt == *u.Total
		c.observation.CompletionTokens = nil
	} else {
		complete = complete && u.Completion != nil && *u.Prompt+*u.Completion == *u.Total
	}
	if complete {
		c.observation.UsageState = "complete"
	} else if u.Prompt != nil || u.Completion != nil || u.Total != nil {
		c.observation.UsageState = "partial"
	}
}
func (c *usageCall) observeUsage(envelope usageEnvelope) {
	if c != nil {
		c.mu.Lock()
		last := c.last
		if last != nil {
			c.mu.Unlock()
			last.observeUsage(envelope)
			return
		}
		c.observeEnvelope(envelope)
		c.mu.Unlock()
	}
}

func (c *usageCall) finish(cause error, eof bool) error {
	if c == nil {
		return nil
	}
	if c.cancel != nil {
		defer c.cancel()
	}
	c.mu.Lock()
	last := c.last
	c.mu.Unlock()
	var lastErr error
	if last != nil {
		// 非流最终响应由SDK/Embedding完成类型校验后结算，避免在Read EOF抢先丢用量。
		lastErr = last.finishAttempt(cause, eof)
	}
	if err := c.finishAttempt(cause, eof); err != nil {
		return err
	}
	return lastErr
}

func (c *usageCall) finishAttempt(cause error, eof bool) error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if c.finished {
		err := c.finishErr
		c.mu.Unlock()
		return err
	}
	c.finished = true
	if !c.stream && !c.truncated && len(c.buffer) > 0 {
		var envelope usageEnvelope
		if json.Unmarshal(c.buffer, &envelope) == nil {
			c.observeEnvelope(envelope)
		}
	}
	c.buffer = nil
	if cause == nil || errors.Is(cause, io.EOF) {
		c.observation.Status = "succeeded"
	} else {
		c.observation.Status = "failed"
		c.observation.ErrorCode = "provider_error"
	}
	if !c.started {
		c.observation.Status = "not_sent"
		c.observation.ErrorCode = "before_dispatch"
	} else if c.ctx.Err() != nil || errors.Is(cause, context.Canceled) {
		c.observation.Status = "cancelled"
		c.observation.ErrorCode = "cancelled"
	} else if c.observation.HTTPStatus >= 400 {
		c.observation.Status = "failed"
		c.observation.ErrorCode = "provider_http_error"
	} else if c.stream && !c.sawDone {
		c.observation.Status = "unknown"
		c.observation.ErrorCode = "stream_incomplete"
	} else if !c.observation.DispatchConfirmed {
		c.observation.Status = "unknown"
		c.observation.ErrorCode = "dispatch_unknown"
	}
	if c.stream && !c.sawDone && c.observation.UsageState == "complete" {
		c.observation.UsageState = "partial"
	}
	observation := c.observation
	c.metadata.finish(c.attempt.AttemptID, cause != nil && !errors.Is(cause, io.EOF) || c.ctx.Err() != nil || c.stream && !c.sawDone)
	c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(c.ctx), 3*time.Second)
	defer cancel()
	var err error
	if c.hook != nil {
		err = c.hook.Finish(ctx, c.attempt.AttemptID, observation)
	}
	if err != nil {
		err = xerr.NewErrCodeMsg(503, "模型用量结算未确认，请保留原请求标识核对")
	}
	c.mu.Lock()
	c.finishErr = err
	c.mu.Unlock()
	return err
}

var providerIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

func safeProviderID(value, key string) string {
	if !providerIDPattern.MatchString(value) || key != "" && strings.Contains(value, key) {
		return ""
	}
	return value
}
