package middleware

import (
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/applet/rpc/client/user"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"go-zero-admin/pkg/result"
	"go-zero-admin/pkg/result/xerr"
	"net/http"
)

type SessionMiddleware struct{ UserRPC user.User }

func NewSessionMiddleware(client user.User) *SessionMiddleware {
	return &SessionMiddleware{UserRPC: client}
}
func (m *SessionMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data := ctxJwt.GetJwtData(r.Context())
		response, err := m.UserRPC.CheckSession(r.Context(), &pb.SessionRequest{UserID: data.ID, SessionVersion: data.SessionVersion, AuthorityId: data.AuthorityId})
		if err != nil {
			logx.WithContext(r.Context()).Errorf("session validation unavailable: %v", err)
			httpx.WriteJson(w, http.StatusServiceUnavailable, result.Error(xerr.SERVER_COMMON_ERROR, "登录校验服务暂不可用"))
			return
		}
		if !response.Valid {
			httpx.WriteJson(w, http.StatusUnauthorized, result.Error(xerr.TOKEN_EXPIRE_ERROR, "登录已失效，请重新登录"))
			return
		}
		next(w, r)
	}
}
