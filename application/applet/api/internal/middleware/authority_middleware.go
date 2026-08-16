package middleware

import (
	"net/http"
	"strconv"

	"go-zero-admin/pkg/ctxJwt"

	casbinRPC "go-zero-admin/application/applet/rpc/client/casbin"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// AuthorityMiddleware 权限中间件 通过RPC调用casbin鉴权 api层不直连数据库
type AuthorityMiddleware struct {
	CasbinRPC casbinRPC.Casbin
}

func NewAuthorityMiddleware(casbinCli casbinRPC.Casbin) *AuthorityMiddleware {
	return &AuthorityMiddleware{
		CasbinRPC: casbinCli,
	}
}

func (m *AuthorityMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authorityId := ctxJwt.GetJwtDataAuthorityId(r.Context())
		resp, err := m.CasbinRPC.Enforce(r.Context(), &casbinRPC.EnforceRequest{
			AuthorityId: strconv.FormatInt(authorityId, 10),
			Path:        r.URL.Path,
			Method:      r.Method,
		})
		if err != nil {
			logx.Errorf("AuthorityMiddleware Enforce err: %v", err)
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]string{"message": "鉴权服务异常"})
			return
		}
		if !resp.Pass {
			logx.Errorf("---------- 权限不足 path: %s method: %s ------------", r.URL.Path, r.Method)
			httpx.WriteJson(w, http.StatusForbidden, map[string]string{"message": "权限不足"})
			return
		}
		next(w, r)
	}
}
