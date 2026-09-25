// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package session

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/session"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 退出登录并撤销该用户全部会话
func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LogoutRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := session.NewLogoutLogic(r.Context(), svcCtx)
		resp, err := l.Logout(&req)
		result.HttpResult(r, w, resp, err)
	}
}
