// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appaccount

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/appaccount"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 退出当前或全部设备
func LogoutAppAccountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AppLogoutRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := appaccount.NewLogoutAppAccountLogic(r.Context(), svcCtx)
		resp, err := l.LogoutAppAccount(&req)
		result.HttpResult(r, w, resp, err)
	}
}
