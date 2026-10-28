// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appmembers

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/appmembers"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/pkg/result"
)

// 读取会员方案
func ListAppPlansHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := appmembers.NewListAppPlansLogic(r.Context(), svcCtx)
		resp, err := l.ListAppPlans()
		result.HttpResult(r, w, resp, err)
	}
}
