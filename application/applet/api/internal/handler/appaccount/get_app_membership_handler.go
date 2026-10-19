// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appaccount

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/appaccount"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/pkg/result"
)

// 读取当前会员权益
func GetAppMembershipHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := appaccount.NewGetAppMembershipLogic(r.Context(), svcCtx)
		resp, err := l.GetAppMembership()
		result.HttpResult(r, w, resp, err)
	}
}
