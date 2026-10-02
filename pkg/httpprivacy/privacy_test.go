package httpprivacy

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/core/load"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/handler"
)

func capturedLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	previous := logx.Reset()
	buffer := new(bytes.Buffer)
	logx.SetWriter(logx.NewWriter(buffer))
	if err := Install(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { logx.SetWriter(previous) })
	return buffer
}

func TestRealJWTFailureDoesNotLogCredentials(t *testing.T) {
	logs := capturedLogs(t)
	body := `{"oldPassword":"old-private-value","newPassword":"new-private-value"}`
	for _, authorization := range []string{"", "Bearer invalid-private-token"} {
		request := httptest.NewRequest(http.MethodPut, "/v1/sys/changePassword?signature=query-private-value", strings.NewReader(body))
		request.Header.Set("Authorization", authorization)
		request.Header.Set("Content-Type", "application/json")
		called := false
		middleware := handler.Authorize("test-only-secret", handler.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
			called = true
			data, _ := io.ReadAll(r.Body)
			if string(data) != body {
				t.Error("framework dump altered the handler body")
			}
			w.WriteHeader(http.StatusUnauthorized)
		}))
		response := httptest.NewRecorder()
		middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("invalid JWT reached the handler") })).ServeHTTP(response, request)
		if !called || response.Code != http.StatusUnauthorized {
			t.Fatal("JWT rejection callback or response changed")
		}
	}
	for _, secret := range []string{"old-private-value", "new-private-value", "invalid-private-token", "query-private-value"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatal("JWT failure persisted request credentials")
		}
	}
	if !strings.Contains(logs.String(), "JWT authorization failed") {
		t.Fatal("failure signal was removed with the private content")
	}
}

func TestFrameworkErrorAndVerboseResponseDumpsAreSuppressed(t *testing.T) {
	logs := capturedLogs(t)
	for _, wrap := range []func(http.Handler) http.Handler{handler.LogHandler, handler.DetailedLogHandler} {
		request := httptest.NewRequest(http.MethodPost, "/v1/sys/login?signature=query-secret", strings.NewReader(`{"password":"request-secret"}`))
		request.Header.Set("Authorization", "Bearer header-secret")
		response := httptest.NewRecorder()
		wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"accessToken":"response-secret","url":"https://objects.example/image?signature=download-secret"}`))
		})).ServeHTTP(response, request)
	}
	for _, secret := range []string{"request-secret", "header-secret", "response-secret", "download-secret", "query-secret"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatal("framework HTTP dump persisted private content")
		}
	}
	logx.Info("ordinary application message")
	if !strings.Contains(logs.String(), "ordinary application message") {
		t.Fatal("privacy wrapper dropped unrelated application logs")
	}
	conf := rest.RestConf{Verbose: true, Middlewares: rest.MiddlewaresConf{Log: true, Prometheus: true, Timeout: true}}
	ConfigureServer(&conf)
	if conf.Verbose || conf.Middlewares.Log || !conf.Middlewares.Prometheus || !conf.Middlewares.Timeout {
		t.Fatal("privacy configuration changed unrelated safety/metrics middleware")
	}
}

func TestInstallRetainsWriterAndConfiguredLevel(t *testing.T) {
	previous := logx.Reset()
	buffer := new(bytes.Buffer)
	configured := logx.NewWriter(buffer)
	logx.SetWriter(configured)
	logx.SetLevel(logx.ErrorLevel)
	t.Cleanup(func() { logx.SetWriter(previous); logx.SetLevel(logx.InfoLevel) })
	if err := Install(); err != nil {
		t.Fatal(err)
	}
	wrapper := logx.Reset()
	if got, ok := wrapper.(*privacyWriter); !ok || got.next != configured {
		t.Fatal("configured writer was replaced instead of wrapped")
	}
	logx.SetWriter(wrapper)
	logx.Info("level-must-stay-suppressed")
	logx.Error("ordinary error retained")
	if strings.Contains(buffer.String(), "level-must-stay-suppressed") || !strings.Contains(buffer.String(), "ordinary error retained") {
		t.Fatal("privacy installation changed the configured log level")
	}
	if err := Install(); err != nil {
		t.Fatal("repeated install failed", err)
	}
}

type rejectingShedder struct{}

func (rejectingShedder) Allow() (load.Promise, error) { return nil, errors.New("capacity exceeded") }

func TestSheddingDoesNotLogQueryCredentials(t *testing.T) {
	logs := capturedLogs(t)
	request := httptest.NewRequest(http.MethodGet, "/v1/sys/files/access?id=1&signature=private-query", nil)
	response := httptest.NewRecorder()
	handler.SheddingHandler(rejectingShedder{}, stat.NewMetrics("httpprivacy-test"))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("rejected request reached handler") })).ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable || strings.Contains(logs.String(), "private-query") || !strings.Contains(logs.String(), "load shedding") {
		t.Fatal("capacity rejection leaked query or lost its failure signal")
	}
}
