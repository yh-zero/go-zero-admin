// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appmembers

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/appmembers"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 审核或禁用恢复 App 账号
func SetAppUserStatusHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AppStatusRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := appmembers.NewSetAppUserStatusLogic(r.Context(), svcCtx)
		resp, err := l.SetAppUserStatus(&req)
		result.HttpResult(r, w, resp, err)
	}
}
