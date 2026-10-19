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

// 邮箱或手机号注册，等待管理员审核
func RegisterAppAccountHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AppRegisterRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := appauth.NewRegisterAppAccountLogic(r.Context(), svcCtx)
		resp, err := l.RegisterAppAccount(&req)
		result.HttpResult(r, w, resp, err)
	}
}
