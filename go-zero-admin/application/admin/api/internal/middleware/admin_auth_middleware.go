package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/client/deptservice"
	"go-zero-admin/application/admin/rpc/client/profileservice"
	"go-zero-admin/pkg/casbinauth"
	"go-zero-admin/pkg/jwtx"
	"go-zero-admin/pkg/xerr"
)

// context key 类型（避免冲突）
type ctxKey string

const (
	CtxKeyUserID      ctxKey = "user_id"
	CtxKeyUsername    ctxKey = "username"
	CtxKeyDeptId      ctxKey = "dept_id"
	CtxKeySessionId   ctxKey = "session_id"
	CtxKeyAccessToken ctxKey = "access_token"
)

// ActorFromContext 只使用鉴权与请求元数据中间件写入的可信身份。
func ActorFromContext(ctx context.Context) *adminpb.Actor {
	userId, _ := ctx.Value(CtxKeyUserID).(int64)
	username, _ := ctx.Value(CtxKeyUsername).(string)
	deptId, _ := ctx.Value(CtxKeyDeptId).(int64)
	ip, _ := ctx.Value(CtxKeyClientIP).(string)
	ua, _ := ctx.Value(CtxKeyUserAgent).(string)
	return &adminpb.Actor{UserId: userId, Username: username, DeptId: deptId, Ip: ip, UserAgent: ua}
}

// Authority Casbin 接口鉴权中间件
type Authority struct {
	enforcer *casbinauth.Holder
	secret   string
	sessions *jwtx.SessionStore
	profiles profileservice.ProfileService
	depts    deptservice.DeptService
}

func NewAuthority(enforcer *casbinauth.Holder, jwtSecret string, sessions *jwtx.SessionStore, profiles profileservice.ProfileService, depts deptservice.DeptService) *Authority {
	return &Authority{enforcer: enforcer, secret: jwtSecret, sessions: sessions, profiles: profiles, depts: depts}
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
		if _, err = a.sessions.Validate(r.Context(), claims.SessionId, claims.UserID, claims.ID); err != nil {
			code := uint32(xerr.SERVER_COMMON_ERROR)
			if errors.Is(err, jwtx.ErrSessionInvalid) {
				code = xerr.TOKEN_EXPIRE_ERROR
			}
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(code))
			return
		}
		current, err := a.profiles.GetProfile(r.Context(), &adminpb.GetProfileRequest{Actor: &adminpb.Actor{UserId: claims.UserID}})
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if current.User == nil || current.User.Status != 1 {
			_ = a.sessions.RevokeAll(r.Context(), claims.UserID)
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(xerr.USER_DISABLED))
			return
		}
		deptId := current.User.DeptId
		MarkAuditIdentity(r.Context(), claims.UserID, current.User.Username, deptId)
		departments, err := a.depts.GetMyDepts(r.Context(), &adminpb.GetMyDeptsRequest{Actor: &adminpb.Actor{UserId: claims.UserID, Username: current.User.Username, DeptId: deptId}})
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		allowed := make(map[int64]bool, len(departments.List))
		for _, dept := range departments.List {
			allowed[dept.Id] = true
		}
		if header := r.Header.Get("X-Dept-Id"); header != "" {
			selected, parseErr := strconv.ParseInt(header, 10, 64)
			if parseErr != nil || selected < 0 || (selected == 0 && current.User.DeptId != 0) || (selected > 0 && !allowed[selected]) {
				httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(xerr.PERMISSION_DENIED))
				return
			}
			deptId = selected
		} else if deptId > 0 && !allowed[deptId] {
			deptId = 0
			if len(departments.List) > 0 {
				deptId = departments.List[0].Id
			}
		}
		MarkAuditIdentity(r.Context(), claims.UserID, current.User.Username, deptId)
		ctx := context.WithValue(r.Context(), CtxKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, CtxKeyUsername, current.User.Username)
		ctx = context.WithValue(ctx, CtxKeyDeptId, deptId)
		ctx = context.WithValue(ctx, CtxKeySessionId, claims.SessionId)
		ctx = context.WithValue(ctx, CtxKeyAccessToken, tokenString)
		if isOwnAccountRoute(r) {
			next(w, r.WithContext(ctx))
			return
		}

		sub := fmt.Sprintf("user:%d", claims.UserID)
		dom := fmt.Sprintf("dept:%d", deptId)
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

		next(w, r.WithContext(ctx))
	}
}

func isOwnAccountRoute(r *http.Request) bool {
	if r.Method == http.MethodPost && r.URL.Path == "/v1/sys/auth/logout" {
		return true
	}
	if r.Method == http.MethodGet && (r.URL.Path == "/v1/sys/auth/sessions" || r.URL.Path == "/v1/sys/dept/my") {
		return true
	}
	if r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/v1/sys/auth/sessions/") {
		return !strings.Contains(strings.TrimPrefix(r.URL.Path, "/v1/sys/auth/sessions/"), "/")
	}
	return false
}
