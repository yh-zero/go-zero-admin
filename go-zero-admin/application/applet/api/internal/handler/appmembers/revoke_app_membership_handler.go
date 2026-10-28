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

// 撤销指定会员授予
func RevokeAppMembershipHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AppGrantRevokeRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := appmembers.NewRevokeAppMembershipLogic(r.Context(), svcCtx)
		resp, err := l.RevokeAppMembership(&req)
		result.HttpResult(r, w, resp, err)
	}
}
