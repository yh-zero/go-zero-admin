package aiagent

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
)

// The initial history is limited to 64 KiB; this larger ceiling includes system
// instructions, schemas and generated tool turns without rejecting normal data.
const maxModelInputBytes = 256 << 10

// New builds the Eino ADK adapter without making a network request. Disabled
// configuration produces a disabled runner, so an unconfigured server can boot.
func New(ctx context.Context, config Config) (Runner, error) {
	c, err := config.normalized()
	if err != nil {
		return nil, err
	}
	if !c.Enabled {
		return &runner{config: c}, nil
	}
	transport := &http.Transport{
		Proxy:             http.ProxyFromEnvironment,
		DialContext:       (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2: true, MaxIdleConns: 20, MaxIdleConnsPerHost: 5,
		ResponseHeaderTimeout: c.RequestTimeout, MaxResponseHeaderBytes: 32 << 10,
		IdleConnTimeout: 30 * time.Second, TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	client := &http.Client{
		Timeout:   c.RequestTimeout,
		Transport: &boundedTransport{base: transport, maxBytes: c.MaxResponseBytes},
		// A model endpoint is configured by the server, and must never redirect
		// the request (including its bearer credential) to another endpoint.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	extra := map[string]any{"thinking": map[string]any{"type": "disabled"}}
	if c.Provider == ProviderQwen {
		extra = map[string]any{"enable_thinking": false}
	}
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey: c.APIKey, BaseURL: c.BaseURL, Model: c.Model,
		HTTPClient: client, MaxTokens: &c.MaxOutputTokens, ExtraFields: extra,
	})
	if err != nil {
		return nil, ErrModelFailed
	}
	return &runner{config: c, chatModel: chatModel}, nil
}

type boundedTransport struct {
	base     http.RoundTripper
	maxBytes int64
}

func (t *boundedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.ContentLength > maxModelInputBytes {
		if req.Body != nil {
			_ = req.Body.Close()
		}
		return nil, ErrBudgetExceeded
	}
	if req.Body != nil {
		data, err := io.ReadAll(io.LimitReader(req.Body, maxModelInputBytes+1))
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		if len(data) > maxModelInputBytes {
			return nil, ErrBudgetExceeded
		}
		copy := req.Clone(req.Context())
		copy.Body = io.NopCloser(bytes.NewReader(data))
		copy.ContentLength = int64(len(data))
		req = copy
	}
	response, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.ContentLength > t.maxBytes {
		return nil, ErrBudgetExceeded
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, t.maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > t.maxBytes {
		return nil, ErrBudgetExceeded
	}
	response.Body = io.NopCloser(bytes.NewReader(data))
	return response, nil
}
