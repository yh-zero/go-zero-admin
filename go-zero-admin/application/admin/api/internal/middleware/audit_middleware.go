package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/client/auditservice"
)

const auditContextKey ctxKey = "audit_state"

type auditState struct {
	mutex    sync.Mutex
	userId   int64
	username string
	deptId   int64
	failed   bool
}

// MarkAuditIdentity 只传入服务端验证后的身份，不从请求体接收操作者 ID。
func MarkAuditIdentity(ctx context.Context, userId int64, username string, deptId int64) {
	if state, ok := ctx.Value(auditContextKey).(*auditState); ok {
		state.mutex.Lock()
		defer state.mutex.Unlock()
		state.userId, state.username, state.deptId = userId, username, deptId
	}
}

func MarkAuditError(ctx context.Context) {
	if state, ok := ctx.Value(auditContextKey).(*auditState); ok {
		state.mutex.Lock()
		defer state.mutex.Unlock()
		state.failed = true
	}
}

type auditWriter struct {
	http.ResponseWriter
	status int
}

func (w *auditWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
		w.ResponseWriter.WriteHeader(code)
	}
}
func (w *auditWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(data)
}
func (w *auditWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Audit 在认证外层记录写请求结果，既不读取请求体也不缓冲含令牌的响应。
func Audit(client auditservice.AuditService) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
				next(w, r)
				return
			}
			state := &auditState{}
			ctx := context.WithValue(r.Context(), auditContextKey, state)
			writer := &auditWriter{ResponseWriter: w}
			started := time.Now()
			completed := false
			defer func() { persistAudit(ctx, client, r, state, writer.status, started, completed) }()
			next(writer, r.WithContext(ctx))
			completed = true
		}
	}
}

func persistAudit(ctx context.Context, client auditservice.AuditService, r *http.Request, state *auditState, status int, started time.Time, completed bool) {
	state.mutex.Lock()
	actor := &adminpb.Actor{UserId: state.userId, Username: state.username, DeptId: state.deptId}
	failed := state.failed || status >= 400 || !completed
	state.mutex.Unlock()
	actor.Ip, _ = ctx.Value(CtxKeyClientIP).(string)
	actor.UserAgent, _ = ctx.Value(CtxKeyUserAgent).(string)
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	module, action := "", strings.ToLower(r.Method)
	if len(parts) >= 3 {
		module = strings.Join(parts[1:3], "/")
	}
	if len(parts) >= 4 && parts[2] == "auth" {
		action = parts[3]
	}
	result, errorMessage := int32(1), ""
	if failed {
		result, errorMessage = 2, "请求失败"
	}
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
	defer cancel()
	_, err := client.CreateAuditLog(auditCtx, &adminpb.CreateAuditLogRequest{Actor: actor, Module: module, Action: action, Method: r.Method, Path: r.URL.Path, Result: result, ErrorMsg: errorMessage, DurationMs: time.Since(started).Milliseconds()})
	if err != nil {
		logx.WithContext(ctx).Error("audit record could not be persisted")
	}
}

const (
	// CtxKeyClientIP 真实客户端 IP（限流与登录溯源用）
	CtxKeyClientIP ctxKey = "client_ip"
	// CtxKeyUserAgent 客户端 UA
	CtxKeyUserAgent ctxKey = "user_agent"
)
