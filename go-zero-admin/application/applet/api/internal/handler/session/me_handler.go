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

// 获取当前登录用户
func MeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.MeRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := session.NewMeLogic(r.Context(), svcCtx)
		resp, err := l.Me(&req)
		result.HttpResult(r, w, resp, err)
	}
}
