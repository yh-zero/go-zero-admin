// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appauth

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/appauth"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 已审核 App 账号登录
func LoginAppAccountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AppLoginRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := appauth.NewLoginAppAccountLogic(r.Context(), svcCtx)
		resp, err := l.LoginAppAccount(&req)
		result.HttpResult(r, w, resp, err)
	}
}
