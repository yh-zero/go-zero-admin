package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"go-zero-admin/pkg/casbinauth"
	"go-zero-admin/pkg/jwtx"
	"go-zero-admin/pkg/xerr"
)

// context key 类型（避免冲突）
type ctxKey string

const (
	CtxKeyUserID   ctxKey = "user_id"
	CtxKeyUsername ctxKey = "username"
	CtxKeyDeptId   ctxKey = "dept_id"
)

// Authority Casbin 接口鉴权中间件
type Authority struct {
	enforcer casbinauth.Enforcer
	secret   string
}

func NewAuthority(enforcer casbinauth.Enforcer, jwtSecret string) *Authority {
	return &Authority{enforcer: enforcer, secret: jwtSecret}
}

// Handle JWT 解析 → Casbin 鉴权 → 注入 claims 到 context
func (a *Authority) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR))
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := jwtx.ParseAccessToken(a.secret, tokenString)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR))
			return
		}

		sub := fmt.Sprintf("user:%d", claims.UserID)
		dom := fmt.Sprintf("dept:%d", claims.DeptId)
		if deptHeader := r.Header.Get("X-Dept-Id"); deptHeader != "" {
			dom = "dept:" + deptHeader
		}
		obj := r.URL.Path
		act := r.Method

		ok, err := a.enforcer.Enforce(sub, dom, obj, act)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("casbin enforce error: %v", err)
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR))
			return
		}
		if !ok {
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(xerr.PERMISSION_DENIED))
			return
		}

		ctx := context.WithValue(r.Context(), CtxKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, CtxKeyUsername, claims.Username)
		ctx = context.WithValue(ctx, CtxKeyDeptId, claims.DeptId)
		next(w, r.WithContext(ctx))
	}
}
