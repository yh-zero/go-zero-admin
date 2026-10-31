package middleware

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
	"net/http"
	"strings"
)

const CtxKeyAppAccessToken ctxKey = "app_access_token"
const CtxKeyAppUserId ctxKey = "app_user_id"

type AppAuthMiddleware struct{ svcCtx *svc.ServiceContext }

func NewAppAuthMiddleware(s *svc.ServiceContext) *AppAuthMiddleware {
	return &AppAuthMiddleware{svcCtx: s}
}
func (m *AppAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if m.svcCtx.AppAuthRpc == nil {
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE))
			return
		}
		if publicAppRoute(r) {
			next(w, r)
			return
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR))
			return
		}
		token := strings.TrimPrefix(auth, "Bearer ")
		resp, err := m.svcCtx.AppAuthRpc.AppAccess(r.Context(), &apppb.AppAccessRequest{AccessToken: token})
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if resp.User == nil || resp.User.Id <= 0 {
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR))
			return
		}
		MarkAuditIdentity(r.Context(), 0, fmt.Sprintf("app:%d", resp.User.Id), 0)
		ctx := context.WithValue(r.Context(), CtxKeyAppAccessToken, token)
		ctx = context.WithValue(ctx, CtxKeyAppUserId, resp.User.Id)
		next(w, r.WithContext(ctx))
	}
}
func publicAppRoute(r *http.Request) bool {
	if r.Method == http.MethodGet && r.URL.Path == "/v1/app/auth/options" {
		return true
	}
	if r.Method != http.MethodPost {
		return false
	}
	switch r.URL.Path {
	case "/v1/app/auth/register", "/v1/app/auth/login", "/v1/app/auth/refresh", "/v1/app/auth/email-code":
		return true
	}
	return false
}
