package middleware

import (
	"context"
	"net"
	"net/http"
)

const (
	// CtxKeyClientIP 真实客户端 IP（限流与登录溯源用）
	CtxKeyClientIP ctxKey = "client_ip"
	// CtxKeyUserAgent 客户端 UA
	CtxKeyUserAgent ctxKey = "user_agent"
)

// RequestMeta 提取客户端 IP 与 UA 注入 ctx。
// 只信 TCP 对端（RemoteAddr），不解析 X-Forwarded-For——本地开发无可信反代，
// 信任 XFF 会让客户端伪造 IP 绕过限流；挂可信反代后应改用 go-zero TrustedProxies 配置。
func RequestMeta(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), CtxKeyClientIP, clientIP(r))
		ctx = context.WithValue(ctx, CtxKeyUserAgent, r.UserAgent())
		next(w, r.WithContext(ctx))
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
