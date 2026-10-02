package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/ctxJwt"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

type AuditRecorder interface {
	RecordAudit(context.Context, *pb.RecordAuditRequest, ...grpc.CallOption) (*pb.NoDataResponse, error)
}

type AuditMiddleware struct{ recorder AuditRecorder }

func NewAuditMiddleware(recorder AuditRecorder) *AuditMiddleware {
	return &AuditMiddleware{recorder: recorder}
}

// Handle records mutations and login attempts. It also runs outside JWT/session
// middleware to retain denied attempts. Read-only requests are not logged.
func (m *AuditMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		traceID := ""
		if span := trace.SpanContextFromContext(r.Context()); span.IsValid() {
			traceID = span.TraceID().String()
		}
		ip := remoteIP(r.RemoteAddr)
		ctx := audit.WithRequest(r.Context(), audit.Request{Path: r.URL.Path, Method: r.Method, IP: ip, TraceID: traceID, UserAgent: r.UserAgent()})
		verified := ctxJwt.GetJwtData(ctx)
		if verified.ID > 0 {
			audit.SetActor(ctx, audit.Actor{ID: verified.ID, Name: verified.Username, AuthorityID: verified.AuthorityId})
		}
		r = r.WithContext(ctx)
		if !auditRequest(r) {
			next(w, r)
			return
		}
		params, attemptedName := requestParams(r)
		response := &auditResponseWriter{ResponseWriter: w}
		panicked := true
		defer func() {
			actor := audit.ActorFromContext(ctx)
			eventType := "operation"
			if r.URL.Path == "/v1/sys/login" {
				eventType = "login"
				if actor.Name == "" {
					actor.Name = attemptedName
				}
				// Extract only identity from the successful login response, never
				// persist any other response field (especially accessToken).
				if actor.ID == 0 && response.success() {
					actor = loginActor(response.body.Bytes(), actor)
				}
			}
			status := response.status
			if status == 0 {
				status = http.StatusOK
			}
			result := "success"
			if panicked {
				status = http.StatusInternalServerError
			}
			if panicked || !response.success() {
				result = "failure"
			}
			module, action := requestAction(r.URL.Path)
			log := &pb.AuditLog{ActorID: actor.ID, ActorName: actor.Name, AuthorityId: actor.AuthorityID,
				EventType: eventType, Module: module, Action: action, Object: objectFromParams(params),
				Path: r.URL.Path, Method: r.Method, Result: result, StatusCode: int64(status), IP: ip,
				TraceID: traceID, DurationMs: time.Since(started).Milliseconds(), Params: params}
			// A canceled browser request must not cancel its audit event. Bound
			// the separate RPC; failure is surfaced in operational logs.
			auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer cancel()
			if m.recorder != nil {
				if _, err := m.recorder.RecordAudit(auditCtx, &pb.RecordAuditRequest{Log: log}); err != nil {
					logx.WithContext(ctx).Errorf("audit persistence failed path=%s action=%s: %v", r.URL.Path, action, err)
				}
			}
		}()
		next(response, r)
		panicked = false
	}
}

func auditRequest(r *http.Request) bool {
	if !strings.HasPrefix(r.URL.Path, "/v1/") {
		return false
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

func requestParams(r *http.Request) (string, string) {
	params := map[string]any{}
	for key, values := range r.URL.Query() {
		if len(values) == 1 {
			var number json.Number
			if json.Unmarshal([]byte(values[0]), &number) == nil {
				params[key] = number
			}
		}
	}
	var attemptedName string
	if r.Body != nil && strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		// Read a bounded prefix and restore all bytes so the handler sees the
		// identical body, even for multipart/oversized/invalid requests.
		prefix, _ := io.ReadAll(io.LimitReader(r.Body, 16*1024+1))
		r.Body = &restoredBody{Reader: io.MultiReader(bytes.NewReader(prefix), r.Body), Closer: r.Body}
		if len(prefix) <= 16*1024 {
			var input map[string]json.RawMessage
			if json.Unmarshal(prefix, &input) == nil {
				for key, value := range input {
					params[key] = value
				}
				if r.URL.Path == "/v1/sys/login" {
					for key, value := range input {
						if strings.EqualFold(key, "userName") {
							_ = json.Unmarshal(value, &attemptedName)
						}
					}
				}
			}
		}
	}
	encoded, _ := json.Marshal(params)
	return audit.SanitizeJSON(encoded), attemptedName
}

type restoredBody struct {
	io.Reader
	io.Closer
}

type auditResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *auditResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *auditResponseWriter) WriteHeader(status int) {
	if status >= 200 && w.status != 0 {
		return
	}
	if status >= 200 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *auditResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if room := 16*1024 - w.body.Len(); room > 0 {
		if len(data) > room {
			w.body.Write(data[:room])
		} else {
			w.body.Write(data)
		}
	}
	return w.ResponseWriter.Write(data)
}
func (w *auditResponseWriter) Flush() {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}
func (w *auditResponseWriter) success() bool {
	if w.status >= 400 {
		return false
	}
	var envelope struct {
		Code    *int64 `json:"code"`
		Success *bool  `json:"success"`
	}
	if json.Unmarshal(w.body.Bytes(), &envelope) == nil {
		if envelope.Success != nil && !*envelope.Success {
			return false
		}
		if envelope.Code != nil && *envelope.Code != 0 && *envelope.Code != 200 {
			return false
		}
	}
	return true
}

func loginActor(data []byte, fallback audit.Actor) audit.Actor {
	var response struct {
		Result struct {
			UserInfo struct {
				ID          int64  `json:"id"`
				Username    string `json:"username"`
				AuthorityID int64  `json:"authorityId"`
			} `json:"userInfo"`
		} `json:"result"`
	}
	if json.Unmarshal(data, &response) == nil && response.Result.UserInfo.ID > 0 {
		user := response.Result.UserInfo
		return audit.Actor{ID: user.ID, Name: user.Username, AuthorityID: user.AuthorityID}
	}
	return fallback
}

func remoteIP(address string) string {
	if host, _, err := net.SplitHostPort(address); err == nil {
		return host
	}
	return address
}

func requestAction(value string) (string, string) {
	action := path.Base(value)
	parts := strings.Split(strings.Trim(value, "/"), "/")
	module := "sys"
	if len(parts) >= 4 {
		module = parts[2]
	} else if len(parts) >= 3 && parts[1] == "sys" {
		module = "user"
	}
	return module, action
}

func objectFromParams(params string) string {
	var values map[string]json.RawMessage
	_ = json.Unmarshal([]byte(params), &values)
	for _, key := range []string{"userid", "id", "authorityid", "sysbasemenuid", "sysdictionaryid", "ids"} {
		if value, ok := values[key]; ok {
			return string(value)
		}
	}
	return ""
}
