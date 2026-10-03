package aiagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func httpFixture(t *testing.T, provider, endpoint string, modify func(*Config)) Runner {
	t.Helper()
	c := Config{Enabled: true, Provider: provider, Model: "fixture-model", APIKey: "fixture-secret", BaseURL: endpoint, AllowHTTPForLoopback: true}
	if modify != nil {
		modify(&c)
	}
	r, err := New(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestExternalAdaptersUseRealHTTPAndFunctionCalling(t *testing.T) {
	for _, provider := range []string{ProviderDeepSeek, ProviderQwen} {
		t.Run(provider, func(t *testing.T) {
			var requests, toolCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer fixture-secret" {
					t.Error("incorrect compatible endpoint or server-side credential")
				}
				var payload map[string]json.RawMessage
				if json.NewDecoder(r.Body).Decode(&payload) != nil {
					t.Error("invalid compatible API body")
				}
				if string(payload["model"]) != `"fixture-model"` || string(payload["max_tokens"]) != "1024" {
					t.Error("model/output budget was not transmitted")
				}
				if provider == ProviderQwen && string(payload["enable_thinking"]) != "false" {
					t.Error("Qwen non-streaming thinking must be disabled")
				}
				if provider == ProviderDeepSeek && string(payload["thinking"]) != `{"type":"disabled"}` {
					t.Error("DeepSeek thinking must be disabled when reasoning is not persisted")
				}
				w.Header().Set("Content-Type", "application/json")
				if requests.Add(1) == 1 {
					if !strings.Contains(string(payload["tools"]), "get_file_status") {
						t.Error("registered tool schema was omitted")
					}
					fmt.Fprint(w, `{"id":"fixture","object":"chat.completion","model":"fixture-model","choices":[{"index":0,"message":{"role":"assistant","content":"","reasoning_content":"never-persist-this","tool_calls":[{"id":"call_http","type":"function","function":{"name":"get_file_status","arguments":"{\"fileId\":1}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":2,"total_tokens":12}}`)
				} else {
					if !strings.Contains(string(payload["messages"]), "untrustedBusinessData") {
						t.Error("tool result did not reach provider")
					}
					if !strings.Contains(string(payload["messages"]), "never-persist-this") {
						t.Error("provider-required private reasoning was not echoed for the tool turn")
					}
					fmt.Fprint(w, `{"id":"fixture2","object":"chat.completion","model":"fixture-model","choices":[{"index":0,"message":{"role":"assistant","content":"文件正常","reasoning_content":"private-final"},"finish_reason":"stop"}],"usage":{"prompt_tokens":20,"completion_tokens":3,"total_tokens":23}}`)
				}
			}))
			defer server.Close()
			r := httpFixture(t, provider, server.URL+"/v1", nil)
			result, err := r.Run(context.Background(), fixtureRequest(fixtureTool(func(context.Context, json.RawMessage) (ToolOutput, error) {
				toolCalls.Add(1)
				return ToolOutput{Content: `{"status":"active"}`, Summary: "文件状态已返回", Count: 1}, nil
			})), nil)
			if err != nil || requests.Load() != 2 || toolCalls.Load() != 1 || result.FinalAnswer != "文件正常" || result.Usage.TotalTokens != 35 || result.Provider != provider || result.Model != "fixture-model" {
				t.Fatalf("adapter/ADK did not complete: %+v %v", result, err)
			}
			encoded, _ := json.Marshal(result)
			if strings.Contains(string(encoded), "private-final") || strings.Contains(string(encoded), "never-persist") {
				t.Fatal("reasoning leaked into result")
			}
		})
	}
}

func TestModelRedirectCannotSendCredentialToDestination(t *testing.T) {
	var destinationRequests atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { destinationRequests.Add(1); w.WriteHeader(http.StatusOK) }))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", destination.URL+"/stolen")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	defer source.Close()
	_, err := httpFixture(t, ProviderDeepSeek, source.URL+"/v1", nil).Run(context.Background(), fixtureRequest(), nil)
	if !errors.Is(err, ErrModelFailed) || destinationRequests.Load() != 0 {
		t.Fatalf("redirect followed: requests=%d err=%v", destinationRequests.Load(), err)
	}
}

func TestModelResponseBoundsAndProviderErrorsStayPrivate(t *testing.T) {
	for _, variant := range []string{"content-length", "chunked", "provider-error"} {
		t.Run(variant, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if variant == "provider-error" {
					w.WriteHeader(http.StatusUnauthorized)
					fmt.Fprint(w, `{"error":{"message":"fixture-secret and private-request-content","type":"invalid_api_key","code":"bad_key"}}`)
					return
				}
				if variant == "chunked" {
					w.(http.Flusher).Flush()
				} else {
					w.Header().Set("Content-Length", "4096")
				}
				fmt.Fprint(w, strings.Repeat("s", 4096))
			}))
			defer server.Close()
			r := httpFixture(t, ProviderDeepSeek, server.URL+"/v1", func(c *Config) { c.MaxResponseBytes = 256 })
			result, err := r.Run(context.Background(), fixtureRequest(), nil)
			if err == nil || strings.Contains(err.Error(), "fixture-secret") || strings.Contains(err.Error(), "private-request") || result.FinalAnswer != "" {
				t.Fatalf("unsafe model error: %+v %v", result, err)
			}
		})
	}
}

func TestHTTPCallReceivesCancellationAndRequestTimeout(t *testing.T) {
	for _, variant := range []string{"cancel", "timeout"} {
		t.Run(variant, func(t *testing.T) {
			started := make(chan struct{})
			requestCanceled := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()
				close(started)
				select {
				case <-r.Context().Done():
					close(requestCanceled)
				case <-release:
				}
			}))
			defer func() { close(release); server.Close() }()
			r := httpFixture(t, ProviderDeepSeek, server.URL+"/v1", func(c *Config) {
				if variant == "timeout" {
					c.RequestTimeout = 50 * time.Millisecond
				}
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() { _, err := r.Run(ctx, fixtureRequest(), nil); done <- err }()
			<-started
			if variant == "cancel" {
				cancel()
			}
			select {
			case err := <-done:
				if variant == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if variant == "timeout" && !errors.Is(err, ErrModelFailed) {
					t.Fatal(err)
				}
			case <-time.After(time.Second):
				t.Fatal("HTTP cancellation/timeout did not return promptly")
			}
			select {
			case <-requestCanceled:
			case <-time.After(time.Second):
				t.Fatal("provider request context was not canceled")
			}
		})
	}
}
