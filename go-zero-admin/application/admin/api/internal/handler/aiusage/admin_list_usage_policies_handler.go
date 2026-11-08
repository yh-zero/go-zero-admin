// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package aiusage

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/admin/api/internal/logic/aiusage"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
)

func AdminListUsagePoliciesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AiUsageAdminQueryReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := aiusage.NewAdminListUsagePoliciesLogic(r.Context(), svcCtx)
		resp, err := l.AdminListUsagePolicies(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
