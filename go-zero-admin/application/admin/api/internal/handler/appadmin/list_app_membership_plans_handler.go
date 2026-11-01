// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appadmin

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/admin/api/internal/logic/appadmin"
	"go-zero-admin/application/admin/api/internal/svc"
)

func ListAppMembershipPlansHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := appadmin.NewListAppMembershipPlansLogic(r.Context(), svcCtx)
		resp, err := l.ListAppMembershipPlans()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
